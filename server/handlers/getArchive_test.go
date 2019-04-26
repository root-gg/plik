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
	"archive/zip"
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	metadata_test "github.com/root-gg/plik/server/metadata/testing"
	"github.com/stretchr/testify/require"
)

func TestGetArchive(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

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

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	require.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "invalid response content type")
	require.Equal(t, "", rr.Header().Get("Content-Length"), "invalid response content length")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	z, err := zip.NewReader(bytes.NewReader(respBody), int64(len(respBody)))
	require.NoError(t, err, "unable to unzip response body")

	require.Equal(t, len(upload.Files), len(z.File), "invalid archive file count")
	require.Equal(t, file.Name, z.File[0].Name, "invalid archived file name")

	fileReader, err := z.File[0].Open()
	require.NoError(t, err, "unable to open archived file")

	content, err := ioutil.ReadAll(fileReader)
	require.NoError(t, err, "unable to read archived file")
	require.Equal(t, data, string(content), "invalid archived file content")
}

func TestGetArchiveNoFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Nothing to archive")
}

func TestGetArchiveInvalidDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)
	config.DownloadDomain = "http://download.domain"

	err := config.Initialize()
	require.NoError(t, err, "Unable to initialize config")

	req, err := http.NewRequest("GET", "/archive/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)
	require.Equal(t, 301, rr.Code, "handler returned wrong status code")
}

func TestGetArchiveMissingUpload(t *testing.T) {
	config := common.NewConfiguration()
	ctx := context.NewTestingContext(config)

	req, err := http.NewRequest("GET", "/archive/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)
	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestGetArchiveOneShot(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	data := "data"
	upload := common.NewUpload()
	upload.OneShot = true
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	require.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "invalid response content type")
	require.Equal(t, "", rr.Header().Get("Content-Length"), "invalid response content length")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	z, err := zip.NewReader(bytes.NewReader(respBody), int64(len(respBody)))
	require.NoError(t, err, "unable to unzip response body")

	require.Equal(t, len(upload.Files), len(z.File), "invalid archive file count")
	require.Equal(t, file.Name, z.File[0].Name, "invalid archived file name")

	fileReader, err := z.File[0].Open()
	require.NoError(t, err, "unable to open archived file")

	content, err := ioutil.ReadAll(fileReader)
	require.NoError(t, err, "unable to read archived file")
	require.Equal(t, data, string(content), "invalid archived file content")

	_, err = context.GetDataBackend(ctx).GetFile(ctx, upload, file.ID)
	require.Error(t, err, "downloaded file still exists")

	_, err = context.GetMetadataBackend(ctx).Get(ctx, upload.ID)
	require.Error(t, err, "downloaded upload still exists")
}

func TestGetArchiveNoArchiveName(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	data := "data"
	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing file name")
}

func TestGetArchiveInvalidArchiveName(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	data := "data"
	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.tar",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing .zip extension")
}

func TestGetArchiveDataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	data := "data"
	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	context.GetDataBackend(ctx).(*data_test.Backend).SetError(errors.New("data backend error"))

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Failed to read file")
}

func TestGetArchiveMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	data := "data"
	upload := common.NewUpload()
	upload.OneShot = true
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(ctx, upload)

	err := createTestFile(ctx, upload, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))

	rr := httptest.NewRecorder()
	GetArchive(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Nothing to archive")
}
