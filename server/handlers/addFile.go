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
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend"
	"github.com/root-gg/plik/server/metadataBackend"
)

// AddFile add a file to an existing upload.
func AddFile(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	user := common.GetUser(ctx)
	if user == nil && !common.IsWhitelisted(ctx) {
		log.Warning("Unable to add file from untrusted source IP address")
		common.Fail(ctx, req, resp, "Unable to add file from untrusted source IP address. Please login or use a cli token.", 403)
		return
	}

	// Get upload from context
	upload := common.GetUpload(ctx)
	if upload == nil {
		// This should never append
		log.Critical("Missing upload in AddFileHandler")
		common.Fail(ctx, req, resp, "Internal error", 500)
		return
	}

	// Check authorization
	if !upload.IsAdmin {
		log.Warningf("Unable to add file : unauthorized")
		common.Fail(ctx, req, resp, "You are not allowed to add file to this upload", 403)
		return
	}

	// Get the file id from the url params
	vars := mux.Vars(req)
	fileID := vars["fileID"]

	// Create a new file object
	var newFile *common.File
	if fileID == "" {
		newFile = common.NewFile()
		newFile.Type = "application/octet-stream"
	} else {
		if _, ok := upload.Files[fileID]; ok {
			newFile = upload.Files[fileID]
		} else {
			log.Warningf("Invalid file id %s", fileID)
			common.Fail(ctx, req, resp, "Invalid file id", 404)
			return
		}
	}

	// Update request logger prefix
	prefix := fmt.Sprintf("%s[%s]", log.Prefix, newFile.ID)
	log.SetPrefix(prefix)

	ctx.Set("file", newFile)

	// Get file handle from multipart request
	var file io.Reader
	multiPartReader, err := req.MultipartReader()
	if err != nil {
		log.Warningf("Failed to get file from multipart request : %s", err)
		common.Fail(ctx, req, resp, "Failed to get file from multipart request", 400)
		return
	}

	// Read multipart body until the "file" part
	for {
		part, errPart := multiPartReader.NextPart()
		if errPart == io.EOF {
			break
		}
		if part.FormName() == "file" {
			file = part

			// Check file name length
			if len(part.FileName()) > 1024 {
				log.Warning("File name is too long")
				common.Fail(ctx, req, resp, "File name is too long. Maximum length is 1024 characters", 400)
				return
			}

			newFile.Name = part.FileName()
			break
		}
	}
	if file == nil {
		log.Warning("Missing file from multipart request")
		common.Fail(ctx, req, resp, "Missing file from multipart request", 400)
		return
	}
	if newFile.Name == "" {
		log.Warning("Missing file name from multipart request")
		common.Fail(ctx, req, resp, "Missing file name from multipart request", 400)
		return
	}

	// Update request logger prefix
	prefix = fmt.Sprintf("%s[%s]", log.Prefix, newFile.Name)
	log.SetPrefix(prefix)

	// Pipe file data from the request body to a preprocessing goroutine
	//  - Guess content type
	//  - Compute md5sum
	//  - Limit upload size
	preprocessReader, preprocessWriter := io.Pipe()
	md5Hash := md5.New()
	totalBytes := 0
	go func() {
		for {
			buf := make([]byte, 1024)
			bytesRead, err := file.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Warningf("Unable to read data from request body : %s", err)
				}

				preprocessWriter.Close()
				return
			}

			// Detect the content-type using the 512 first bytes
			if totalBytes == 0 {
				newFile.Type = http.DetectContentType(buf)
			}

			// Increment size
			totalBytes += bytesRead

			// Compute md5sum
			md5Hash.Write(buf[:bytesRead])

			// Check upload max size limit
			if int64(totalBytes) > common.Config.MaxFileSize {
				err = fmt.Errorf("File too big (limit is set to %d bytes)", common.Config.MaxFileSize)
				log.Warning(err.Error())
				preprocessWriter.CloseWithError(err)
				return
			}

			// Pass file data to data backend
			preprocessWriter.Write(buf[:bytesRead])
		}
	}()

	// Save file in the data backend
	var backend dataBackend.DataBackend
	if upload.Stream {
		backend = dataBackend.GetStreamBackend()
	} else {
		backend = dataBackend.GetDataBackend()
	}
	backendDetails, err := backend.AddFile(ctx, upload, newFile, preprocessReader)
	if err != nil {
		log.Warningf("Unable to save file : %s", err)
		common.Fail(ctx, req, resp, "Unable to save file", 500)
		return
	}

	// Fill-in file information
	newFile.CurrentSize = int64(totalBytes)
	if upload.Stream {
		newFile.Status = "downloaded"
	} else {
		newFile.Status = "uploaded"
	}
	newFile.Md5 = fmt.Sprintf("%x", md5Hash.Sum(nil))
	newFile.UploadDate = time.Now().Unix()
	newFile.BackendDetails = backendDetails

	// Update upload metadata
	upload.Files[newFile.ID] = newFile
	err = metadataBackend.GetMetaDataBackend().AddOrUpdateFile(ctx, upload, newFile)
	if err != nil {
		log.Warningf("Unable to update metadata : %s", err)
		common.Fail(ctx, req, resp, "Unable to update upload metadata", 500)
		return
	}

	// Remove all private information (ip, data backend details, ...) before
	// sending metadata back to the client
	newFile.Sanitize()

	// Print file metadata in the json response.
	var json []byte
	if json, err = utils.ToJson(newFile); err == nil {
		resp.Write(json)
	} else {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}
}
