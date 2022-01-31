package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// CreateUpload create a new upload on the fly to be used in the next handler
func CreateUpload(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if !ctx.IsWhitelisted() {
			ctx.Forbidden("untrusted source IP address")
			return
		}

		// Create upload with default params
		upload, err := ctx.CreateUpload(&common.Upload{})
		if err != nil {
			ctx.BadRequest("unable to create upload : %s", err)
			return
		}

		// Save the upload metadata
		err = ctx.GetMetadataBackend().CreateUpload(upload)
		if err != nil {
			ctx.InternalServerError("unable to create upload", err)
			return
		}

		// You are always admin of your own uploads
		upload.IsAdmin = true

		// Save upload in the request context
		ctx.SetUpload(upload)

		// Change the output of the addFile handler
		ctx.SetQuick(true)

		next.ServeHTTP(resp, req)
	})
}
