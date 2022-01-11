package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// Context sets necessary request context values
func Context(setupContext func(ctx *context.Context)) context.Middleware {
	return func(ctx *context.Context, next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

			setupContext(ctx)
			ctx.SetReq(req)
			ctx.SetResp(resp)

			next.ServeHTTP(resp, req)
		})
	}
}
