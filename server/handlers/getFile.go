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
	"io"
	"net/http"
	"strconv"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend"
	"github.com/root-gg/plik/server/metadataBackend"
)

// GetFile download a file
func GetFile(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Get upload from context
	upload := common.GetUpload(ctx)
	if upload == nil {
		// This should never append
		log.Critical("Missing upload in getFileHandler")
		common.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// Get file from context
	file := common.GetFile(ctx)
	if file == nil {
		// This should never append
		log.Critical("Missing file in getFileHandler")
		common.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// If upload has OneShot option, test if file has not been already downloaded once
	if upload.OneShot && file.Status == "downloaded" {
		log.Warningf("File %s has already been downloaded", file.Name)
		common.Fail(ctx, req, resp, "File %s has already been downloaded", 404)
		return
	}

	// If the file is marked as deleted by a previous call, we abort request
	if file.Status == "removed" {
		log.Warningf("File %s has been removed", file.Name)
		common.Fail(ctx, req, resp, "File %s has been removed", 404)
		return
	}

	// If upload is yubikey protected, user must send an OTP when he wants to get a file.
	if upload.Yubikey != "" {

		// Error if yubikey is disabled on server, and enabled on upload
		if !common.Config.YubikeyEnabled {
			log.Warningf("Got a Yubikey upload but Yubikey backend is disabled")
			common.Fail(ctx, req, resp, "Yubikey are disabled on this server", 403)
			return
		}

		vars := mux.Vars(req)
		token := vars["yubikey"]
		if token == "" {
			log.Warningf("Missing yubikey token")
			common.Fail(ctx, req, resp, "Invalid yubikey token", 401)
			return
		}
		if len(token) != 44 {
			log.Warningf("Invalid yubikey token : %s", token)
			common.Fail(ctx, req, resp, "Invalid yubikey token", 401)
			return
		}
		if token[:12] != upload.Yubikey {
			log.Warningf("Invalid yubikey device : %s", token)
			common.Fail(ctx, req, resp, "Invalid yubikey token", 401)
			return
		}

		_, isValid, err := common.Config.YubiAuth.Verify(token)
		if err != nil {
			log.Warningf("Failed to validate yubikey token : %s", err)
			common.Fail(ctx, req, resp, "Invalid yubikey token", 500)
			return
		}
		if !isValid {
			log.Warningf("Invalid yubikey token : %s", token)
			common.Fail(ctx, req, resp, "Invalid yubikey token", 401)
			return
		}
	}

	// Set content type and print file
	resp.Header().Set("Content-Type", file.Type)
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
		var backend dataBackend.DataBackend
		if upload.Stream {
			backend = dataBackend.GetStreamBackend()
		} else {
			backend = dataBackend.GetDataBackend()
		}
		fileReader, err := backend.GetFile(ctx, upload, file.ID)
		if err != nil {
			log.Warningf("Failed to get file %s in upload %s : %s", file.Name, upload.ID, err)
			common.Fail(ctx, req, resp, fmt.Sprintf("Failed to read file %s", file.Name), 404)
			return
		}
		defer fileReader.Close()

		// Update metadata if oneShot option is set
		if upload.OneShot {
			file.Status = "downloaded"
			err = metadataBackend.GetMetaDataBackend().AddOrUpdateFile(ctx, upload, file)
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
		if upload.OneShot {
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
