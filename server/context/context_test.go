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
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	require.NotNil(t, GetConfig(ctx), "invalid nil config")
	ctx.Clear()
	require.Nil(t, GetConfig(ctx), "invalid not nil config")
}

func TestGetLogger(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	require.NotNil(t, GetLogger(ctx), "invalid nil logger")
	ctx.Clear()
	require.Nil(t, GetLogger(ctx), "invalid not nil logger")
}

func TestGetMetadataBackend(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	require.NotNil(t, GetMetadataBackend(ctx), "invalid nil metadata backend")
	ctx.Clear()
	require.Nil(t, GetMetadataBackend(ctx), "invalid not nil metadata backend")
}

func TestGetDataBackend(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	require.NotNil(t, GetDataBackend(ctx), "invalid nil data backend")
	ctx.Clear()
	require.Nil(t, GetDataBackend(ctx), "invalid not nil data backend")

}

func TestGetStreamBackend(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	require.NotNil(t, GetStreamBackend(ctx), "invalid nil stream backend")
	ctx.Clear()
	require.Nil(t, GetStreamBackend(ctx), "invalid not nil stream backend")
}

func TestGetSourceIP(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("ip", net.ParseIP("1.1.1.1"))
	require.NotNil(t, GetSourceIP(ctx), "invalid nil source ip")
	ctx.Clear()
	require.Nil(t, GetSourceIP(ctx), "invalid not nil source ip")
}

func TestIsWhitelistedAlreadyInContext(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())

	ctx.Set("IsWhitelisted", false)
	require.False(t, IsWhitelisted(ctx), "invalid whitelisted status")

	ctx.Set("IsWhitelisted", true)
	require.True(t, IsWhitelisted(ctx), "invalid whitelisted status")
}

func TestIsWhitelistedNoWhitelist(t *testing.T) {
	config := common.NewConfiguration()
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")

	ctx := NewTestingContext(config)
	ctx.Set("ip", net.ParseIP("1.1.1.1"))

	require.True(t, IsWhitelisted(ctx), "invalid whitelisted status")
}

func TestIsWhitelistedNoIp(t *testing.T) {
	config := common.NewConfiguration()
	config.UploadWhitelist = append(config.UploadWhitelist, "1.1.1.1")
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")

	ctx := NewTestingContext(config)

	require.False(t, IsWhitelisted(ctx), "invalid whitelisted status")
}

func TestIsWhitelisted(t *testing.T) {
	config := common.NewConfiguration()
	config.UploadWhitelist = append(config.UploadWhitelist, "1.1.1.1")
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")

	ctx := NewTestingContext(config)
	ctx.Set("ip", net.ParseIP("1.1.1.1"))

	require.True(t, IsWhitelisted(ctx), "invalid whitelisted status")
}

func TestGetUser(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("user", common.NewUser())
	require.NotNil(t, GetUser(ctx), "invalid nil user")
	ctx.Clear()
	require.Nil(t, GetUser(ctx), "invalid not nil user")
}

func TestGetToken(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("token", common.NewToken())
	require.NotNil(t, GetToken(ctx), "invalid nil token")
	ctx.Clear()
	require.Nil(t, GetToken(ctx), "invalid not nil token")
}

func TestGetFile(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("file", common.NewFile())
	require.NotNil(t, GetFile(ctx), "invalid nil file")
	ctx.Clear()
	require.Nil(t, GetFile(ctx), "invalid not nil file")
}

func TestGetUpload(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("upload", common.NewUpload())
	require.NotNil(t, GetUpload(ctx), "invalid nil upload")
	ctx.Clear()
	require.Nil(t, GetUpload(ctx), "invalid not nil upload")
}

func TestIsRedirectOnFailure(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("redirect", true)
	require.True(t, IsRedirectOnFailure(ctx), "invalid redirect status")
	ctx.Clear()
	require.False(t, IsRedirectOnFailure(ctx), "invalid redirect status")
}

func TestFailNoRedirect(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/path", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Fail(ctx, req, rr, "error", http.StatusInternalServerError)
	TestFail(t, rr, http.StatusInternalServerError, "error")
}

func TestFailWebRedirect(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("redirect", true)
	GetConfig(ctx).Path = "/root"

	req, err := http.NewRequest("GET", "/path", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.RequestURI = "/path"

	rr := httptest.NewRecorder()
	Fail(ctx, req, rr, "error", http.StatusInternalServerError)

	require.Equal(t, http.StatusMovedPermanently, rr.Code, "invalid http response status code")
	require.Contains(t, rr.Result().Header.Get("Location"), "/root", "invalid redirect root")
	require.Contains(t, rr.Result().Header.Get("Location"), "err=error", "invalid redirect message")
	require.Contains(t, rr.Result().Header.Get("Location"), "errcode=500", "invalid redirect code")
	require.Contains(t, rr.Result().Header.Get("Location"), "uri=/path", "invalid redirect path")
}

func TestFailCliNoRedirect(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	ctx.Set("redirect", true)

	req, err := http.NewRequest("GET", "/path", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("User-Agent", "wget")

	rr := httptest.NewRecorder()
	Fail(ctx, req, rr, "error", http.StatusInternalServerError)
	TestFail(t, rr, http.StatusInternalServerError, "error")
}

func TestTestFail(t *testing.T) {
	result := common.NewResult("error", nil)

	bytes, err := json.Marshal(result)
	require.NoError(t, err, "unable to marshal result")

	rr := httptest.NewRecorder()
	rr.WriteHeader(http.StatusInternalServerError)
	_, err = rr.Write(bytes)

	require.NoError(t, err, "unable to write response")
	TestFail(t, rr, http.StatusInternalServerError, "error")
}

func TestNewTestingContext(t *testing.T) {
	ctx := NewTestingContext(common.NewConfiguration())
	require.NotNil(t, ctx, "invalid nil context")
}
