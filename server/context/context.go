/* The MIT License (MIT)

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
THE SOFTWARE. */

package context

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	datatest "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
	metadatatest "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

// TODO Error Management

// GetConfig from the request context.
func GetConfig(ctx *juliet.Context) (config *common.Configuration) {
	if config, ok := ctx.Get("config"); ok {
		return config.(*common.Configuration)
	}
	return nil
}

// GetLogger from the request context.
func GetLogger(ctx *juliet.Context) *logger.Logger {
	if log, ok := ctx.Get("logger"); ok {
		return log.(*logger.Logger)
	}
	return nil
}

// GetMetadataBackend from the request context.
func GetMetadataBackend(ctx *juliet.Context) metadata.Backend {
	if backend, ok := ctx.Get("metadata_backend"); ok {
		return backend.(metadata.Backend)
	}
	return nil
}

// GetDataBackend from the request context.
func GetDataBackend(ctx *juliet.Context) data.Backend {
	if backend, ok := ctx.Get("data_backend"); ok {
		return backend.(data.Backend)
	}
	return nil
}

// GetStreamBackend from the request context.
func GetStreamBackend(ctx *juliet.Context) data.Backend {
	if backend, ok := ctx.Get("stream_backend"); ok {
		return backend.(data.Backend)
	}
	return nil
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

	uploadWhitelist := GetConfig(ctx).GetUploadWhitelist()

	// Check if the source IP address is in whitelist
	whitelisted := false
	if len(uploadWhitelist) > 0 {
		sourceIP := GetSourceIP(ctx)
		if sourceIP != nil {
			for _, subnet := range uploadWhitelist {
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
func GetUser(ctx *juliet.Context) *common.User {
	if user, ok := ctx.Get("user"); ok {
		return user.(*common.User)
	}
	return nil
}

// GetToken from the request context.
func GetToken(ctx *juliet.Context) *common.Token {
	if token, ok := ctx.Get("token"); ok {
		return token.(*common.Token)
	}
	return nil
}

// GetFile from the request context.
func GetFile(ctx *juliet.Context) *common.File {
	if file, ok := ctx.Get("file"); ok {
		return file.(*common.File)
	}
	return nil
}

// GetUpload from the request context.
func GetUpload(ctx *juliet.Context) *common.Upload {
	if upload, ok := ctx.Get("upload"); ok {
		return upload.(*common.Upload)
	}
	return nil
}

// IsUploadAdmin returns true if the context has verified that current request can modify the upload
func IsUploadAdmin(ctx *juliet.Context) bool {
	if admin, ok := ctx.Get("is_upload_admin"); ok {
		return admin.(bool)
	}
	return false
}

// IsAdmin check if the user is a Plik server administrator
func IsAdmin(ctx *juliet.Context) bool {
	if admin, ok := ctx.Get("is_admin"); ok {
		return admin.(bool)
	}
	return false
}

// IsRedirectOnFailure return true if the http response should return
// a http redirect instead of an error string.
func IsRedirectOnFailure(ctx *juliet.Context) bool {
	if redirect, ok := ctx.Get("redirect"); ok {
		return redirect.(bool)
	}
	return false
}

var userAgents = []string{"wget", "curl", "python-urllib", "libwwww-perl", "php", "pycurl", "Go-http-client"}

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
			config := GetConfig(ctx)
			http.Redirect(resp, req, fmt.Sprintf("%s/#/?err=%s&errcode=%d&uri=%s", config.Path, message, status, req.RequestURI), 301)
			return
		}
	}

	http.Error(resp, common.NewResult(message, nil).ToJSONString(), status)
}

// TestFail is a helper to test a httptest.ResponseRecoreder status
func TestFail(t *testing.T, resp *httptest.ResponseRecorder, status int, message string) {
	require.Equal(t, status, resp.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, err, 0, len(respBody), "empty response body")

	var result = &common.Result{}
	err = json.Unmarshal(respBody, result)
	require.NoError(t, err, "unable to unmarshal error")

	if message != "" {
		require.Contains(t, result.Message, message, "invalid response error message")
	}
}

// NewTestingContext is a helper to create a context to test handlers and middlewares
func NewTestingContext(config *common.Configuration) (ctx *juliet.Context) {
	ctx = juliet.NewContext()
	ctx.Set("config", config)
	ctx.Set("logger", logger.NewLogger())
	ctx.Set("metadata_backend", metadatatest.NewBackend())
	ctx.Set("data_backend", datatest.NewBackend())
	ctx.Set("stream_backend", datatest.NewBackend())
	return ctx
}
