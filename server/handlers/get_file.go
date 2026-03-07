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
		http.ServeContent(resp, req, file.Name, time.Time{}, fileReader)
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

			// File is piped directly to http response body without buffering
			_, err = io.Copy(resp, fileReader)
			if err != nil {
				log.Warningf("error while copying file to response : %s", err)
			} else {
				// Record file_downloaded event on successful download
				event := common.NewEvent(upload.ID, common.EventFileDownloaded)
				event.FileID = file.ID
				event.FileName = file.Name
				event.FileSize = file.Size
				if ctx.GetSourceIP() != nil {
					event.RemoteIP = ctx.GetSourceIP().String()
				}
				if user := ctx.GetUser(); user != nil {
					event.User = user.ID
				}
				if err := ctx.GetMetadataBackend().CreateEvent(event); err != nil {
					log.Warningf("unable to record file_downloaded event: %s", err)
				}
			}
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
