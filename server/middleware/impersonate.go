package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// Impersonate allow an administrator to pretend being another user
func Impersonate(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// Get user to impersonate from header
		newUserID := req.Header.Get("X-Plik-Impersonate")
		if newUserID != "" {

			// Check authorization
			if !ctx.IsAdmin() {
				ctx.Forbidden("you need administrator privileges")
				return
			}

			newUser, err := ctx.GetMetadataBackend().GetUser(newUserID)
			if err != nil {
				ctx.InternalServerError("unable to get user", err)
				return
			}

			if newUser == nil {
				ctx.Forbidden("user to impersonate does not exists")
				return
			}

			// Change user in the request context
			ctx.SetUser(newUser)

			// Keep the admin rights
			newUser.IsAdmin = true
		}

		next.ServeHTTP(resp, req)
	})
}
