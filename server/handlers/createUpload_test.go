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

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	metadatadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func createTestUpload(ctx *juliet.Context, uploadToCreate *common.Upload) {
	metadataBackend := context.GetMetadataBackend(ctx)
	_ = metadataBackend.Upsert(ctx, uploadToCreate)
}

func TestCreateUploadWithoutOptions(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", upload.ID, "missing upload id")
	require.NotEqual(t, "", upload.UploadToken, "missing upload token")
}

func TestCreateUploadWithOptions(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.OneShot = true
	uploadToCreate.Removable = true
	uploadToCreate.Stream = true
	uploadToCreate.User = "user"
	uploadToCreate.Token = "token"
	uploadToCreate.ProtectedByPassword = true
	uploadToCreate.Login = "foo"
	uploadToCreate.Password = "bar"

	fileToUpload := &common.File{}
	fileToUpload.Name = "file"
	fileToUpload.Reference = "0"
	uploadToCreate.Files = make(map[string]*common.File)
	uploadToCreate.Files[fileToUpload.Reference] = fileToUpload

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", upload.ID, "missing upload id")
	require.NotEqual(t, "", upload.UploadToken, "missing upload token")
	require.Equal(t, uploadToCreate.OneShot, upload.OneShot, "invalid upload oneshot status")
	require.Equal(t, uploadToCreate.Removable, upload.Removable, "invalid upload removable status")
	require.Equal(t, uploadToCreate.Stream, upload.Stream, "invalid upload stream status")
	require.Equal(t, "", upload.User, "invalid upload user")
	require.Equal(t, "", upload.Token, "invalid upload token")
	require.Equal(t, uploadToCreate.ProtectedByPassword, upload.ProtectedByPassword, "invalid upload protected by password status")
	require.Equal(t, "", upload.Login, "invalid upload login")
	require.Equal(t, "", upload.Password, "invalid upload password")
	require.Equal(t, len(uploadToCreate.Files), len(upload.Files), "invalid upload password")

	for id, file := range upload.Files {
		require.NotEqual(t, "", file.ID, "missing file id")
		require.Equal(t, id, file.ID, "invalid file id")
		require.Equal(t, fileToUpload.Name, file.Name, "invalid file name")
		require.Equal(t, fileToUpload.Reference, file.Reference, "invalid file reference")
		require.Equal(t, "missing", file.Status, "invalid file status")
	}
}

func TestCreateWithForbiddenOptions(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.ID = "custom"
	uploadToCreate.Creation = 12345
	uploadToCreate.DownloadDomain = "hack.me"
	uploadToCreate.UploadToken = "token"

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, uploadToCreate.ID, upload.ID, "invalid upload id")
	require.NotEqual(t, uploadToCreate.Creation, upload.Creation, "invalid upload creation date")
	require.NotEqual(t, uploadToCreate.UploadToken, upload.UploadToken, "invalid upload token")
	require.NotEqual(t, uploadToCreate.DownloadDomain, upload.DownloadDomain, "invalid download domain")
	require.Equal(t, 0, len(upload.Files), "invalid upload files count")
}

func TestCreateWithoutAnonymousUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).NoAnonymousUploads = true

	uploadToCreate := &common.Upload{}
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Unable to create upload from anonymous user")
}

func TestCreateNotWhitelisted(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("IsWhitelisted", false)

	uploadToCreate := &common.Upload{}
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Unable to create upload from untrusted source IP address")
}

func TestCreateInvalidRequestBody(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer([]byte("invalid request body")))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Unable to deserialize json request body")
}

func TestCreateTooManyFiles(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).MaxFilePerUpload = 2

	uploadToCreate := &common.Upload{}
	uploadToCreate.Files = make(map[string]*common.File)

	for i := 0; i < 10; i++ {
		fileToUpload := &common.File{}
		fileToUpload.Reference = strconv.Itoa(i)
		uploadToCreate.Files[fileToUpload.Reference] = fileToUpload
	}

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Maximum number file per upload reached")
}

func TestCreateOneShotWhenOneShotIsDisabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).OneShot = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.OneShot = true
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "One shot downloads are not enabled")
}

func TestCreateOneShotWhenRemovableIsDisabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).Removable = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.Removable = true
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Removable uploads are not enabled.")
}

func TestCreateStreamWhenStreamIsDisabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).StreamMode = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.Stream = true
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Stream mode is not enabled")
}

func TestCreateInvalidTTL(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).MaxTTL = 30

	uploadToCreate := &common.Upload{}
	uploadToCreate.TTL = 365
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Cannot set ttl to 365 (maximum allowed is : 30)")
}

func TestCreateInvalidNegativeTTL(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.TTL = -365
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Invalid value for ttl : -365")
}

func TestCreateWithPasswordWhenPasswordIsNotEnabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).ProtectedByPassword = false

	uploadToCreate := &common.Upload{}
	uploadToCreate.Password = "password"
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Password protection is not enabled")
}

func TestCreateWithPasswordAndDefaultLogin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.Password = "password"
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var upload = &common.Upload{}
	err = json.Unmarshal(respBody, upload)
	require.NoError(t, err, "unable to unmarshal response body")
}

func TestCreateWithYubikeyWhenYubikeyIsNotEnabled(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	uploadToCreate := &common.Upload{}
	uploadToCreate.Yubikey = "yubikey"
	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Yubikey are disabled on this server")
}

func TestCreateWithFilenameTooLong(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	uploadToCreate := common.NewUpload()
	file := common.NewFile()
	name := make([]byte, 2000)
	for i := range name {
		name[i] = 'x'
	}
	file.Name = string(name)
	uploadToCreate.Files[file.ID] = file

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "File name is too long")
}

func TestCreateWithMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetMetadataBackend(ctx).(*metadatadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	uploadToCreate := common.NewUpload()
	file := common.NewFile()
	file.Name = "name"
	uploadToCreate.Files[file.ID] = file

	reqBody, err := json.Marshal(uploadToCreate)
	require.NoError(t, err, "unable to marshal request body")

	req, err := http.NewRequest("POST", "/upload", bytes.NewBuffer(reqBody))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	CreateUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to create new upload")
}
