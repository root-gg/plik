package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func getUserFromToken(ctx *context.Context) (*common.User, *common.Token, *common.HTTPError) {
	req := ctx.GetReq()

	// Get user from token header
	tokenHeader := req.Header.Get("X-PlikToken")
	if tokenHeader != "" {
		token, err := ctx.GetMetadataBackend().GetToken(tokenHeader)
		if err != nil {
			return nil, nil, &common.HTTPError{Message: "unable to get token", Err: err, StatusCode: http.StatusInternalServerError}
		}
		if token == nil {
			return nil, nil, &common.HTTPError{Message: "invalid token", StatusCode: http.StatusForbidden}
		}

		user, err := ctx.GetMetadataBackend().GetUser(token.UserID)
		if err != nil {
			return nil, nil, &common.HTTPError{Message: "unable to get user", Err: err, StatusCode: http.StatusInternalServerError}
		}
		if user == nil {
			return nil, nil, &common.HTTPError{Message: "invalid token", StatusCode: http.StatusForbidden}
		}

		return user, token, nil
	}

	return nil, nil, nil
}

func getUserFromSessionCookie(ctx *context.Context) (*common.User, *common.HTTPError) {
	req := ctx.GetReq()

	sessionCookie, err := req.Cookie(common.SessionCookieName)
	if err == nil && sessionCookie != nil {
		// Parse session cookie
		uid, xsrf, err := ctx.GetAuthenticator().ParseSessionCookie(sessionCookie.Value)
		if err != nil {
			return nil, &common.HTTPError{Message: "invalid session", StatusCode: http.StatusForbidden}
		}

		// Verify XSRF token
		if req.Method != "GET" && req.Method != "HEAD" {
			xsrfHeader := req.Header.Get("X-XSRFToken")
			if xsrfHeader == "" {
				return nil, &common.HTTPError{Message: "missing xsrf header", StatusCode: http.StatusForbidden}
			}
			if xsrf != xsrfHeader {
				return nil, &common.HTTPError{Message: "invalid xsrf header", StatusCode: http.StatusForbidden}
			}
		}

		// Get user from session
		user, err := ctx.GetMetadataBackend().GetUser(uid)
		if err != nil {
			return nil, &common.HTTPError{Message: "unable to get user from session", Err: err, StatusCode: http.StatusInternalServerError}
		}
		if user == nil {
			return nil, &common.HTTPError{Message: "invalid session : user does not exists", StatusCode: http.StatusForbidden}
		}

		return user, nil
	}

	return nil, nil
}

// Authenticate verify that a request has either a whitelisted url or a valid auth token
func Authenticate(allowToken bool) context.Middleware {
	return func(ctx *context.Context, next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			config := ctx.GetConfig()
			if config.FeatureAuthentication != common.FeatureDisabled {
				if allowToken {
					user, token, err := getUserFromToken(ctx)
					if err != nil {
						ctx.Error(err)
						return
					}

					if user != nil && token != nil {
						// Save user and token in the request context
						ctx.SetUser(user)
						ctx.SetToken(token)

						// Continue to the next middleware in the chain
						next.ServeHTTP(resp, req)
						return
					}
				}

				user, err := getUserFromSessionCookie(ctx)
				if err != nil {
					common.Logout(resp, ctx.GetAuthenticator())
					ctx.Error(err)
					return
				}

				if user != nil {
					// Save user in the request context
					ctx.SetUser(user)

					// Continue to the next middleware in the chain
					next.ServeHTTP(resp, req)
					return
				}
			}

			// Continue to the next middleware in the chain
			next.ServeHTTP(resp, req)
		})
	}
}

// AuthenticatedOnly middleware to allow only authenticated users to the next middleware in the chain
func AuthenticatedOnly(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if ctx.GetUser() == nil {
			ctx.Unauthorized("you must be authenticated, please login first")
			return
		}

		next.ServeHTTP(resp, req)
	})
}

// AdminOnly middleware to allow only admin users to the next middleware in the chain
func AdminOnly(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		user := ctx.GetUser()
		if user == nil {
			ctx.Unauthorized("you must be authenticated, please login first")
			return
		}

		if !user.IsAdmin {
			ctx.Forbidden("you need administrator privileges")
			return
		}

		next.ServeHTTP(resp, req)
	})
}
