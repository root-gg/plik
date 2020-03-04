package middleware

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/root-gg/plik/server/context"
)

// Log the http request
func Log(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		log := ctx.GetLogger()
		config := ctx.GetConfig()

		if config.DebugRequests {

			// Don't dump request body for file upload
			dumpBody := true
			if (strings.HasPrefix(req.URL.Path, "/file") ||
				strings.HasPrefix(req.URL.Path, "/stream")) &&
				req.Method == "POST" {
				dumpBody = false
			}

			// Dump the full http request
			dump, err := httputil.DumpRequest(req, dumpBody)
			if err == nil {
				log.Debug(string(dump))
			} else {
				log.Warningf("Unable to dump HTTP request : %s", err)
			}
		} else {
			log.Infof("%v %v", req.Method, req.RequestURI)
		}

		next.ServeHTTP(resp, req)
	})
}
