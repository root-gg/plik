package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
)

// parsedRange is one validated entry of a "bytes=" Range header. hasStart/hasEnd
// distinguish the three RFC 7233 forms without clamping to a file size yet:
// "start-end" (both set), "start-" (open-ended, hasEnd false), and the suffix
// "-N" (hasStart false, end carries N).
type parsedRange struct {
	hasStart bool
	start    int64
	hasEnd   bool
	end      int64
}

// parseByteRanges tokenizes and validates a "bytes=" Range header into its
// individual ranges. It returns ok=false for a non-bytes unit or any
// syntactically invalid range (unparseable/negative bound, empty spec, or an end
// before its start), matching the subset http.ServeContent accepts before it
// streams content. It is the single parser shared by the download-counting policy
// and the multi-range egress cap so the two can never disagree on what a header
// means.
func parseByteRanges(rangeHeader string) (ranges []parsedRange, ok bool) {
	rangeHeader = strings.TrimSpace(rangeHeader)
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, false
	}

	for byteRange := range strings.SplitSeq(strings.TrimPrefix(rangeHeader, "bytes="), ",") {
		parts := strings.SplitN(strings.TrimSpace(byteRange), "-", 2)
		if len(parts) != 2 {
			return nil, false
		}

		start := strings.TrimSpace(parts[0])
		end := strings.TrimSpace(parts[1])
		if start == "" && end == "" {
			return nil, false
		}

		var r parsedRange
		if start != "" {
			value, err := strconv.ParseInt(start, 10, 64)
			if err != nil || value < 0 {
				return nil, false
			}
			r.hasStart = true
			r.start = value
		}
		if end != "" {
			value, err := strconv.ParseInt(end, 10, 64)
			if err != nil || value < 0 {
				return nil, false
			}
			if r.hasStart && value < r.start {
				return nil, false
			}
			r.hasEnd = true
			r.end = value
		}
		ranges = append(ranges, r)
	}

	return ranges, true
}

// shouldRecordFileDownload applies the download counting policy before serving
// file content. Full GET requests count, and Range GET requests count only when
// the first requested byte is 0 so media players do not inflate trending stats
// with follow-up range fetches.
func shouldRecordFileDownload(req *http.Request) bool {
	if req.Method != http.MethodGet {
		return false
	}

	rangeHeader := strings.TrimSpace(req.Header.Get("Range"))
	if rangeHeader == "" {
		return true
	}

	ranges, ok := parseByteRanges(rangeHeader)
	if !ok || len(ranges) == 0 {
		return false
	}

	first := ranges[0]
	return first.hasStart && first.start == 0
}

// cappedDownloadBytes caps a multi-range GET's recorded egress at the sum of its
// requested ranges, each clamped to the file size. http.ServeContent answers a
// multi-range request with a multipart/byteranges body whose MIME boundaries and
// per-part headers flow through the counting writer but are not file egress; a
// single-range or full-file response streams only file bytes, so its count is
// already exact and is returned unchanged. Capping at the summed clamped range
// lengths (not merely at the file size, which a multi-range sum can fall well
// under) keeps download egress consistent with the single-range/full-file paths
// and with the upload side, which also excludes multipart framing (see
// server/ARCHITECTURE.md). Capping never inflates: an unparseable header, a
// single range, or an already-smaller written count is returned as-is.
func cappedDownloadBytes(req *http.Request, written int64, size int64) int64 {
	rangeHeader := strings.TrimSpace(req.Header.Get("Range"))
	if !strings.Contains(rangeHeader, ",") {
		return written
	}

	ranges, ok := parseByteRanges(rangeHeader)
	if !ok {
		return written
	}

	var sum int64
	for _, r := range ranges {
		var first, last int64
		switch {
		case !r.hasStart:
			// Suffix range "-N": the last N bytes, clamped to the file size.
			suffix := min(r.end, size)
			first = size - suffix
			last = size - 1
		case !r.hasEnd:
			first = r.start
			last = size - 1
		default:
			first = r.start
			last = min(r.end, size-1)
		}
		if first < 0 {
			first = 0
		}
		if last < first {
			// Range does not intersect the file; it contributes no body bytes.
			continue
		}
		sum += last - first + 1
	}

	if sum < written {
		return sum
	}
	return written
}

// recordFileDownloadBestEffort records the egress (bytes served) and, when
// countEvent is true, the download event of a file download. It runs after the
// response has been streamed so bytes reflects the bytes actually written to the
// client (partial on disconnect); countEvent carries the pre-stream
// download-counting policy decision. Stats availability must not block a valid
// download, so failures are logged and swallowed.
func recordFileDownloadBestEffort(ctx *context.Context, upload *common.Upload, file *common.File, bytes int64, countEvent bool) {
	err := ctx.GetMetadataBackend().RecordFileDownload(upload, file, bytes, countEvent)
	if err != nil {
		ctx.GetLogger().Warningf("unable to record download statistics : %s", err)
	}
}

