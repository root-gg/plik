/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// Authenticate verify that a request has either a whitelisted url or a valid auth token
func Authenticate(allowToken bool) juliet.ContextMiddleware {
	return func(ctx *juliet.Context, next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			log := context.GetLogger(ctx)
			config := context.GetConfig(ctx)

			if config.Authentication {
				if allowToken {
					// Get user from token header
					tokenHeader := req.Header.Get("X-PlikToken")
					if tokenHeader != "" {
						user, err := context.GetMetadataBackend(ctx).GetUser(ctx, "", tokenHeader)
						if err != nil {
							log.Warningf("Unable to get user from token %s : %s", tokenHeader, err)
							context.Fail(ctx, req, resp, "Unable to get user", 500)
							return
						}
						if user == nil {
							log.Warningf("Unable to get user from token %s", tokenHeader)
							context.Fail(ctx, req, resp, "Invalid token", 403)
							return
						}

						// Get token from user
						var token *common.Token
						for _, t := range user.Tokens {
							if t.Token == tokenHeader {
								token = t
								break
							}
						}
						if token == nil {
							// THIS SHOULD NEVER HAPPEN
							log.Warningf("Unable to get token %s from user %s", tokenHeader, user.ID)
							context.Fail(ctx, req, resp, "Invalid token", 500)
							return
						}

						// Save user and token in the request context
						ctx.Set("user", user)
						ctx.Set("token", token)

						next.ServeHTTP(resp, req)
						return
					}
				}

				sessionCookie, err := req.Cookie("plik-session")
				if err == nil && sessionCookie != nil {
					// Parse session cookie
					uid, xsrf, err := common.ParseSessionCookie(sessionCookie.Value, config)
					if err != nil {
						log.Warningf("Invalid session : %s", err)
						common.Logout(resp)
						context.Fail(ctx, req, resp, "Invalid session", 403)
						return
					}

					// Verify XSRF token
					if req.Method != "GET" && req.Method != "HEAD" {
						xsrfHeader := req.Header.Get("X-XSRFToken")
						if xsrfHeader == "" {
							log.Warning("Missing xsrf header")
							common.Logout(resp)
							context.Fail(ctx, req, resp, "Missing xsrf header", 403)
							return
						}
						if xsrf != xsrfHeader {
							log.Warning("Invalid xsrf header")
							common.Logout(resp)
							context.Fail(ctx, req, resp, "Invalid xsrf header", 403)
							return
						}
					}

					// Get user from session
					user, err := context.GetMetadataBackend(ctx).GetUser(ctx, uid, "")
					if err != nil {
						log.Warningf("Unable to get user from session : %s", err)
						common.Logout(resp)
						context.Fail(ctx, req, resp, "Unable to get user", 500)
						return
					}
					if user == nil {
						log.Warningf("Invalid session : user does not exists")
						common.Logout(resp)
						context.Fail(ctx, req, resp, "Invalid session : User does not exists", 403)
						return
					}

					// Save user in the request context
					ctx.Set("user", user)

					// Authenticate admin users
					if config.IsUserAdmin(user) {
						ctx.Set("is_admin", true)
					}
				}
			}

			next.ServeHTTP(resp, req)
		})
	}
}
