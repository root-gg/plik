package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// PatchMe partially updates the authenticated user's self-editable fields.
// Only fields present in the JSON body are updated — absent fields are untouched.
// Admin-only fields (IsAdmin, MaxFileSize, MaxUserSize, MaxTTL) cannot be set via this endpoint.
func PatchMe(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user := ctx.GetUser()
	if user == nil {
		ctx.Unauthorized("missing user, please login first")
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
		ctx.BadRequest("missing request body")
		return
	}

	// Parse as map to detect which fields were provided
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(body, &patch); err != nil {
		ctx.BadRequest("unable to deserialize request body : %s", err)
		return
	}

	// Apply only provided self-editable fields
	if raw, ok := patch["theme"]; ok {
		if err := json.Unmarshal(raw, &user.Theme); err != nil {
			ctx.BadRequest("invalid theme value : %s", err)
			return
		}
	}
	if raw, ok := patch["language"]; ok {
		if err := json.Unmarshal(raw, &user.Language); err != nil {
			ctx.BadRequest("invalid language value : %s", err)
			return
		}
	}
	if raw, ok := patch["name"]; ok {
		if err := json.Unmarshal(raw, &user.Name); err != nil {
			ctx.BadRequest("invalid name value : %s", err)
			return
		}
	}
	if raw, ok := patch["email"]; ok {
		if err := json.Unmarshal(raw, &user.Email); err != nil {
			ctx.BadRequest("invalid email value : %s", err)
			return
		}
	}

	if err := ctx.GetMetadataBackend().UpdateUser(user); err != nil {
		ctx.InternalServerError("unable to update user : %s", err)
		return
	}

	common.WriteJSONResponse(resp, user)
}

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
	user, tokenStr, err := getUserAndTokenStr(ctx, req)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	pagingQuery := ctx.GetPagingQuery()
	sort := req.URL.Query().Get("sort")

	filters := parseBadgeFilters(req)
	filters.User = user.ID
	filters.Token = tokenStr

	var uploads []*common.Upload
	var cursor *paginator.Cursor

	if sort == "size" {
		uploads, cursor, err = ctx.GetMetadataBackend().GetUploadsSortedBySize(filters, true, pagingQuery)
		if err != nil {
			ctx.InternalServerError("unable to get user uploads : %s", err)
			return
		}
	} else {
		uploads, cursor, err = ctx.GetMetadataBackend().GetUploads(filters, true, pagingQuery)
		if err != nil {
			ctx.InternalServerError("unable to get user uploads : %s", err)
			return
		}
	}

	total, err := ctx.GetMetadataBackend().CountUploads(filters)
	if err != nil {
		ctx.InternalServerError("unable to count user uploads : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(uploads, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}

// RemoveUserUploads delete all user uploads
func RemoveUserUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user, tokenStr, err := getUserAndTokenStr(ctx, req)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	deleted, err := ctx.GetMetadataBackend().RemoveUserUploads(user.ID, tokenStr)
	if err != nil {
		ctx.InternalServerError("unable to delete user uploads", err)
		return
	}

	_, _ = resp.Write(fmt.Appendf(nil, "%d uploads removed", deleted))
}

// GetUserStatistics return the user statistics
func GetUserStatistics(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	user, tokenStr, err := getUserAndTokenStr(ctx, req)
	if err != nil {
		handleHTTPError(ctx, err)
		return
	}

	var tokenPtr *string
	if tokenStr != "" {
		tokenPtr = &tokenStr
	}

	// Get user statistics
	stats, err := ctx.GetMetadataBackend().GetUserStatistics(user.ID, tokenPtr)
	if err != nil {
		ctx.InternalServerError("unable to get user statistics", err)
		return
	}

	common.WriteJSONResponse(resp, stats)
}

// DeleteAccount remove a user account
func DeleteAccount(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Feature flag check : only gate the self-delete endpoint (DELETE /me)
	// The admin route (DELETE /user/{userID}) always has a userID mux variable
	vars := mux.Vars(req)
	if _, isAdminRoute := vars["userID"]; !isAdminRoute {
		config := ctx.GetConfig()
		if config.FeatureDeleteAccount == common.FeatureDisabled {
			ctx.BadRequest("delete account is not enabled")
			return
		}
	}

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

// uuidRe matches a UUID v4 string (8-4-4-4-12 hex).
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// getUserAndTokenStr extracts the authenticated user and an optional token
// query parameter (validated as a UUID string, no DB lookup).
// This allows filtering by revoked tokens — the token doesn't need to exist.
func getUserAndTokenStr(ctx *context.Context, req *http.Request) (user *common.User, tokenStr string, err error) {
	user = ctx.GetUser()
	if user == nil {
		return nil, "", common.NewHTTPError("missing user, please login first", nil, http.StatusUnauthorized)
	}

	tokenStr = req.URL.Query().Get("token")
	if tokenStr != "" && !uuidRe.MatchString(tokenStr) {
		return nil, "", common.NewHTTPError("invalid token format", nil, http.StatusBadRequest)
	}

	return user, tokenStr, nil
}
