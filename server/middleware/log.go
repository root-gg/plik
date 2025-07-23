package middleware

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/root-gg/plik/server/context"
)

// statusCodeResponseWriter is a responseWriter that keeps track of the response status code
type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newStatusCodeResponseWriter wraps a http.ResponseWriter in a statusCodeResponseWriter
func newStatusCodeResponseWriter(resp http.ResponseWriter) *statusCodeResponseWriter {
	return &statusCodeResponseWriter{resp, http.StatusOK}
}

// WriteHeader implement the ResponseWriter interface
func (resp *statusCodeResponseWriter) WriteHeader(code int) {
	resp.statusCode = code
	resp.ResponseWriter.WriteHeader(code)
}

// Log the http request
func Log(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		start := time.Now()
		log := ctx.GetLogger()
		config := ctx.GetConfig()

		// Create a response writer that keep track of the response status code
		statusCodeResponseWriter := newStatusCodeResponseWriter(resp)
		ctx.SetResp(statusCodeResponseWriter)

		// Serve the request
		next.ServeHTTP(statusCodeResponseWriter, req)

		// Get the response status code
		statusCode := statusCodeResponseWriter.statusCode
		statusCodeString := http.StatusText(statusCode)

		// Get the time elapsed since the request has been received
		elapsed := time.Since(start)

		// Log the request and response status and duration
		log.Infof("%v %v [%v %v] (%v)", req.Method, req.RequestURI, statusCode, statusCodeString, elapsed)

		// Update HTTP metrics
		route := mux.CurrentRoute(req)
		if route != nil {
			path, err := route.GetPathTemplate()
			if err == nil {
				ctx.GetMetrics().UpdateHTTPMetrics(req.Method, path, statusCodeResponseWriter.statusCode, elapsed)
			}
		}

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
		}

	})
}

// Log the http request
func AuditLog(ctx *context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		auditLogger := ctx.GetAuditLogger()
		if auditLogger == nil {
			return
		}

		statusCodeResponseWriter := newStatusCodeResponseWriter(resp)
		ctx.SetResp(statusCodeResponseWriter)
		next.ServeHTTP(statusCodeResponseWriter, req)

		// Get the response status code
		statusCode := statusCodeResponseWriter.statusCode

		user := ctx.GetUser()
		method := req.Method
		uri := req.RequestURI
		ipSrc := ctx.GetSourceIP().String()
		file := ctx.GetFile()
		upload := ctx.GetUpload()

		log := ""
		if user != nil {
			log += fmt.Sprintf("user_id: %s, user_login: %s, user_email: %s, ", user.ID, user.Login, user.Email)
		}
		if file != nil {
			log += fmt.Sprintf("file_name: %s, file_md5: %s, file_size: %d, mime_type: %s, upload_id: %s, ", file.Name, file.Md5, file.Size, file.Type, file.UploadID)
		}
		if upload != nil {
			log += fmt.Sprintf("upload_login: %s, upload_ttl: %d, upload_upload_token: %s, upload_ip_src: %s, ", upload.Login, upload.TTL, upload.UploadToken, upload.RemoteIP)
		}
		log += fmt.Sprintf("method: %s, uri: %s, status_code: %d, ip_src: %s", method, uri, statusCode, ipSrc)

		auditLogger.Info(log)
	})
}
