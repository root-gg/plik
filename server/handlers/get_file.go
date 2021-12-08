package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
)

// GetFile download a file
func GetFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()

	if !checkDownloadDomain(ctx) {
		return
	}

	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
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
		}
	}

	// Avoid rendering HTML in browser
	if strings.Contains(file.Type, "html") {
		file.Type = "text/plain"
	}

	// Force the download of the following types as they are blocked by the CSP Header and won't display properly.
	if file.Type == "" || strings.Contains(file.Type, "flash") || strings.Contains(file.Type, "pdf") {
		file.Type = "application/octet-stream"
	}

	// Set content type and print file
	resp.Header().Set("Content-Type", file.Type)

	/* Additional security headers for possibly unsafe content */
	if ctx.GetConfig().EnhancedWebSecurity {
		resp.Header().Set("X-Content-Type-Options", "nosniff")
		resp.Header().Set("X-XSS-Protection", "1; mode=block")
		resp.Header().Set("X-Frame-Options", "DENY")
		resp.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; style-src 'none'; img-src 'none'; connect-src 'none'; font-src 'none'; object-src 'none'; media-src 'self'; child-src 'none'; form-action 'none'; frame-ancestors 'none'; plugin-types; sandbox")
	}

	/* Additional header for disabling cache if the upload is OneShot */
	if upload.OneShot || upload.Stream { // If this is a one shot or stream upload we have to ensure it's downloaded only once.
		resp.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
		resp.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
		resp.Header().Set("Expires", "0")                                         // Proxies
	}

	if file.Size > 0 {
		resp.Header().Set("Content-Length", strconv.Itoa(int(file.Size)))
	}

	// If "dl" GET params is set
	// -> Set Content-Disposition header
	// -> The client should download file instead of displaying it
	dl := req.URL.Query().Get("dl")
	if dl != "" {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`attachement; filename="%s"`, file.Name))
	} else {
		resp.Header().Set("Content-Disposition", fmt.Sprintf(`filename="%s"`, file.Name))
	}

	// HEAD Request => Do not print file, user just wants http headers
	// GET  Request => Print file content
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

		// File is piped directly to http response body without buffering
		_, err = io.Copy(resp, fileReader)
		if err != nil {
			log.Warningf("error while copying file to response : %s", err)
		}
	}
}
