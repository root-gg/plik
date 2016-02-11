/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
	"net/http"
	"strconv"
)

// UserInfo return user information ( name / email / tokens / ... )
func UserInfo(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Get user from context
	user := common.GetUser(ctx)
	if user == nil {
		common.Fail(ctx, req, resp, "Missing user, Please login first", 401)
		return
	}

	// Serialize user to JSON
	// Print token in the json response.
	json, err := utils.ToJson(user)
	if err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}
	resp.Write(json)
}

// DeleteAccount remove a user account
func DeleteAccount(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Get user from context
	user := common.GetUser(ctx)
	if user == nil {
		// This should never append
		common.Fail(ctx, req, resp, "Missing user, Please login first", 401)
		return
	}

	err := metadataBackend.GetMetaDataBackend().RemoveUser(ctx, user)
	if err != nil {
		log.Warningf("Unable to remove user %s : %s", user.ID, err)
		common.Fail(ctx, req, resp, "Unable to remove user", 500)
		return
	}
}

// GetUserUploads get user uploads
func GetUserUploads(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Get user from context
	user := common.GetUser(ctx)
	if user == nil {
		common.Fail(ctx, req, resp, "Missing user, Please login first", 401)
		return
	}

	// Get token from URL query parameter
	var token *common.Token
	tokenStr := req.URL.Query().Get("token")

	if tokenStr != "" {
		for _, t := range user.Tokens {
			if t.Token == tokenStr {
				token = t
				break
			}
		}
		if token == nil {
			log.Warningf("Unable to get uploads for token %s : Invalid token", tokenStr)
			common.Fail(ctx, req, resp, "Unable to get uploads : Invalid token", 400)
			return
		}
	}

	// Get uploads
	ids, err := metadataBackend.GetMetaDataBackend().GetUserUploads(ctx, user, token)
	if err != nil {
		log.Warningf("Unable to get uploads for user %s : %s", user.ID, err)
		common.Fail(ctx, req, resp, "Unable to get uploads", 500)
		return
	}

	// Get size from URL query parameter
	size := 100
	sizeStr := req.URL.Query().Get("size")
	if sizeStr != "" {
		size, err = strconv.Atoi(sizeStr)
		if err != nil || size <= 0 || size > 100 {
			log.Warningf("Invalid size parameter : %s", sizeStr)
			common.Fail(ctx, req, resp, "Invalid size parameter", 400)
			return
		}
	}

	// Get offset from URL query parameter
	offset := 0
	offsetStr := req.URL.Query().Get("offset")
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			log.Warningf("Invalid offset parameter : %s", offsetStr)
			common.Fail(ctx, req, resp, "Invalid offset parameter", 400)
			return
		}
	}

	// Adjust offset
	if offset > len(ids) {
		offset = len(ids)
	}

	// Adjust size
	if offset+size > len(ids) {
		size = len(ids) - offset
	}

	uploads := []*common.Upload{}
	for _, id := range ids[offset : offset+size] {
		upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, id)
		if err != nil {
			log.Warningf("Unable to get upload %s : %s", id, err)
			continue
		}

		if !upload.IsExpired() {
			token := upload.Token
			upload.Sanitize()
			upload.Token = token
			upload.IsAdmin = true
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

// RemoveUserUploads delete all user uploads
func RemoveUserUploads(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Get user from context
	user := common.GetUser(ctx)
	if user == nil {
		common.Fail(ctx, req, resp, "Missing user, Please login first", 401)
		return
	}

	// Get token from URL query parameter
	var token *common.Token
	tokenStr := req.URL.Query().Get("token")
	if tokenStr != "" {
		for _, t := range user.Tokens {
			if t.Token == tokenStr {
				token = t
			}
		}
	}

	// Get uploads
	ids, err := metadataBackend.GetMetaDataBackend().GetUserUploads(ctx, user, token)
	if err != nil {
		log.Warningf("Unable to get uploads for user %s : %s", user.ID, err)
		common.Fail(ctx, req, resp, "Unable to get uploads", 500)
		return
	}

	removed := 0
	for _, id := range ids {
		upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, id)
		if err != nil {
			log.Warningf("Unable to get upload %s : %s", id, err)
			continue
		}

		err = metadataBackend.GetMetaDataBackend().Remove(ctx, upload)
		if err != nil {
			log.Warningf("Unable to remove upload %s : %s", id, err)
		} else {
			removed++
		}
	}

	resp.Write(common.NewResult(fmt.Sprintf("%d uploads removed", removed), nil).ToJSON())
}
