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

		// Create upload
		upload := &common.Upload{}

		// Assign context parameters ( ip / user / token )
		ctx.ConfigureUploadFromContext(upload)

		// Set and validate upload parameters
		err := upload.PrepareInsert(ctx.GetConfig())
		if err != nil {
			ctx.BadRequest(err.Error())
			return
		}

		// Save the upload metadata
		err = ctx.GetMetadataBackend().CreateUpload(upload)
		if err != nil {
			ctx.InternalServerError("unable to create upload", err)
			return
		}

		// Save upload in the request context
		ctx.SetUpload(upload)
		ctx.SetUploadAdmin(true)

		// Change the output of the addFile handler
		ctx.SetQuick(true)

		next.ServeHTTP(resp, req)
	})
}