// GetFile download a file
func GetFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()

	if !checkDownloadDomain(ctx) {
		return
	}

	// Set CORS headers for cross-origin file viewer / E2EE decrypt fetch
	setCORSHeaders(ctx, resp, req)

	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
	}

	// For E2EE uploads, redirect the webapp to the download page
	// so decryption can happen client-side
	if upload.E2EE != "" && common.IsPlikWebapp(req) {
		config := ctx.GetConfig()
		redirectURL := fmt.Sprintf("%s/#/?id=%s", config.Path, upload.ID)
		http.Redirect(resp, req, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Get file from context
	file := ctx.GetFile()
	if file == nil {
		panic("missing file from context")
	}

	// File status check
	if upload.Stream {
		if file.Status != common.FileUploading {
			ctx.NotFound("file %s (%s) is not available : %s", file.Name, file.ID, file.Status)
			return
		}
	} else {
		if file.Status != common.FileUploaded {
			ctx.NotFound("file %s (%s) is not available : %s", file.Name, file.ID, file.Status)
			return
		}
	}

	if req.Method == "GET" && upload.OneShot {
		// Update file status
		// For streaming upload the status is set to deleted by the add_file handler
		err := ctx.GetMetadataBackend().UpdateFileStatus(file, file.Status, common.FileRemoved)
		if err != nil {
			ctx.InternalServerError("unable to update file status", err)
			return
		}
	}

	// Neutralize content types that could execute code in the browser
	// Force download as binary to prevent XSS via inline scripts, SVG onload handlers, etc.
	if file.Type == "" ||
		strings.Contains(file.Type, "html") ||
		strings.Contains(file.Type, "svg") ||
		strings.Contains(file.Type, "xml") ||
		strings.Contains(file.Type, "javascript") ||
		strings.Contains(file.Type, "flash") ||
		strings.Contains(file.Type, "pdf") {
		file.Type = "application/octet-stream"
	}

	// For E2EE uploads, always serve as binary data — content-type detection
	// on encrypted bytes is meaningless
	if upload.E2EE != "" {
		file.Type = "application/octet-stream"
	}

	// Set content type and print file
	resp.Header().Set("Content-Type", file.Type)

	/* Security headers — always set */
	resp.Header().Set("X-Content-Type-Options", "nosniff")
	resp.Header().Set("X-Frame-Options", "DENY")
	resp.Header().Set("Content-Security-Policy", "default-src 'none'; media-src 'self'; form-action 'none'; frame-ancestors 'none'; sandbox")

	/* Additional header for disabling cache if the upload is OneShot */
	if upload.OneShot || upload.Stream { // If this is a one shot or stream upload we have to ensure it's downloaded only once.
		resp.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
		resp.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
		resp.Header().Set("Expires", "0")                                         // Proxies
	}

	// If "dl" GET params is set
	// -> Set Content-Disposition header
	// -> The client should download file instead of displaying it
	dl := req.URL.Query().Get("dl")
	if dl != "" {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, common.SanitizeFilenameForDisposition(file.Name)))
	} else {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, common.SanitizeFilenameForDisposition(file.Name)))
	}

	// HEAD Request => Do not print file, user just wants http headers
	// GET  Request => Print file content
	if !upload.Stream && !upload.OneShot {
		backend := ctx.GetDataBackend()
		fileReader, err := backend.GetFile(file)
		if err != nil {
			ctx.InternalServerError("unable to get file from data backend", err)
			return
		}
		defer func() { _ = fileReader.Close() }()

		// The event decision is made from the request before streaming (byte-0
		// range policy), but recording happens after so we also capture the bytes
		// actually served. A mid-range GET streams bytes without counting an
		// event; a HEAD serves no body and records nothing.
		countEvent := shouldRecordFileDownload(req)
		cw := newCountingResponseWriter(resp)

		http.ServeContent(cw, req, file.Name, time.Time{}, fileReader)

		// A multi-range request is answered with a multipart/byteranges body whose
		// framing the counting writer also tallies; cap the recorded egress at the
		// summed clamped range lengths so only file bytes are counted.
		recordFileDownloadBestEffort(ctx, upload, file, cappedDownloadBytes(req, cw.written, file.Size), countEvent)
	} else {
		// Set content length otherwise handled by http.ServeContent
		if file.Size > 0 && !upload.Stream {
			resp.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
		}

		if req.Method == "GET" {
			// Get file in data backend
			var backend data.Backend
			if upload.Stream {
				backend = ctx.GetStreamBackend()
			} else {
				backend = ctx.GetDataBackend()
			}

			fileReader, err := backend.GetFile(file)
			if err != nil {
				ctx.InternalServerError("unable to get file from data backend", err)
				return
			}
			defer func() { _ = fileReader.Close() }()

			// This branch ignores Range headers and always streams the full
			// file, so every GET is a complete download — applying the
			// byte-0 range policy here would let a Range header evade
			// counting a one-shot/stream file that was fully consumed. The
			// event is therefore always counted; recording
			// happens after streaming so the bytes served are captured too.
			cw := newCountingResponseWriter(resp)

			// File is piped directly to http response body without buffering
			_, err = io.Copy(cw, fileReader)
			if err != nil {
				log.Warningf("error while copying file to response : %s", err)
			}

			recordFileDownloadBestEffort(ctx, upload, file, cw.written, true)
		}
	}

	if file.Status == common.FileRemoved {
		// Remove the file asynchronously
		err := purge(ctx, file)
		if err != nil {
			log.Warningf("%s", err.Error())
		}
	}
}
