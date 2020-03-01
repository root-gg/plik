package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// RedirectOnFailure enable webapp http redirection instead of string error
func RedirectOnFailure(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx.SetRedirectOnFailure(true)
		next.ServeHTTP(resp, req)
	})
}
