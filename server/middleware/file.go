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

package middleware

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/context"
)

// File retrieve the requested file metadata from the metadataBackend and save it in the request context.
func File(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := context.GetLogger(ctx)

		// Get upload from context
		upload := context.GetUpload(ctx)
		if upload == nil {
			// This should never append
			log.Critical("Missing upload in file middleware")
			context.Fail(ctx, req, resp, "Internal error", 500)
			return
		}

		// Get the file id from the url params
		vars := mux.Vars(req)
		fileID := vars["fileID"]
		if fileID == "" {
			log.Warning("Missing file id")
			context.Fail(ctx, req, resp, "Missing file id", 400)
			return
		}

		// Get the file name from the url params
		fileName := vars["filename"]
		if fileName == "" {
			log.Warning("Missing file name")
			context.Fail(ctx, req, resp, "Missing file name", 400)
			return
		}

		// Get file object in upload metadata
		file, ok := upload.Files[fileID]
		if !ok {
			log.Warningf("File %s not found", fileID)
			context.Fail(ctx, req, resp, fmt.Sprintf("File %s not found", fileID), 404)
			return
		}

		// Compare url filename with upload filename
		if file.Name != fileName {
			log.Warningf("Invalid filename %s mismatch %s", fileName, file.Name)
			context.Fail(ctx, req, resp, fmt.Sprintf("File %s not found", fileName), 404)
			return
		}

		// Save file in the request context
		ctx.Set("file", file)

		next.ServeHTTP(resp, req)
	})
}
