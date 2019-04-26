/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package handlers

import (
	"fmt"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// GetFile download a file
func GetFile(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := context.GetLogger(ctx)
	config := context.GetConfig(ctx)

	// If a download domain is specified verify that the request comes from this specific domain
	if config.GetDownloadDomain() != nil {
		if req.Host != config.GetDownloadDomain().Host {
			downloadURL := fmt.Sprintf("%s://%s%s",
				config.GetDownloadDomain().Scheme,
				config.GetDownloadDomain().Host,
				req.RequestURI)
			log.Warningf("Invalid download domain %s, expected %s", req.Host, config.GetDownloadDomain().Host)
			http.Redirect(resp, req, downloadURL, 301)
			return
		}
	}

	// Get upload from context
	upload := context.GetUpload(ctx)
	if upload == nil {
		// This should never append
		log.Critical("Missing upload in getFileHandler")
		context.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// Get file from context
	file := context.GetFile(ctx)
	if file == nil {
		// This should never append
		log.Critical("Missing file in getFileHandler")
		context.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// If upload has OneShot option, test if file has not been already downloaded once
	if upload.OneShot && file.Status == "downloaded" {
		log.Warningf("File %s has already been downloaded", file.Name)
		context.Fail(ctx, req, resp, fmt.Sprintf("File %s has already been downloaded", file.Name), 404)
		return
	}

	// If the file is marked as deleted by a previous call, we abort request
	if file.Status == "removed" {
		log.Warningf("File %s has been removed", file.Name)
		context.Fail(ctx, req, resp, fmt.Sprintf("File %s has been removed", file.Name), 404)
		return
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
	resp.Header().Set("X-Content-Type-Options", "nosniff")
	resp.Header().Set("X-XSS-Protection", "1; mode=block")
	resp.Header().Set("X-Frame-Options", "DENY")
	resp.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'none'; style-src 'none'; img-src 'none'; connect-src 'none'; font-src 'none'; object-src 'none'; media-src 'self'; child-src 'none'; form-action 'none'; frame-ancestors 'none'; plugin-types; sandbox")

	/* Additional header for disabling cache if the upload is OneShot */
	if upload.OneShot {
		resp.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1
		resp.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0
		resp.Header().Set("Expires", "0")                                         // Proxies
	}

	if file.CurrentSize > 0 {
		resp.Header().Set("Content-Length", strconv.Itoa(int(file.CurrentSize)))
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
			backend = context.GetStreamBackend(ctx)
		} else {
			backend = context.GetDataBackend(ctx)
		}

		fileReader, err := backend.GetFile(ctx, upload, file.ID)
		if err != nil {
			log.Warningf("Failed to get file %s in upload %s : %s", file.Name, upload.ID, err)
			context.Fail(ctx, req, resp, fmt.Sprintf("Failed to read file %s", file.Name), 404)
			return
		}
		defer fileReader.Close()

		// Update metadata if oneShot option is set
		// There is a small possible race from upload.OneShot && file.Status == "downloaded" to here.
		// To avoid the race completely AddOrUpdateFile should return the previous version of the metadata
		// and ensure proper locking ( which is the case of bolt and looks doable with mongodb but would break the interface ).
		if upload.OneShot {
			file.Status = "downloaded"
			err = context.GetMetadataBackend(ctx).Upsert(ctx, upload)
			if err != nil {
				log.Warningf("Error while deleting file %s from upload %s metadata : %s", file.Name, upload.ID, err)
			}
		}

		// File is piped directly to http response body without buffering
		_, err = io.Copy(resp, fileReader)
		if err != nil {
			log.Warningf("Error while copying file to response : %s", err)
		}

		// Remove file from data backend if oneShot option is set
		if upload.OneShot && !upload.Stream {
			err = backend.RemoveFile(ctx, upload, file.ID)
			if err != nil {
				log.Warningf("Error while deleting file %s from upload %s : %s", file.Name, upload.ID, err)
				return
			}
		}

		// Remove upload if no files anymore
		RemoveUploadIfNoFileAvailable(ctx, upload)
	}
}
