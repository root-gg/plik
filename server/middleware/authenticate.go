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

	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/metadataBackend"
)

// Authenticate verify that a request has either a whitelisted url or a valid auth token
func Authenticate(allowToken bool) juliet.ContextMiddleware {
	return func(ctx *juliet.Context, next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			log := common.GetLogger(ctx)
			log.Debug("User middleware")

			if common.Config.Authentication {
				if allowToken {
					// Get user from token header
					tokenHeader := req.Header.Get("X-PlikToken")
					if tokenHeader != "" {
						user, err := metadataBackend.GetMetaDataBackend().GetUser(ctx, "", tokenHeader)
						if err != nil {
							log.Warningf("Unable to get user from token %s : %s", tokenHeader, err)
							common.Fail(ctx, req, resp, "Unable to get user", 500)
							return
						}
						if user == nil {
							log.Warningf("Unable to get user from token %s", tokenHeader)
							common.Fail(ctx, req, resp, "Invalid token", 403)
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
							log.Warningf("Unable to get token %s from user %s", tokenHeader, user.ID)
							common.Fail(ctx, req, resp, "Invalid token", 403)
							return
						}

						// Save user and token in the request context
						ctx.Set("user", user)
						ctx.Set("token", token)
					}
				}

				// Get user from session cookie
				sessionCookie, err := req.Cookie("plik-session")
				if err == nil && sessionCookie != nil {

					// Parse session cookie
					session, err := jwt.Parse(sessionCookie.Value, func(t *jwt.Token) (interface{}, error) {
						// Verify signing algorithm
						if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, fmt.Errorf("Unexpected siging method : %v", t.Header["alg"])
						}

						// Get authentication provider
						provider, ok := t.Claims["provider"]
						if !ok {
							return nil, fmt.Errorf("Missing authentication provider")
						}

						switch provider {
						case "google":
							if !common.Config.GoogleAuthentication {
								return nil, fmt.Errorf("Missing Google API credentials")
							}
							return []byte(common.Config.GoogleAPISecret), nil
						case "ovh":
							if !common.Config.OvhAuthentication {
								return nil, fmt.Errorf("Missing OVH API credentials")
							}
							return []byte(common.Config.OvhAPISecret), nil
						default:
							return nil, fmt.Errorf("Invalid authentication provider : %s", provider)
						}
					})
					if err != nil {
						log.Warningf("Invalid session : %s", err)
						common.Logout(resp)
						common.Fail(ctx, req, resp, "Invalid session", 403)
						return
					}

					// Verify xsrf token
					if req.Method != "GET" && req.Method != "HEAD" {
						if xsrfCookie, ok := session.Claims["xsrf"]; ok {
							xsrfHeader := req.Header.Get("X-XRSFToken")
							if xsrfHeader == "" {
								log.Warning("Missing xsrf header")
								common.Logout(resp)
								common.Fail(ctx, req, resp, "Missing xsrf header", 403)
								return
							}
							if xsrfCookie != xsrfHeader {
								log.Warning("Invalid xsrf header")
								common.Logout(resp)
								common.Fail(ctx, req, resp, "Invalid xsrf header", 403)
								return
							}
						} else {
							log.Warning("Invalid session : missing xsrf token")
							common.Logout(resp)
							common.Fail(ctx, req, resp, "Invalid session : missing xsrf token", 500)
							return
						}
					}

					// Get user from session
					if userID, ok := session.Claims["uid"]; ok {
						user, err := metadataBackend.GetMetaDataBackend().GetUser(ctx, userID.(string), "")
						if err != nil {
							log.Warningf("Unable to get user from session : %s", err)
							common.Logout(resp)
							common.Fail(ctx, req, resp, "Unable to get user", 500)
							return
						}
						if user == nil {
							log.Warningf("Invalid session : user does not exists")
							common.Logout(resp)
							common.Fail(ctx, req, resp, "Invalid session : User does not exists", 403)
							return
						}

						// Save user in the request context
						ctx.Set("user", user)
					}
				}
			}

			next.ServeHTTP(resp, req)
		})
	}
}
