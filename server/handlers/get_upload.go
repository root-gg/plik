package handlers

import (
	"net/http"

	"github.com/root-gg/plik/server/common"

	"github.com/root-gg/plik/server/context"
)

// GetUpload return upload metadata
func GetUpload(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	config := ctx.GetConfig()

	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
	}

	files, err := ctx.GetMetadataBackend().GetFiles(upload.ID)
	if err != nil {
		ctx.InternalServerError("unable to get upload files", err)
		return
	}

	upload.Files = files

	// Hide private information (IP, data backend details, User ID, Login/Password, ...)
	upload.Sanitize(config)

	common.WriteJSONResponse(resp, upload)
}
