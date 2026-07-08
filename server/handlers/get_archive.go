package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/root-gg/logger"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// recordArchiveDownloadBestEffort records the egress (bytes served) and, when
// countEvent is true, the download event of an archive download. It runs after
// the archive stream has been written so bytes reflects the bytes actually sent
// to the client (partial on a mid-stream failure); countEvent is false for a
// bytes-only partial recording. Stats availability must not block a valid
// download, so failures are logged and swallowed — mirroring
// recordFileDownloadBestEffort on the single-file path.
func recordArchiveDownloadBestEffort(ctx *context.Context, upload *common.Upload, files []*common.File, bytes int64, countEvent bool) {
	if err := ctx.GetMetadataBackend().RecordArchiveDownload(upload, files, bytes, countEvent); err != nil {
		ctx.GetLogger().Warningf("unable to record archive download statistics : %s", err)
	}
}

// recordPartialArchiveBytes records the bytes already streamed to the client
// as a bytes-only archive download (no event) when the handler is about to
// abort on an internal error before the zip stream could complete. This
// mirrors the mid-stream archiveFailed policy: an earlier
// file in the same archive may have already fully streamed, and that must not
// be silently dropped just because a later file's internal error (GetFile /
// CreateHeader) aborts the whole response.
//
// anyFileStreamed must be true only once at least one earlier file's content
// has been fully written AND its reader closed. archive/zip buffers each
// entry's compressed output internally and only emits it to the underlying
// writer (hence to cw) when the NEXT entry starts or the archive closes, so
// without an explicit Close here cw.written would still read 0 even though a
// whole prior file is sitting flushable in that internal buffer. Closing is
// skipped when anyFileStreamed is false (the failure hit the very first file)
// so this stays a true no-op in the common case, matching prior behavior and
// not manufacturing a bogus empty-zip trailer as spurious "egress".
func recordPartialArchiveBytes(ctx *context.Context, log *logger.Logger, upload *common.Upload, files []*common.File, archive *zip.Writer, cw *countingResponseWriter, anyFileStreamed bool) {
	if !anyFileStreamed {
		return
	}
	if err := archive.Close(); err != nil {
		log.Warningf("error while closing zip archive : %s", err)
	}
	if cw.written <= 0 {
		return
	}
	recordArchiveDownloadBestEffort(ctx, upload, files, cw.written, false)
}

// GetArchive download all file of the upload in a zip archive
func GetArchive(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()

	if !checkDownloadDomain(ctx) {
		return
	}

	// Set CORS headers for cross-origin fetch
	setCORSHeaders(ctx, resp, req)

	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
	}

	if upload.Stream {
		ctx.BadRequest("archive feature is not available in stream mode")
		return
	}

	// Set content type
	resp.Header().Set("Content-Type", "application/zip")

	/* Security headers — always set */
	resp.Header().Set("X-Content-Type-Options", "nosniff")
	resp.Header().Set("X-Frame-Options", "DENY")
	resp.Header().Set("Content-Security-Policy", "default-src 'none'; form-action 'none'; frame-ancestors 'none'; sandbox")

	/* Additional header for disabling cache if the upload is OneShot */
	if upload.OneShot {
		resp.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
		resp.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
		resp.Header().Set("Expires", "0")                                         // Proxies
	}

	// Get the file name from the url params
	vars := mux.Vars(req)
	fileName := vars["filename"]
	if fileName == "" {
		ctx.MissingParameter("archive name")
		return
	}

	if len(fileName) > 1024 {
		ctx.InvalidParameter("archive name too long, maximum 1024 characters")
		return
	}

	if !strings.HasSuffix(fileName, ".zip") {
		ctx.InvalidParameter("archive name, missing .zip extension")
		return
	}

	// If "dl" GET params is set
	// -> Set Content-Disposition header
	// -> The client should download file instead of displaying it
	dl := req.URL.Query().Get("dl")
	if dl != "" {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, common.SanitizeFilenameForDisposition(fileName)))
	} else {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, common.SanitizeFilenameForDisposition(fileName)))
	}

	// HEAD Request => Do not print file, user just wants http headers
	// GET  Request => Print file content
	if req.Method == "GET" {
		// Get files to archive

		var files []*common.File
		f := func(file *common.File) error {
			// Ignore uploading, missing, removed, one shot already downloaded,...
			if file.Status != common.FileUploaded {
				return nil
			}

			if upload.OneShot {
				// Update file status
				err := ctx.GetMetadataBackend().UpdateFileStatus(file, file.Status, common.FileRemoved)
				if err != nil {
					return fmt.Errorf("unable to update file status : %s", err)
				}
			}

			files = append(files, file)

			return nil
		}

		err := ctx.GetMetadataBackend().ForEachUploadFiles(upload.ID, f)
		if err != nil {
			ctx.InternalServerError("unable to update file status", err)
			return
		}

		if len(files) == 0 {
			ctx.BadRequest("nothing to archive")
			return
		}

		backend := ctx.GetDataBackend()

		// The zip archive is piped directly to http response body without
		// buffering. Wrap the writer so we can record the egress (bytes served)
		// once the archive stream closes successfully.
		cw := newCountingResponseWriter(resp)
		archive := zip.NewWriter(cw)
		archiveFailed := false
		// anyFileStreamed tracks whether an earlier file's content has already
		// been fully written and its reader closed, so a later internal error
		// (GetFile / CreateHeader) knows there is real prior egress that must
		// not be dropped. See recordPartialArchiveBytes.
		anyFileStreamed := false

		for _, file := range files {
			fileReader, err := backend.GetFile(file)
			if err != nil {
				recordPartialArchiveBytes(ctx, log, upload, files, archive, cw, anyFileStreamed)
				ctx.InternalServerError("unable to get file from data backend", err)
				return
			}

			method := zip.Deflate // Default: compression enabled
			if !ctx.GetConfig().EnableArchiveCompression {
				method = zip.Store // Disable compression to prevent CPU exhaustion
			}

			header := &zip.FileHeader{
				Name:   file.Name,
				Method: method,
			}

			fileWriter, err := archive.CreateHeader(header)
			if err != nil {
				recordPartialArchiveBytes(ctx, log, upload, files, archive, cw, anyFileStreamed)
				ctx.InternalServerError("error while creating zip archive", err)
				return
			}

			// File is piped directly to zip archive thus to the http response body without buffering
			_, err = io.Copy(fileWriter, fileReader)
			if err != nil {
				log.Warningf("error while copying zip archive to response body : %s", err)
				archiveFailed = true
			}

			err = fileReader.Close()
			if err != nil {
				log.Warningf("error while closing zip archive reader : %s", err)
				archiveFailed = true
			}

			if archiveFailed {
				break
			}
			anyFileStreamed = true
		}

		err = archive.Close()
		if err != nil {
			log.Warningf("error while closing zip archive : %s", err)
			archiveFailed = true
		}

		// On success this records the full archive download event. On a
		// mid-stream failure (archiveFailed) the bytes already streamed to the
		// client before the failure are still recorded, as a bytes-only event
		// (downloads = 0, countEvent = false) — mirroring the single-file
		// download path's "a client that disconnects mid-stream records exactly
		// its egress" policy. See RecordArchiveDownload's doc comment and the
		// egress paragraph in server/ARCHITECTURE.md.
		recordArchiveDownloadBestEffort(ctx, upload, files, cw.written, !archiveFailed)
	}
}
