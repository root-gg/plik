package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/root-gg/plik/server/common"
)

// RestrictDownloadDomain returns a gorilla/mux middleware that prevents the
// webapp UI and API from being served on the download domain (or any alias).
//
// This is a gorilla/mux middleware (func(http.Handler) http.Handler) rather
// than a plik context middleware because it must also intercept requests to
// static file handlers (webapp, /clients/, /changelog/) which are served by
// raw http.FileServer and don't go through any plik context middleware chain.
//
// When a non-file request hits the download domain (or alias):
//   - If PlikDomain is set → 302 redirect to the same path on PlikDomain
//   - If PlikDomain is not set → 403 Forbidden
func RestrictDownloadDomain(config *common.Configuration) func(http.Handler) http.Handler {
	// Defensive check: PlikDomain matching DownloadDomain (or alias) would cause redirect loops.
	// This should be caught by config.Initialize(), but panic here as a safety net.
	if config.GetPlikDomain() != nil && config.IsDownloadDomain(config.GetPlikDomain().Host) {
		panic("PlikDomain and DownloadDomain must be different domains, using the same domain would cause redirect loops")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			// No download domain separation configured → pass through.
			// Both PlikDomain and DownloadDomain must be set for download domain
			// restriction to take effect. If only DownloadDomain is set (no PlikDomain),
			// there is no domain to redirect non-file requests to, so we pass through
			// for backward compatibility with pre-1.4 deployments.
			if config.GetDownloadDomain() == nil || config.GetPlikDomain() == nil {
				next.ServeHTTP(resp, req)
				return
			}

			// Request is not on the download domain (or alias) → pass through
			if !config.IsDownloadDomain(req.Host) {
				next.ServeHTTP(resp, req)
				return
			}

			// File/stream/archive endpoints are allowed on the download domain
			if isFileEndpoint(req.URL.Path) {
				next.ServeHTTP(resp, req)
				return
			}

			// Health endpoint is exempt (load balancer probes)
			if req.URL.Path == "/health" {
				next.ServeHTTP(resp, req)
				return
			}

			// Non-file request on the download domain → redirect or reject
			if config.GetPlikDomain() != nil {
				redirectURL := fmt.Sprintf("%s%s%s", config.PlikDomain, config.Path, req.URL.RequestURI())
				http.Redirect(resp, req, redirectURL, http.StatusFound)
			} else {
				http.Error(resp, "This domain is reserved for file downloads", http.StatusForbidden)
			}
		})
	}
}

// isFileEndpoint returns true if the path is a file, stream, or archive endpoint.
func isFileEndpoint(path string) bool {
	return strings.HasPrefix(path, "/file/") ||
		strings.HasPrefix(path, "/stream/") ||
		strings.HasPrefix(path, "/archive/")
}
