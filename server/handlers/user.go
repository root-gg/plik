package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// CreateUser create a new user
func CreateUser(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest("unable to read request body : %s", err)
		return
	}

	if len(body) == 0 {
		ctx.BadRequest("unable to deserialize user : missing")
		return
	}

	// Deserialize json body
	userParams := &common.User{}
	err = json.Unmarshal(body, userParams)
	if err != nil {
		ctx.BadRequest("unable to deserialize user : %s", err)
		return
	}

	// Deserialize password because it's a private field
	password := &struct {
		Password string `json:"password"`
	}{}
	err = json.Unmarshal(body, password)
	if err != nil {
		ctx.BadRequest("unable to deserialize password : %s", err)
		return
	}
	userParams.Password = password.Password

	// Create user from user params
	user, err := common.CreateUserFromParams(userParams)
	if err != nil {
		ctx.BadRequest("unable to create user : %s", err)
		return
	}

	err = ctx.GetMetadataBackend().CreateUser(user)
	if err != nil {
		ctx.InternalServerError("unable to save user : %s", err)
		return
	}

	common.WriteJSONResponse(resp, user)
}

// UpdateUser edit an existing user
func UpdateUser(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user := ctx.GetUser()

	// Double check authorization
	if user == nil {
		ctx.Unauthorized("you need to be authenticated, please login first")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()
	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := io.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest("unable to read request body : %s", err)
		return
	}

	if len(body) == 0 {
		ctx.BadRequest("unable to deserialize user : missing")
		return
	}

	// Deserialize json body
	userParams := &common.User{}
	err = json.Unmarshal(body, userParams)
	if err != nil {
		ctx.BadRequest("unable to deserialize user : %s", err)
		return
	}

	if userParams.ID != user.ID {
		ctx.BadRequest("user id mismatch")
		return
	}

	if !ctx.IsAdmin() {
		if userParams.IsAdmin && !user.IsAdmin {
			ctx.Forbidden("can't grant yourself admin right, nice try!")
			return
		}
		if userParams.MaxTTL != user.MaxTTL || userParams.MaxFileSize != user.MaxFileSize {
			ctx.Forbidden("can't edit your own quota, nice try!")
			return
		}
	}

	// Deserialize password because it's a private field
	password := &struct {
		Password string `json:"password"`
	}{}
	err = json.Unmarshal(body, password)
	if err != nil {
		ctx.BadRequest("unable to deserialize password : %s", err)
		return
	}
	userParams.Password = password.Password

	// Create user from user params
	err = common.UpdateUser(user, userParams)
	if err != nil {
		ctx.BadRequest("unable to update user : %s", err)
		return
	}

	err = ctx.GetMetadataBackend().UpdateUser(user)
	if err != nil {
		ctx.InternalServerError("unable to save user : %s", err)
		return
	}

	common.WriteJSONResponse(resp, user)
}
