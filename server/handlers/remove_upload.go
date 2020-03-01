package handlers

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// RemoveUpload remove an upload and all associated files
func RemoveUpload(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		ctx.InternalServerError("missing upload from context", nil)
		return
	}

	// Check authorization
	if !upload.Removable && !ctx.IsUploadAdmin() {
		ctx.Forbidden("you are not allowed to remove this upload")
		return
	}

	err := ctx.GetMetadataBackend().DeleteUpload(upload.ID)
	if err != nil {
		ctx.InternalServerError("unable tuto delete upload", err)
		return
	}

	_, _ = resp.Write([]byte("ok"))
}
