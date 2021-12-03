package handlers

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// RemoveFile remove a file from an existing upload
func RemoveFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		ctx.InternalServerError("missing upload from context", nil)
		return
	}

	// Check authorization
	if !upload.Removable && !upload.IsAdmin {
		ctx.Forbidden("you are not allowed to remove files from this upload")
		return
	}

	// Get file from context
	file := ctx.GetFile()
	if file == nil {
		ctx.InternalServerError("missing file from context", nil)
		return
	}

	// Delete file
	err := ctx.GetMetadataBackend().RemoveFile(file)
	if err != nil {
		ctx.InternalServerError("unable to delete file", err)
		return
	}

	_, _ = resp.Write([]byte("ok"))
}
