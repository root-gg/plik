package middleware

import (
	"net/http"
	"strings"

	"github.com/root-gg/plik/server/context"
)

// maxAPIBodySize is the maximum allowed request body size for API endpoints (1 MiB).
// File upload endpoints have their own stream-based size limiting in the preprocessor.
const maxAPIBodySize = 1 << 20

// LimitBody wraps the request body with http.MaxBytesReader to reject
// oversized payloads on API endpoints that expect small JSON bodies.
//
// It skips requests that don't carry a body (GET, HEAD, DELETE, OPTIONS)
// and file upload paths which have their own stream-based size limiting.
func LimitBody(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// Only limit methods that carry a request body
		if req.Method != http.MethodPost && req.Method != http.MethodPatch && req.Method != http.MethodPut {
			next.ServeHTTP(resp, req)
			return
		}

		// Skip file upload paths (they have stream-based size limiting)
		if req.URL.Path == "/" ||
			strings.HasPrefix(req.URL.Path, "/file/") ||
			strings.HasPrefix(req.URL.Path, "/resumable/") ||
			strings.HasPrefix(req.URL.Path, "/stream/") {
			next.ServeHTTP(resp, req)
			return
		}

		req.Body = http.MaxBytesReader(resp, req.Body, maxAPIBodySize)
		next.ServeHTTP(resp, req)
	})
}
