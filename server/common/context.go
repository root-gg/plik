/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package common

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
)

var rootLogger = logger.NewLogger()

// Logger return the root logger.
func Logger() *logger.Logger {
	return rootLogger
}

// GetLogger from the request context ( defaults to rootLogger ).
func GetLogger(ctx *juliet.Context) *logger.Logger {
	if log, ok := ctx.Get("logger"); ok {
		return log.(*logger.Logger)
	}
	return rootLogger
}

// GetSourceIP from the request context.
func GetSourceIP(ctx *juliet.Context) net.IP {
	if sourceIP, ok := ctx.Get("ip"); ok {
		return sourceIP.(net.IP)
	}
	return nil
}

// IsWhitelisted return true if the IP address in the request context is whitelisted.
func IsWhitelisted(ctx *juliet.Context) bool {
	if whitelisted, ok := ctx.Get("IsWhitelisted"); ok {
		return whitelisted.(bool)
	}

	// Check if the source IP address is in whitelist
	whitelisted := false
	if len(UploadWhitelist) > 0 {
		sourceIP := GetSourceIP(ctx)
		if sourceIP != nil {
			for _, subnet := range UploadWhitelist {
				if subnet.Contains(sourceIP) {
					whitelisted = true
					break
				}
			}
		}
	} else {
		whitelisted = true
	}
	ctx.Set("IsWhitelisted", whitelisted)
	return whitelisted
}

// GetUser from the request context.
func GetUser(ctx *juliet.Context) *User {
	if user, ok := ctx.Get("user"); ok {
		return user.(*User)
	}
	return nil
}

// GetToken from the request context.
func GetToken(ctx *juliet.Context) *Token {
	if token, ok := ctx.Get("token"); ok {
		return token.(*Token)
	}
	return nil
}

// GetFile from the request context.
func GetFile(ctx *juliet.Context) *File {
	if file, ok := ctx.Get("file"); ok {
		return file.(*File)
	}
	return nil
}

// GetUpload from the request context.
func GetUpload(ctx *juliet.Context) *Upload {
	if upload, ok := ctx.Get("upload"); ok {
		return upload.(*Upload)
	}
	return nil
}

// IsRedirectOnFailure return true if the http responde should return
// a http redirect instead of an error string.
func IsRedirectOnFailure(ctx *juliet.Context) bool {
	if redirect, ok := ctx.Get("redirect"); ok {
		return redirect.(bool)
	}
	return false
}

var userAgents = []string{"wget", "curl", "python-urllib", "libwwww-perl", "php", "pycurl"}

// Fail return write an error to the http response body.
// If IsRedirectOnFailure is true it write a http redirect that can be handled by the web client instead.
func Fail(ctx *juliet.Context, req *http.Request, resp http.ResponseWriter, message string, status int) {
	if IsRedirectOnFailure(ctx) {
		// The web client uses http redirect to get errors
		// from http redirect and display a nice HTML error message
		// But cli clients needs a clean string response
		userAgent := strings.ToLower(req.UserAgent())
		redirect := true
		for _, ua := range userAgents {
			if strings.HasPrefix(userAgent, ua) {
				redirect = false
			}
		}
		if redirect {
			http.Redirect(resp, req, fmt.Sprintf("%s/#/?err=%s&errcode=%d&uri=%s", Config.Path, message, status, req.RequestURI), 301)
			return
		}
	}

	http.Error(resp, NewResult(message, nil).ToJSONString(), status)
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
			http.Redirect(resp, req, prefix+"/", 301)
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
