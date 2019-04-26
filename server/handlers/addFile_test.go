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
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
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

func getMultipartFormData(name string, in io.Reader) (out io.Reader, contentType string, err error) {
	buffer := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buffer)

	writer, err := multipartWriter.CreateFormFile("file", name)
	if err != nil {
		return nil, "", fmt.Errorf("Unable to create multipartWriter : %s", err)
	}

	_, err = io.Copy(writer, in)
	if err != nil {
		return nil, "", err
	}

	err = multipartWriter.Close()
	if err != nil {
		return nil, "", err
	}

	return buffer, multipartWriter.FormDataContentType(), nil
}

func TestAddFileWithID(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "file"

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	data := "data"
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	// Check the status code is what we expect.
	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, file.ID, fileResult.ID, "invalid file id")
	require.Equal(t, file.Name, fileResult.Name, "invalid file name")
	require.Equal(t, file.Md5, fileResult.Md5, "invalid file md5")
	require.Equal(t, file.Status, fileResult.Status, "invalid file status")
	require.Equal(t, "application/octet-stream", fileResult.Type, "invalid file type")
	require.Equal(t, int64(len(data)), fileResult.CurrentSize, "invalid file size")
}

func TestAddFileWithInvalidID(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()

	file := common.NewFile()
	file.Name = "file"

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	data := "data"
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "Invalid file id")
}

func TestAddFileWithoutID(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	name := "file"
	data := "data"
	reader, contentType, err := getMultipartFormData(name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var fileResult = &common.File{}
	err = json.Unmarshal(respBody, fileResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEqual(t, "", fileResult.ID, "invalid file id")
	require.NotEqual(t, "", fileResult.Md5, "invalid file md5")
	require.Equal(t, name, fileResult.Name, "invalid file name")
	require.Equal(t, "uploaded", fileResult.Status, "invalid file status")
	require.Equal(t, "application/octet-stream", fileResult.Type, "invalid file type")
	require.Equal(t, int64(len(data)), fileResult.CurrentSize, "invalid file size")
}

func TestAddFileWithoutUploadInContext(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("POST", "/file/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestAddFileWithoutAnonymousUploads(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).NoAnonymousUploads = true
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	req, err := http.NewRequest("POST", "/file/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Unable to add file from anonymous user")
}

func TestAddFileNotAdmin(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	req, err := http.NewRequest("POST", "/file/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "You are not allowed to add file to this upload")
}

func TestAddFileTooManyFiles(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).MaxFilePerUpload = 2
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()

	for i := 0; i < 5; i++ {
		upload.NewFile()
	}

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	req, err := http.NewRequest("POST", "/file/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusForbidden, "Maximum number file per upload reached")
}

func TestAddFileInvalidMultipartData(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	req, err := http.NewRequest("POST", "/file/"+upload.ID, bytes.NewBuffer([]byte("invalid multipart data")))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Failed to get file from multipart request")
}

func TestAddFileWithFilenameTooLong(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()

	file := upload.NewFile()
	name := make([]byte, 2000)
	for i := range name {
		name[i] = 'x'
	}
	file.Name = string(name)

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	data := "data"
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "File name is too long")
}

func TestAddFileWithNoFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	buffer := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(buffer)

	_, err := multipartWriter.CreateFormFile("invalid_form_field", "filename")
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Unable to read file")
}

func TestAddFileWithEmptyName(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	file := upload.NewFile()

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	data := "data"
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing file name from multipart request")
}

func TestAddFileWithDataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetDataBackend(ctx).(*data_test.Backend).SetError(errors.New("data backend error"))
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "name"

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	data := "data"
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to save file")
}

func TestAddFileWithMetadataBackendError(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetMetadataBackend(ctx).(*metadata_test.MetadataBackend).SetError(errors.New("metadata backend error"))
	ctx.Set("is_upload_admin", true)

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "name"

	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	data := "data"
	reader, contentType, err := getMultipartFormData(file.Name, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable get multipart form data")

	req, err := http.NewRequest("POST", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, reader)
	require.NoError(t, err, "unable to create new request")

	req.Header.Set("Content-Type", contentType)

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": file.ID,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	AddFile(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to update upload metadata")
}
