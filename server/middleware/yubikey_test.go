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
package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestYubikeyNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Yubikey(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestYubikeyNotEnabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	upload := common.NewUpload()
	upload.Yubikey = "token"
	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Yubikey(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Yubikey are disabled on this server")
}

func TestYubikeyMissingToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).YubikeyEnabled = true

	upload := common.NewUpload()
	upload.Yubikey = "token"
	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Yubikey(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Invalid yubikey token")
}

func TestYubikeyInvalidToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).YubikeyEnabled = true

	upload := common.NewUpload()
	upload.Yubikey = "token"
	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"yubikey": "token",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Yubikey(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Invalid yubikey token")
}

func TestYubikeyInvalidDevice(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).YubikeyEnabled = true

	upload := common.NewUpload()
	upload.Yubikey = "token"
	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"yubikey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Yubikey(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Invalid yubikey token")
}
