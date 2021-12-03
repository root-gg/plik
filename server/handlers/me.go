package handlers

import (
	"fmt"
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// UserInfo return user information ( name / email / ... )
func UserInfo(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	common.WriteJSONResponse(resp, user)
}

// GetUserTokens return user tokens
func GetUserTokens(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	// Get user tokens
	tokens, cursor, err := ctx.GetMetadataBackend().GetTokens(user.ID, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get user tokens", err)
		return
	}

	pagingResponse := common.NewPagingResponse(tokens, cursor)
	common.WriteJSONResponse(resp, pagingResponse)
}

// GetUserUploads get user uploads
func GetUserUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user, token, err := getUserAndToken(ctx, req)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	var userID, tokenStr string
	if user != nil {
		userID = user.ID
	}
	if token != nil {
		tokenStr = token.Token
	}

	// Get uploads
	uploads, cursor, err := ctx.GetMetadataBackend().GetUploads(userID, tokenStr, true, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get user uploads : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(uploads, cursor)
	common.WriteJSONResponse(resp, pagingResponse)
}

// RemoveUserUploads delete all user uploads
func RemoveUserUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user, token, err := getUserAndToken(ctx, req)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	var userID string
	if user != nil {
		userID = user.ID
	}
	var tokenStr string
	if token != nil {
		tokenStr = token.Token
	}

	deleted, err := ctx.GetMetadataBackend().RemoveUserUploads(userID, tokenStr)
	if err != nil {
		ctx.InternalServerError("unable to delete user uploads", err)
		return
	}

	_, _ = resp.Write([]byte(fmt.Sprintf("%d uploads removed", deleted)))
}

// GetUserStatistics return the user statistics
func GetUserStatistics(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user, token, err := getUserAndToken(ctx, req)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	var tokenStr *string
	if token != nil {
		tokenStr = &token.Token
	}

	// Get user statistics
	stats, err := ctx.GetMetadataBackend().GetUserStatistics(user.ID, tokenStr)
	if err != nil {
		ctx.InternalServerError("unable to get user statistics", err)
		return
	}

	common.WriteJSONResponse(resp, stats)
}

// DeleteAccount remove a user account
func DeleteAccount(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get user from context
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
		return
	}

	_, err := ctx.GetMetadataBackend().DeleteUser(user.ID)
	if err != nil {
		ctx.InternalServerError("unable to delete user account", err)
		return
	}

	_, _ = resp.Write([]byte("ok"))
}

func getUserAndToken(ctx *context.Context, req *http.Request) (user *common.User, token *common.Token, err error) {
	// Get user from context
	user = ctx.GetUser()
	if user == nil {
		return nil, nil, common.NewHTTPError("missing user, please login first", nil, http.StatusUnauthorized)
	}

	// Get token from URL query parameter
	tokenStr := req.URL.Query().Get("token")
	if tokenStr != "" {
		token, err = ctx.GetMetadataBackend().GetToken(tokenStr)
		if err != nil {
			ctx.InternalServerError("unable to get token", err)
			return nil, nil, common.NewHTTPError("unable to get token", err, http.StatusInternalServerError)
		}
		if token == nil {
			return nil, nil, common.NewHTTPError("token not found", nil, http.StatusNotFound)
		}
		if token.UserID != user.ID {
			return nil, nil, common.NewHTTPError("token not found", nil, http.StatusNotFound)
		}
	}

	return user, token, nil
}
