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
	"net/http"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend"
	"github.com/root-gg/plik/server/metadataBackend"
)

// RemoveFile remove a file from an existing upload
func RemoveFile(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Get upload from context
	upload := common.GetUpload(ctx)
	if upload == nil {
		// This should never append
		log.Critical("Missing upload in removeFileHandler")
		common.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// Check authorization
	if !upload.Removable && !upload.IsAdmin {
		log.Warningf("Unable to remove file : unauthorized")
		common.Fail(ctx, req, resp, "You are not allowed to remove file from this upload", 403)
		return
	}

	// Get file from context
	file := common.GetFile(ctx)
	if file == nil {
		// This should never append
		log.Critical("Missing file in removeFileHandler")
		common.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// Check if file is not already removed
	if file.Status == "removed" {
		log.Warning("Can't remove an already removed file")
		common.Fail(ctx, req, resp, fmt.Sprintf("File %s has already been removed", file.Name), 404)
		return
	}

	// Set status to removed, and save metadatas
	file.Status = "removed"
	if err := metadataBackend.GetMetaDataBackend().AddOrUpdateFile(ctx, upload, file); err != nil {
		log.Warningf("Unable to update metadata : %s", err)
		common.Fail(ctx, req, resp, "Unable to update upload metadata", 500)
		return
	}

	// Remove file from data backend
	// Get file in data backend
	var backend dataBackend.DataBackend
	if upload.Stream {
		backend = dataBackend.GetStreamBackend()
	} else {
		backend = dataBackend.GetDataBackend()
	}

	if err := backend.RemoveFile(ctx, upload, file.ID); err != nil {
		log.Warningf("Unable to delete file : %s", err)
		common.Fail(ctx, req, resp, "Unable to delete file", 500)
		return
	}

	// Remove upload if no files anymore
	RemoveUploadIfNoFileAvailable(ctx, upload)

	// Print upload metadata in the json response.
	json, err := utils.ToJson(upload)
	if err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}

	resp.Write(json)
}
