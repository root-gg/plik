package common

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/root-gg/utils"
)

//Ensure HTTPError implements error
var _ error = (*HTTPError)(nil)

// HTTPError allows to return an error and a HTTP status code
type HTTPError struct {
	Message    string
	Err        error
	StatusCode int
}

// NewHTTPError return a new HTTPError
func NewHTTPError(message string, err error, code int) HTTPError {
	return HTTPError{message, err, code}
}

// Error return the error
func (e HTTPError) Error() string {
	return e.String()
}

func (e HTTPError) String() string {
	if e.Err != nil {
		return fmt.Sprintf("%s : %s", e.Message, e.Err)
	}
	return e.Message
}

// StripPrefix returns a handler that serves HTTP requests
// removing the given prefix from the request URL's Path
// It differs from http.StripPrefix by defaulting to "/" and not ""
func StripPrefix(prefix string, handler http.Handler) http.Handler {
	if prefix == "" || prefix == "/" {
		return handler
	}
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		// Relative paths to javascript, css, ... imports won't work without a tailing slash
		if req.URL.Path == prefix {
			http.Redirect(resp, req, prefix+"/", http.StatusMovedPermanently)
			return
		}
		if p := strings.TrimPrefix(req.URL.Path, prefix); len(p) < len(req.URL.Path) {
			req.URL.Path = p
		} else {
			http.NotFound(resp, req)
			return
		}
		if !strings.HasPrefix(req.URL.Path, "/") {
			req.URL.Path = "/" + req.URL.Path
		}
		handler.ServeHTTP(resp, req)
	})
}

// EncodeAuthBasicHeader return the base64 version of "login:password"
func EncodeAuthBasicHeader(login string, password string) (value string) {
	return base64.StdEncoding.EncodeToString([]byte(login + ":" + password))
}

// WriteJSONResponse serialize the response to json and write it to the HTTP response body
func WriteJSONResponse(resp http.ResponseWriter, obj interface{}) {
	json, err := utils.ToJson(obj)
	if err != nil {
		panic(fmt.Errorf("unable to serialize json response : %s", err))
	}

	_, _ = resp.Write(json)
}

// AskConfirmation from process input
func AskConfirmation(defaultValue bool) (bool, error) {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(strings.ToLower(input), "y") {
		return true, nil
	} else if strings.HasPrefix(strings.ToLower(input), "n") {
		return false, nil
	} else {
		return defaultValue, nil
	}
}
