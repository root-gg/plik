package middleware

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/root-gg/plik/server/context"
)

// User middleware for all the /user/{userID} routes.
func User(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if ctx.GetUser() == nil {
			ctx.Unauthorized("You must be authenticated, please login first")
			return
		}

		// Get the user id from the url params
		vars := mux.Vars(req)
		userID := vars["userID"]
		if userID == "" {
			ctx.MissingParameter("user id")
			return
		}

		if userID != ctx.GetUser().ID {
			if !ctx.IsAdmin() {
				ctx.Forbidden("you need administrator privileges")
				return
			}

			// Get user from session
			user, err := ctx.GetMetadataBackend().GetUser(userID)
			if err != nil {
				ctx.InternalServerError("unable to get user", err)
				return
			}
			if user == nil {
				ctx.NotFound("user not found")
				return
			}

			ctx.SaveOriginalUser()
			ctx.SetUser(user)
		}

		next.ServeHTTP(resp, req)
	})
}
