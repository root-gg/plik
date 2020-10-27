package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// CreateInviteParams to put in the HTTP request body of a POST call to CreateInvite
type CreateInviteParams struct {
	Admin    bool   `json:"admin"`
	Email    string `json:"email"`
	Validity int    `json:"validity"`
}

// CreateInvite create a new invite
func CreateInvite(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()

	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest("unable to read request body : %s", err)
		return
	}

	params := &CreateInviteParams{}

	// Deserialize json body
	if len(body) > 0 {
		err = json.Unmarshal(body, params)
		if err != nil {
			ctx.BadRequest("unable to deserialize request body : %s", err)
			return
		}
	}

	// Create invite
	invite, err := common.NewInvite(ctx.GetUser(), time.Duration(params.Validity)*time.Second)
	if err != nil {
		ctx.InternalServerError("unable to create invite", err)
		return
	}

	invite.Email = params.Email
	if ctx.IsAdmin() {
		invite.Admin = params.Admin
	}

	err = invite.PrepareInsert(ctx.GetConfig())
	if err != nil {
		ctx.BadRequest("unable to create invite : %s", err)
		return
	}

	// Save invite
	err = ctx.GetMetadataBackend().CreateInvite(invite)
	if err != nil {
		ctx.InternalServerError("unable to create token : %s", err)
		return
	}

	common.WriteJSONResponse(resp, invite)
}

// RevokeInvite remove an invite
func RevokeInvite(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	// Get invite to remove from URL params
	vars := mux.Vars(req)
	inviteID, ok := vars["invite"]
	if !ok || inviteID == "" {
		ctx.MissingParameter("invite")
		return
	}

	invite, err := ctx.GetMetadataBackend().GetInvite(inviteID)
	if err != nil {
		ctx.InternalServerError("unable to get invite", err)
		return
	}

	if invite == nil || invite.Issuer == nil || *invite.Issuer != user.ID {
		ctx.NotFound("invite not found")
		return
	}

	_, err = ctx.GetMetadataBackend().DeleteInvite(inviteID)
	if err != nil {
		ctx.InternalServerError("unable to delete invite : %s", err)
		return
	}

	common.WriteStringResponse(resp, "ok")
}

// GetUserInvites return user invites
func GetUserInvites(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	// Get user invites
	invites, cursor, err := ctx.GetMetadataBackend().GetUserInvites(user.ID, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get user invites", err)
		return
	}

	pagingResponse := common.NewPagingResponse(invites, cursor)
	common.WriteJSONResponse(resp, pagingResponse)
}
