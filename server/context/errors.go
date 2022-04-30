package context

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

var internalServerError = "internal server error"

// InternalServerError is a helper to generate http.StatusInternalServerError responses
func (ctx *Context) InternalServerError(message string, err error) {
	ctx.mu.Lock()
	config := ctx.config
	ctx.mu.Unlock()

	if config != nil && config.Debug {
		// In DEBUG mode return the error message to the user
		if err != nil {
			message = fmt.Sprintf("%s : %s", message, err)
			err = fmt.Errorf("")
		}
	} else {
		// In PROD mode return "internal server error" to the user
		message = internalServerError
	}

	ctx.Fail(message, err, http.StatusInternalServerError)
}

// BadRequest is a helper to generate http.BadRequest responses
func (ctx *Context) BadRequest(message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	ctx.Fail(message, nil, http.StatusBadRequest)
}

// NotFound is a helper to generate http.NotFound responses
func (ctx *Context) NotFound(message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	ctx.Fail(message, nil, http.StatusNotFound)
}

// Forbidden is a helper to generate http.Forbidden responses
func (ctx *Context) Forbidden(message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	ctx.Fail(message, nil, http.StatusForbidden)
}

// Unauthorized is a helper to generate http.Unauthorized responses
func (ctx *Context) Unauthorized(message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	ctx.Fail(message, nil, http.StatusUnauthorized)
}

// MissingParameter is a helper to generate http.BadRequest responses
func (ctx *Context) MissingParameter(message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	ctx.BadRequest(fmt.Sprintf("missing %s", message))
}

// InvalidParameter is a helper to generate http.BadRequest responses
func (ctx *Context) InvalidParameter(message string, params ...interface{}) {
	message = fmt.Sprintf(message, params...)
	ctx.BadRequest(fmt.Sprintf("invalid %s", message))
}

// Recover is a helper to generate http.InternalServerError responses if a panic occurs
func (ctx *Context) Recover() {
	if err := recover(); err != nil {
		ctx.InternalServerError("panic", fmt.Errorf("%v", err))
		debug.PrintStack()
	}
}

var userAgents = []string{"wget", "curl", "python-urllib", "libwwww-perl", "php", "pycurl", "go-http-client", "plik_client"}

// Fail is a helper to generate http error responses
func (ctx *Context) Fail(message string, err error, status int) {

	// Snapshot all we need
	ctx.mu.Lock()
	logger := ctx.logger
	config := ctx.config
	isRedirectOnFailure := ctx.isRedirectOnFailure
	req := ctx.req
	resp := ctx.resp
	ctx.mu.Unlock()

	// Generate log message
	logMessage := fmt.Sprintf("%s -- %d", message, status)
	if err != nil {
		logMessage = fmt.Sprintf("%s -- %v -- %d", message, err, status)
	}

	// Log message
	if logger != nil {
		if err != nil {
			logger.Critical(logMessage)
		}
	} else {
		log.Println(logMessage)
	}

	if req != nil && resp != nil {
		redirect := false
		if isRedirectOnFailure {
			// The web client uses http redirect to get errors
			// from http redirect and display a nice HTML error message
			// But cli clients needs a clean string response
			userAgent := strings.ToLower(req.UserAgent())
			redirect = true
			for _, ua := range userAgents {
				if strings.HasPrefix(userAgent, ua) {
					redirect = false
				}
			}
		}

		if config != nil && redirect {
			url := fmt.Sprintf("%s/#/?err=%s&errcode=%d&uri=%s", config.Path, message, status, req.RequestURI)
			http.Redirect(resp, req, url, http.StatusMovedPermanently)
		} else {
			http.Error(resp, message, status)
		}
	}
}
