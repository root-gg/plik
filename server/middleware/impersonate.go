/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package middleware

import (
	"net/http"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/context"
)

// Impersonate allow an administrator to pretend being another user
func Impersonate(ctx *juliet.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := context.GetLogger(ctx)

		// Get user to impersonate from header
		newUserID := req.Header.Get("X-Plik-Impersonate")
		if newUserID != "" {

			// Check authorization
			if !context.IsAdmin(ctx) {
				log.Warningf("Unable to impersonate user : unauthorized")
				context.Fail(ctx, req, resp, "You need administrator privileges", 403)
				return
			}

			newUser, err := context.GetMetadataBackend(ctx).GetUser(ctx, newUserID, "")
			if err != nil {
				log.Warningf("Unable to get user to impersonate %s : %s", newUserID, err)
				context.Fail(ctx, req, resp, "Unable to get user to impersonate", 500)
				return
			}

			if newUser == nil {
				log.Warningf("Unable to get user to impersonate : user does not exists")
				context.Fail(ctx, req, resp, "Unable to get user to impersonate : User does not exists", 403)
				return
			}

			// Change user in the request context
			ctx.Set("user", newUser)
		}

		next.ServeHTTP(resp, req)
	})
}
