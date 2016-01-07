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
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend"
	"github.com/root-gg/plik/server/metadataBackend"
)

// CreateTokenHandler create a new auth token
func CreateTokenHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.TokenAuthentication {
		log.Warning("Token authentication is not enabled")
		common.Fail(ctx, req, resp, "Token authentication is not enabled", 403)
		return
	}

	if !common.IsWhitelisted(ctx) {
		log.Warning("Unable to create a token from an untrusted IP")
		common.Fail(ctx, req, resp, "Unauthorized source IP address", 403)
		return
	}

	// Create token
	token := common.NewToken()

	// Read request body
	defer req.Body.Close()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warningf("Unable to read request body : %s", err)
		common.Fail(ctx, req, resp, "Unable to read request body", 403)
		return
	}

	// Deserialize json body
	if len(body) > 0 {
		err = json.Unmarshal(body, token)
		if err != nil {
			log.Warningf("Unable to deserialize json request body : %s", err)
			common.Fail(ctx, req, resp, "Unable to deserialize json request body", 400)
			return
		}
	}

	// Initialize token
	token.Create()
	token.SourceIP = common.GetSourceIP(ctx).String()

	// Save token
	err = metadataBackend.GetMetaDataBackend().SaveToken(ctx, token)
	if err != nil {
		log.Warningf("Unable to create token : %s", err)
		common.Fail(ctx, req, resp, "Unable to create token", 500)
		return
	}

	// Print token in the json response.
	var json []byte
	if json, err = utils.ToJson(token); err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}
	resp.Write(json)
}

// GetTokenHandler return token's metadata
func GetTokenHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.TokenAuthentication {
		log.Warning("Token authentication is not enabled")
		common.Fail(ctx, req, resp, "Token authentication is not enabled", 403)
		return
	}

	// Get token from the url params
	vars := mux.Vars(req)
	token := vars["token"]

	// Get token
	t, err := metadataBackend.GetMetaDataBackend().GetToken(ctx, token)
	if err != nil {
		log.Warningf("Unable to get token %s : %s", token, err)
		common.Fail(ctx, req, resp, "Unable to get token", 404)
		return
	}

	// Print token in the json response.
	var json []byte
	if json, err = utils.ToJson(t); err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}
	resp.Write(json)
}

// RevokeTokenHandler revoke an existing token
func RevokeTokenHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.TokenAuthentication {
		log.Warning("Token authentication is not enabled")
		common.Fail(ctx, req, resp, "Token authentication is not enabled", 403)
		return
	}

	// Get token from the url params
	vars := mux.Vars(req)
	token := vars["token"]

	// Revoke token
	err := metadataBackend.GetMetaDataBackend().RevokeToken(ctx, token)
	if err != nil {
		log.Warningf("Unable to revoke token %s : %s", token, err)
		common.Fail(ctx, req, resp, "Unable to revoke token", 500)
		return
	}

	// Remove uploads
	params := req.URL.Query()
	if _, ok := params["removeUploads"]; ok {
		// Get uploads
		ids, err := metadataBackend.GetMetaDataBackend().GetUploadsWithToken(ctx, token)
		if err != nil {
			log.Warningf("Unable to revoke token %s : %s", token, err)
			common.Fail(ctx, req, resp, "Unable to revoke token", 500)
			return
		}

		// Remove uploads
		for _, id := range ids {
			upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, id)
			if err != nil {
				log.Warningf("Unable to get upload %s : %s", id, err)
				continue
			}

			// Remove from data backend
			err = dataBackend.GetDataBackend().RemoveUpload(ctx, upload)
			if err != nil {
				log.Warningf("Unable to remove upload data : %s", err)
				continue
			}

			// Remove from metadata backend
			err = metadataBackend.GetMetaDataBackend().Remove(ctx, upload)
			if err != nil {
				log.Warningf("Unable to remove upload metadata %s : %s", id, err)
				continue
			}
		}
	}
}

// GetUploadsWithTokenHandler list return upload belonging to a specific auth token
func GetUploadsWithTokenHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.TokenAuthentication {
		log.Warning("Token authentication is not enabled")
		common.Fail(ctx, req, resp, "Token authentication is not enabled", 403)
		return
	}

	// Get token from the url params
	vars := mux.Vars(req)
	token := vars["token"]

	// Get uploads
	ids, err := metadataBackend.GetMetaDataBackend().GetUploadsWithToken(ctx, token)
	if err != nil {
		log.Warningf("Unable to revoke token %s : %s", token, err)
		common.Fail(ctx, req, resp, "Unable to revoke token", 500)
		return
	}

	// Fix golint warning
	var uploads []*common.Upload
	uploads = make([]*common.Upload, 0)

	for _, id := range ids {
		upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, id)
		if err != nil {
			log.Warningf("Unable to get upload %s : %s", id, err)
			continue
		}

		if !upload.IsExpired() {
			upload.Sanitize()
			uploads = append(uploads, upload)
		}
	}

	// Print uploads in the json response.
	var json []byte
	if json, err = utils.ToJson(uploads); err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}
	resp.Write(json)
}

// RemoveUploadsWithTokenHandler remove all uploads belonging to a specific auth token
func RemoveUploadsWithTokenHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	if !common.Config.TokenAuthentication {
		log.Warning("Token authentication is not enabled")
		common.Fail(ctx, req, resp, "Token authentication is not enabled", 403)
		return
	}

	// Get token from the url params
	vars := mux.Vars(req)
	token := vars["token"]

	// Get uploads
	ids, err := metadataBackend.GetMetaDataBackend().GetUploadsWithToken(ctx, token)
	if err != nil {
		log.Warningf("Unable to revoke token %s : %s", token, err)
		common.Fail(ctx, req, resp, "Unable to revoke token", 500)
		return
	}

	// Remove uploads
	for _, id := range ids {
		upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, id)
		if err != nil {
			log.Warningf("Unable to get upload %s : %s", id, err)
			continue
		}

		// Remove from data backend
		err = dataBackend.GetDataBackend().RemoveUpload(ctx, upload)
		if err != nil {
			log.Warningf("Unable to remove upload data : %s", err)
			continue
		}

		// Remove from metadata backend
		err = metadataBackend.GetMetaDataBackend().Remove(ctx, upload)
		if err != nil {
			log.Warningf("Unable to remove upload metadata %s : %s", id, err)
			continue
		}
	}
}
