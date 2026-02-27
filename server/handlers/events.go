package handlers

import (
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// GetUploadEvents return events for a specific upload
func GetUploadEvents(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
	}

	// Check authorization — must be upload admin
	if !upload.IsAdmin {
		ctx.Forbidden("you are not allowed to view events for this upload")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	// Get events
	events, cursor, err := ctx.GetMetadataBackend().GetUploadEvents(upload.ID, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get upload events : %s", err)
		return
	}

	// Sanitize events for non-admin callers (hide IP, user)
	if !ctx.IsAdmin() {
		for _, event := range events {
			event.Sanitize()
		}
	}

	// Compute human-readable messages
	for _, event := range events {
		event.ComputeMessage()
	}

	// Count total events
	total, err := ctx.GetMetadataBackend().CountUploadEvents(upload.ID)
	if err != nil {
		ctx.InternalServerError("unable to count upload events : %s", err)
		return
	}

	pagingResponse := common.NewPagingResponse(events, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}

// GetAllEvents return all events (admin only)
func GetAllEvents(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Double check authorization
	if !ctx.IsAdmin() {
		ctx.Forbidden("you need administrator privileges")
		return
	}

	pagingQuery := ctx.GetPagingQuery()

	uploadFilter := req.URL.Query().Get("upload")
	typeFilter := req.URL.Query().Get("type")

	// Get events
	events, cursor, err := ctx.GetMetadataBackend().GetEvents(uploadFilter, typeFilter, pagingQuery)
	if err != nil {
		ctx.InternalServerError("unable to get events : %s", err)
		return
	}

	// Count total events matching the filters
	total, err := ctx.GetMetadataBackend().CountEvents(uploadFilter, typeFilter)
	if err != nil {
		ctx.InternalServerError("unable to count events : %s", err)
		return
	}

	// Compute human-readable messages
	for _, event := range events {
		event.ComputeMessage()
	}

	pagingResponse := common.NewPagingResponse(events, cursor).WithTotal(total)
	common.WriteJSONResponse(resp, pagingResponse)
}
