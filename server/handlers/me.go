package handlers

import (
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
	"net/http"
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

	// TODO TOKEN FILTER

	// Get uploads
	ids, err := metadataBackend.GetMetaDataBackend().GetUserUploads(ctx, user, nil)
	if err != nil {
		log.Warningf("Unable to get uploads for user %s : %s", user.ID, err)
		common.Fail(ctx, req, resp, "Unable to get uploads", 500)
		return
	}

	uploads := []*common.Upload{}
	for _, id := range ids {
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

	// TODO TOKEN FILTER

	// Get uploads
	ids, err := metadataBackend.GetMetaDataBackend().GetUserUploads(ctx, user, nil)
	if err != nil {
		log.Warningf("Unable to get uploads for user %s : %s", user.ID, err)
		common.Fail(ctx, req, resp, "Unable to get uploads", 500)
		return
	}

	for _, id := range ids {
		upload, err := metadataBackend.GetMetaDataBackend().Get(ctx, id)
		if err != nil {
			log.Warningf("Unable to get upload %s : %s", id, err)
			continue
		}

		err = metadataBackend.GetMetaDataBackend().Remove(ctx, upload)
		if err != nil {
			log.Warningf("Unable to remove upload %s : %s", id, err)
		}
	}
}
