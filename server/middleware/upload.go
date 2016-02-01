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
	"strings"
	"time"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
)

// Upload retrieve the requested upload metadata from the metadataBackend and save it to the request context.
func Upload(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := common.GetLogger(ctx)
		log.Debug("Upload handler")

		// Get the upload id from the url params
		vars := mux.Vars(req)
		uploadID := vars["uploadID"]
		if uploadID == "" {
			log.Warning("Missing upload id")
			common.Fail(ctx, req, resp, "Missing upload id", 400)
			return
		}

		// Get upload metadata
		upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, uploadID)
		if err != nil {
			log.Warningf("Upload not found : %s", err)
			common.Fail(ctx, req, resp, fmt.Sprintf("Upload %s not found", uploadID), 404)
			return
		}

		// Update request logger prefix
		prefix := fmt.Sprintf("%s[%s]", log.Prefix, uploadID)
		log.SetPrefix(prefix)

		// Test if upload is not expired
		if upload.IsExpired() {
			log.Warningf("Upload is expired since %s", time.Since(time.Unix(upload.Creation, int64(0)).Add(time.Duration(upload.TTL)*time.Second)).String())
			common.Fail(ctx, req, resp, fmt.Sprintf("Upload %s has expired", uploadID), 404)
			return
		}

		// Save upload in the request context
		ctx.Set("upload", upload)

		forbidden := func() {
			resp.Header().Set("WWW-Authenticate", "Basic realm=\"plik\"")
			common.Fail(ctx, req, resp, "Please provide valid credentials to access this upload", 401)
		}

		// Handle basic auth if upload is password protected
		if upload.ProtectedByPassword {
			if req.Header.Get("Authorization") == "" {
				log.Warning("Missing Authorization header")
				forbidden()
				return
			}

			// Basic auth Authorization header must be set to
			// "Basic base64("login:password")". Only the md5sum
			// of the base64 string is saved in the upload metadata
			auth := strings.Split(req.Header.Get("Authorization"), " ")
			if len(auth) != 2 {
				log.Warningf("Inavlid Authorization header %s", req.Header.Get("Authorization"))
				forbidden()
				return
			}
			if auth[0] != "Basic" {
				log.Warningf("Inavlid http authorization scheme : %s", auth[0])
				forbidden()
				return
			}
			var md5sum string
			md5sum, err = utils.Md5sum(auth[1])
			if err != nil {
				log.Warningf("Unable to hash credentials : %s", err)
				forbidden()
				return
			}
			if md5sum != upload.Password {
				log.Warning("Invalid credentials")
				forbidden()
				return
			}
		}

		// Check upload token
		uploadToken := req.Header.Get("X-UploadToken")
		if uploadToken != "" && uploadToken == upload.UploadToken {
			upload.IsAdmin = true
		} else {
			// Check if upload belongs to user
			if common.Config.Authentication && upload.User != "" {
				user := common.GetUser(ctx)
				if user != nil && user.ID == upload.User {
					upload.IsAdmin = true
				}
			}
		}

		next.ServeHTTP(resp, req)
	})
}
