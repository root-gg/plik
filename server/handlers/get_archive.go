package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// GetArchive download all file of the upload in a zip archive
func GetArchive(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()

	if !checkDownloadDomain(ctx) {
		return
	}

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

	/* Additional security headers for possibly unsafe content */
	resp.Header().Set("X-Content-Type-Options", "nosniff")
	resp.Header().Set("X-XSS-Protection", "1; mode=block")
	resp.Header().Set("X-Frame-Options", "DENY")
	resp.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; style-src 'none'; img-src 'none'; connect-src 'none'; font-src 'none'; object-src 'none'; media-src 'none'; child-src 'none'; form-action 'none'; frame-ancestors 'none'; plugin-types ''; sandbox ''")

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
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachement; filename="%s"`, fileName))
	} else {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, fileName))
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
		}

		if len(files) == 0 {
			ctx.BadRequest("nothing to archive")
			return
		}

		backend := ctx.GetDataBackend()

		// The zip archive is piped directly to http response body without buffering
		archive := zip.NewWriter(resp)

		for _, file := range files {
			fileReader, err := backend.GetFile(file)
			if err != nil {
				ctx.InternalServerError("unable to get file from data backend", err)
				return
			}

			fileWriter, err := archive.Create(file.Name)
			if err != nil {
				ctx.InternalServerError("error while creating zip archive", err)
				return
			}

			// File is piped directly to zip archive thus to the http response body without buffering
			_, err = io.Copy(fileWriter, fileReader)
			if err != nil {
				log.Warningf("error while copying zip archive to response body : %s", err)
			}

			err = fileReader.Close()
			if err != nil {
				log.Warningf("error while closing zip archive reader : %s", err)
			}
		}

		err = archive.Close()
		if err != nil {
			log.Warningf("error while closing zip archive : %s", err)
			return
		}
	}
}
