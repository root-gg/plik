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
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestRemoveUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	data := "data"

	upload := common.NewUpload()
	upload.Create()

	file1 := upload.NewFile()
	file1.Name = "file1"
	file1.Status = "uploaded"

	err := createTestFile(ctx, upload, file1, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 1")

	createTestUpload(ctx, upload)

	ctx.Set("upload", upload)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUpload(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, 0, len(respBody), "invalid response body")

	_, err = context.GetMetadataBackend(ctx).Get(ctx, upload.ID)
	require.Error(t, err, "removed upload still exists")

	_, err = context.GetDataBackend(ctx).GetFile(ctx, upload, file1.ID)
	require.Error(t, err, "removed file still exists")
}

func TestRemoveUploadNotAdmin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	data := "data"

	upload := common.NewUpload()
	upload.Create()

	file1 := upload.NewFile()
	file1.Name = "file1"
	file1.Status = "uploaded"

	err := createTestFile(ctx, upload, file1, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 1")

	createTestUpload(ctx, upload)

	ctx.Set("upload", upload)
	ctx.Set("file", file1)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUpload(ctx, rr, req)
	context.TestFail(t, rr, http.StatusForbidden, "You are not allowed to remove this upload")
}

func TestRemoveUploadNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/upload/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveUpload(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestRemoveUploadMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	upload.Create()

	createTestUpload(ctx, upload)

	ctx.Set("upload", upload)

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	RemoveUpload(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to remove upload metadata")
}

func TestRemoveUploadDataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	upload.Create()
	createTestUpload(ctx, upload)

	ctx.Set("upload", upload)

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetDataBackend(ctx).(*data_test.Backend).SetError(errors.New("data backend error"))

	rr := httptest.NewRecorder()
	RemoveUpload(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to remove upload")
}
