package handlers

import (
	"net/http"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// GetUsers return users
func GetUsers(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	// Get uploads
	users, cursor, err := ctx.GetMetadataBackend().GetUsers("", false, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get users : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(users, cursor)
	common.WriteJSONResponse(resp, pagingResponse)
}

// GetUploads return uploads
func GetUploads(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	user := req.URL.Query().Get("user")
	token := req.URL.Query().Get("token")
	sort := req.URL.Query().Get("sort")

	var uploads []*common.Upload
	var cursor *paginator.Cursor
	var err error

	if sort == "size" {
		// Get uploads
		uploads, cursor, err = ctx.GetMetadataBackend().GetUploadsSortedBySize(user, token, true, pagingQuery)
		if err != nil {
			ctx.InternalServerError("unable to get uploads : %s", err)
			return
		}
	} else {
		// Get uploads
		uploads, cursor, err = ctx.GetMetadataBackend().GetUploads(user, token, true, pagingQuery)
		if err != nil {
			ctx.InternalServerError("unable to get uploads : %s", err)
			return
		}
	}

	pagingResponse := common.NewPagingResponse(uploads, cursor)
	common.WriteJSONResponse(resp, pagingResponse)
}

// GetServerStatistics return the server statistics
func GetServerStatistics(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {

	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	// Get server statistics
	stats, err := ctx.GetMetadataBackend().GetServerStatistics()
	if err != nil {
		ctx.InternalServerError("unable to get server statistics : %s", err)
		return
	}

	common.WriteJSONResponse(resp, stats)
}
