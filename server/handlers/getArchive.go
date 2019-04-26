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
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// GetArchive download all file of the upload in a zip archive
func GetArchive(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
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

	// Get files to archive
	var files []*common.File
	for _, file := range upload.Files {
		// Ignore uploading, missing, removed, one shot already downloaded,...
		if file.Status != "uploaded" {
			continue
		}

		// Update metadata if oneShot option is set.
		// Doing this later would increase the window to race the condition.
		// To avoid the race completely AddOrUpdateFile should return the previous version of the metadata
		// and ensure proper locking ( which is the case of bolt and looks doable with mongodb but would break the interface ).
		if upload.OneShot {
			file.Status = "downloaded"
			err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
			if err != nil {
				log.Warningf("Error while deleting file %s from upload %s metadata : %s", file.Name, upload.ID, err)
				continue
			}
		}

		files = append(files, file)
	}

	if len(files) == 0 {
		context.Fail(ctx, req, resp, "Nothing to archive", 404)
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
		log.Warning("Missing file name")
		context.Fail(ctx, req, resp, "Missing file name", 400)
		return
	}

	if !strings.HasSuffix(fileName, ".zip") {
		log.Warningf("Invalid file name %s. Missing .zip extension", fileName)
		context.Fail(ctx, req, resp, fmt.Sprintf("Invalid file name %s. Missing .zip extension", fileName), 400)
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
		// Get file in data backend

		if upload.Stream {
			context.Fail(ctx, req, resp, "Archive feature is not available in stream mode", 404)
			return
		}

		backend := context.GetDataBackend(ctx)

		// The zip archive is piped directly to http response body without buffering
		archive := zip.NewWriter(resp)

		for _, file := range files {
			fileReader, err := backend.GetFile(ctx, upload, file.ID)
			if err != nil {
				log.Warningf("Failed to get file %s in upload %s : %s", file.Name, upload.ID, err)
				context.Fail(ctx, req, resp, fmt.Sprintf("Failed to read file %s", file.Name), 404)
				return
			}

			fileWriter, err := archive.Create(file.Name)
			if err != nil {
				log.Warningf("Failed to add file %s to the archive : %s", file.Name, err)
				context.Fail(ctx, req, resp, fmt.Sprintf("Failed to add file %s to the archive", file.Name), 500)
				return
			}

			// File is piped directly to zip archive thus to the http response body without buffering
			_, err = io.Copy(fileWriter, fileReader)
			if err != nil {
				log.Warningf("Error while copying file to response : %s", err)
			}

			err = fileReader.Close()
			if err != nil {
				log.Warningf("Error while closing file reader : %s", err)
			}

			// Remove file from data backend if oneShot option is set
			if upload.OneShot {
				err = backend.RemoveFile(ctx, upload, file.ID)
				if err != nil {
					log.Warningf("Error while deleting file %s from upload %s : %s", file.Name, upload.ID, err)
					return
				}
			}
		}

		err := archive.Close()
		if err != nil {
			log.Warningf("Failed to close zip archive : %s", err)
			return
		}

		// Remove upload if no files anymore
		RemoveUploadIfNoFileAvailable(ctx, upload)
	}
}
