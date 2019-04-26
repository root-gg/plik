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
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/root-gg/utils"
	"github.com/stretchr/testify/require"
)

func TestUploadNoUploadID(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing upload id")
}

func TestUploadMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": "uploadID",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Upload uploadID not found")
}

func TestUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	upload.Create()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, upload, context.GetUpload(ctx), "invalid upload from context")
}

func TestUploadExpired(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	upload.Create()
	upload.TTL = 60
	upload.Creation = time.Now().Add(-10 * time.Minute).Unix()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Upload "+upload.ID+" has expired")
}

func TestUploadToken(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	upload.Create()
	upload.UploadToken = "token"

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	req.Header.Set("X-UploadToken", upload.UploadToken)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, upload, context.GetUpload(ctx), "invalid upload from context")
	require.True(t, context.IsUploadAdmin(ctx), "invalid upload admin status")
}

func TestUploadUser(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	user := common.NewUser()
	user.ID = "user"
	ctx.Set("user", user)

	upload := common.NewUpload()
	upload.Create()
	upload.User = user.ID

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, upload, context.GetUpload(ctx), "invalid upload from context")
	require.True(t, context.IsUploadAdmin(ctx), "invalid upload admin status")
}

func TestUploadUserAdmin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	user := common.NewUser()
	user.ID = "user"
	ctx.Set("is_admin", true)
	ctx.Set("user", user)

	upload := common.NewUpload()
	upload.Create()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, upload, context.GetUpload(ctx), "invalid upload from context")
	require.True(t, context.IsUploadAdmin(ctx), "invalid upload admin status")
}

func TestUploadPasswordMissingHeader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	upload := common.NewUpload()
	upload.ProtectedByPassword = true
	upload.Create()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please provide valid credentials to access this upload")
}

func TestUploadPasswordInvalidHeader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	upload := common.NewUpload()
	upload.ProtectedByPassword = true
	upload.Create()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	req.Header.Set("Authorization", "invalid_header")

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please provide valid credentials to access this upload")
}

func TestUploadPasswordInvalidScheme(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	upload := common.NewUpload()
	upload.ProtectedByPassword = true
	upload.Create()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	req.Header.Set("Authorization", "invalid_scheme invalid_creds")

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please provide valid credentials to access this upload")
}

func TestUploadPasswordInvalidPassword(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	upload := common.NewUpload()
	upload.ProtectedByPassword = true
	upload.Create()

	err := context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	req.Header.Set("Authorization", "Basic invalid_creds")

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusUnauthorized, "Please provide valid credentials to access this upload")
}

func TestUploadPassword(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Authentication = true

	var err error

	upload := common.NewUpload()
	upload.ProtectedByPassword = true
	upload.Login = "login"
	upload.Password = "password"
	upload.Create()

	// The Authorization header will contain the base64 version of "login:password"
	// Save only the md5sum of this string to authenticate further requests
	b64str := base64.StdEncoding.EncodeToString([]byte(upload.Login + ":" + upload.Password))
	upload.Password, err = utils.Md5sum(b64str)
	require.NoError(t, err, "unable to b64encode upload credentials")

	err = context.GetMetadataBackend(ctx).Upsert(ctx, upload)
	require.NoError(t, err, "Unable to create upload")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"uploadID": upload.ID,
	}
	req = mux.SetURLVars(req, vars)

	req.Header.Add("Authorization", "Basic "+b64str)

	rr := httptest.NewRecorder()
	Upload(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, upload, context.GetUpload(ctx), "invalid upload from context")
	require.False(t, context.IsUploadAdmin(ctx), "invalid upload admin status")
}
