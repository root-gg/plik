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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func createTestFile(ctx *juliet.Context, upload *common.Upload, file *common.File, reader io.Reader) (err error) {
	dataBackend := context.GetDataBackend(ctx)
	_, err = dataBackend.AddFile(ctx, upload, file, reader)
	return err
}

func TestGetFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	data := "data"

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	file.Md5 = "12345"
	file.Type = "type"
	file.CurrentSize = int64(len(data))
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	require.Equal(t, file.Type, rr.Header().Get("Content-Type"), "invalid response content type")
	require.Equal(t, strconv.Itoa(int(file.CurrentSize)), rr.Header().Get("Content-Length"), "invalid response content length")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, data, string(respBody), "invalid file content")
}

func TestGetOneShotFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	upload.OneShot = true
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(ctx, upload)

	data := "data"
	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, data, string(respBody), "invalid file content")

	rr = httptest.NewRecorder()
	GetFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "File file has already been downloaded")
}

func TestGetDownloadedFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	upload.OneShot = true
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "downloaded"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "File file has already been downloaded")
}

func TestGetRemovedFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "removed"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "File file has been removed")
}

func TestGetFileInvalidDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)
	config.DownloadDomain = "http://download.domain"

	err := config.Initialize()
	require.NoError(t, err, "Unable to initialize config")

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	require.Equal(t, 301, rr.Code, "handler returned wrong status code")
}

func TestGetFileMissingUpload(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestGetFileMissingFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)
	ctx.Set("upload", common.NewUpload())

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestGetHtmlFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	upload := common.NewUpload()
	upload.Create()

	file := upload.NewFile()
	file.Type = "html"
	file.Status = "uploaded"
	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	require.Equal(t, "text/plain", rr.Header().Get("Content-Type"), "invalid content type")
}

func TestGetFileNoType(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	upload := common.NewUpload()
	upload.Create()

	file := upload.NewFile()
	file.Status = "uploaded"
	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "invalid content type")
}

func TestGetFileDataBackendError(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	upload := common.NewUpload()
	upload.Create()

	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	context.GetDataBackend(ctx).(*data_test.Backend).SetError(errors.New("data backend error"))
	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	context.TestFail(t, rr, http.StatusNotFound, "Failed to read file file")
}

func TestGetFileMetadataBackendError(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	upload := common.NewUpload()
	upload.OneShot = true
	upload.Create()

	file := upload.NewFile()
	file.Status = "uploaded"
	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)
	ctx.Set("file", file)

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))
	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")
}
