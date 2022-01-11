package middleware

import (
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// Recover the http request in case of panic
func Recover(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {

		// Recover from panic and return a nice http.InternalServerError()
		defer ctx.Recover()

		// This middleware need to be added after the Log middleware to
		// properly log the recovered InternalServerError

		next.ServeHTTP(resp, req)
	})
}
