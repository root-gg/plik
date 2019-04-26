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
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestRemoveFile(t *testing.T) {
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

	file2 := upload.NewFile()
	file2.Name = "file2"
	file2.Status = "uploaded"

	err = createTestFile(ctx, upload, file2, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 2")

	createTestUpload(ctx, upload)

	ctx.Set("upload", upload)
	ctx.Set("file", file1)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var uploadResult = &common.Upload{}
	err = json.Unmarshal(respBody, uploadResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 2, len(uploadResult.Files), "invalid upload files count")
	require.Equal(t, "removed", uploadResult.Files[file1.ID].Status, "invalid removed file status")

	_, err = context.GetDataBackend(ctx).GetFile(ctx, upload, file1.ID)
	require.Error(t, err, "removed file still exists")
}

func TestRemoveFileNotAdmin(t *testing.T) {
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
	RemoveFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "You are not allowed to remove file from this upload")
}

func TestRemoveRemovedFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	data := "data"

	upload := common.NewUpload()
	upload.Create()

	file1 := upload.NewFile()
	file1.Name = "file1"
	file1.Status = "removed"

	err := createTestFile(ctx, upload, file1, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 1")

	createTestUpload(ctx, upload)

	ctx.Set("upload", upload)
	ctx.Set("file", file1)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusNotFound, "File file1 has already been removed")
}

func TestRemoveLastFile(t *testing.T) {
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
	ctx.Set("file", file1)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var uploadResult = &common.Upload{}
	err = json.Unmarshal(respBody, uploadResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, 1, len(uploadResult.Files), "invalid upload files count")
	require.Equal(t, "removed", uploadResult.Files[file1.ID].Status, "invalid removed file status")

	_, err = context.GetMetadataBackend(ctx).Get(ctx, upload.ID)
	require.Error(t, err, "removed upload still exists")

	_, err = context.GetDataBackend(ctx).GetFile(ctx, upload, file1.ID)
	require.Error(t, err, "removed file still exists")
}

func TestRemoveFileNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestRemoveFileNoFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	ctx.Set("upload", upload)

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestRemoveFileMetadataBackendError(t *testing.T) {
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
	ctx.Set("file", file1)

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to update upload metadata")
}

func TestRemoveFileDataBackendError(t *testing.T) {
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
	ctx.Set("file", file1)

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	context.GetDataBackend(ctx).(*data_test.Backend).SetError(errors.New("data backend error"))

	rr := httptest.NewRecorder()
	RemoveFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to delete file")
}
