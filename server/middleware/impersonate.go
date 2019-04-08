/*
 * Charles-Antoine Mathieu <charles-antoine.mathieu@ovh.net>
 */

package middleware

import (
	"net/http"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadata"
)

// Impersonate allow an administrator to pretend being another user
func Impersonate(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := common.GetLogger(ctx)

		// Get user to impersonate from header
		newUserID := req.Header.Get("X-Plik-Impersonate")
		if newUserID != "" {

			// Check authorization
			user := common.GetUser(ctx)
			if user == nil || !user.IsAdmin() {
				log.Warningf("Unable to impersonate user : unauthorized")
				common.Fail(ctx, req, resp, "You need administrator privileges", 403)
				return
			}

			newUser, err := metadata.GetMetaDataBackend().GetUser(ctx, newUserID, "")
			if err != nil {
				log.Warningf("Unable to get user to impersonate %s : %s", newUserID, err)
				common.Fail(ctx, req, resp, "Unable to get user to impersonate", 500)
				return
			}

			if newUser == nil {
				log.Warningf("Unable to get user to impersonate : user does not exists")
				common.Fail(ctx, req, resp, "Unable to get user to impersonate : User does not exists", 403)
				return
			}

			// Fake the user as admin so you don't lose access to the admin routes
			newUser.Admin = true

			// Change user in the request context
			ctx.Set("user", newUser)
		}

		next.ServeHTTP(resp, req)
	})
}
