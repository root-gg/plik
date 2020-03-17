package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// Authenticate verify that a request has either a whitelisted url or a valid auth token
func Authenticate(allowToken bool) context.Middleware {
	return func(ctx *context.Context, next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			config := ctx.GetConfig()

			if config.Authentication {
				if allowToken {
					// Get user from token header
					tokenHeader := req.Header.Get("X-PlikToken")
					if tokenHeader != "" {
						token, err := ctx.GetMetadataBackend().GetToken(tokenHeader)
						if err != nil {
							ctx.InternalServerError("unable to get token", err)
							return
						}
						if token == nil {
							ctx.Forbidden("invalid token")
							return
						}

						user, err := ctx.GetMetadataBackend().GetUser(token.UserID)
						if err != nil {
							ctx.InternalServerError("unable to get user", err)
							return
						}
						if user == nil {
							ctx.Forbidden("invalid token")
							return
						}

						// Save user and token in the request context
						ctx.SetUser(user)
						ctx.SetToken(token)

						next.ServeHTTP(resp, req)
						return
					}
				}

				sessionCookie, err := req.Cookie("plik-session")
				if err == nil && sessionCookie != nil {
					// Parse session cookie
					uid, xsrf, err := ctx.GetAuthenticator().ParseSessionCookie(sessionCookie.Value)
					if err != nil {
						common.Logout(resp, ctx.GetAuthenticator())
						ctx.Forbidden("invalid session")
						return
					}

					// Verify XSRF token
					if req.Method != "GET" && req.Method != "HEAD" {
						xsrfHeader := req.Header.Get("X-XSRFToken")
						if xsrfHeader == "" {
							common.Logout(resp, ctx.GetAuthenticator())
							ctx.Forbidden("missing xsrf header")
							return
						}
						if xsrf != xsrfHeader {
							common.Logout(resp, ctx.GetAuthenticator())
							ctx.Forbidden("invalid xsrf header")
							return
						}
					}

					// Get user from session
					user, err := ctx.GetMetadataBackend().GetUser(uid)
					if err != nil {
						common.Logout(resp, ctx.GetAuthenticator())
						ctx.InternalServerError("unable to get user from session", err)
						return
					}
					if user == nil {
						common.Logout(resp, ctx.GetAuthenticator())
						ctx.Forbidden("invalid session : user does not exists")
						return
					}

					// Save user in the request context
					ctx.SetUser(user)
				}
			}

			next.ServeHTTP(resp, req)
		})
	}
}
