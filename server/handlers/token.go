package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// CreateToken create a new token
func CreateToken(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	// Read request body
	defer func() { _ = req.Body.Close() }()

	req.Body = http.MaxBytesReader(resp, req.Body, 1048576)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		ctx.BadRequest(fmt.Sprintf("unable to read request body : %s", err))
		return
	}

	// Create token
	token := common.NewToken()

	// Deserialize json body
	if len(body) > 0 {
		err = json.Unmarshal(body, token)
		if err != nil {
			ctx.BadRequest(fmt.Sprintf("unable to deserialize request body : %s", err))
			return
		}
	}

	// Generate token uuid and set creation date
	token.Initialize()
	token.UserID = user.ID

	// Save token
	err = ctx.GetMetadataBackend().CreateToken(token)
	if err != nil {
		ctx.InternalServerError("unable to create token : %s", err)
		return
	}

	// Print token in the json response.
	var bytes []byte
	if bytes, err = utils.ToJson(token); err != nil {
		panic(fmt.Errorf("unable to serialize json response : %s", err))
	}

	_, _ = resp.Write(bytes)
}

// RevokeToken remove a token
func RevokeToken(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	// Get token to remove from URL params
	vars := mux.Vars(req)
	tokenStr, ok := vars["token"]
	if !ok || tokenStr == "" {
		ctx.MissingParameter("token")
		return
	}

	token, err := ctx.GetMetadataBackend().GetToken(tokenStr)
	if err != nil {
		ctx.InternalServerError("unable to get token : %s", err)
		return
	}

	if token == nil || token.UserID != user.ID {
		ctx.NotFound("token not found")
		return
	}

	_, err = ctx.GetMetadataBackend().DeleteToken(token.Token)
	if err != nil {
		ctx.InternalServerError("unable to delete token : %s", err)
		return
	}

	_, _ = resp.Write([]byte("ok"))
}
