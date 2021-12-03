package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
)

func createTestFile(ctx *context.Context, file *common.File, reader io.Reader) (err error) {
	dataBackend := ctx.GetDataBackend()
	err = dataBackend.AddFile(file, reader)
	return err
}

func TestGetFile(t *testing.T) {
	config := common.NewConfiguration()
	config.EnhancedWebSecurity = true
	ctx := newTestingContext(config)

	data := "data"

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Md5 = "12345"
	file.Type = "type"
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name+"?dl=true", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.URL.Query().Set("dl", "true")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Equal(t, file.Type, rr.Header().Get("Content-Type"), "invalid response content type")
	require.Equal(t, strconv.Itoa(int(file.Size)), rr.Header().Get("Content-Length"), "invalid response content length")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, data, string(respBody), "invalid file content")
	require.NotEmpty(t, rr.Header().Get("X-Content-Type-Options"))
	require.NotEmpty(t, rr.Header().Get("X-XSS-Protection"))
	require.NotEmpty(t, rr.Header().Get("X-Frame-Options"))
	require.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	require.Equal(t, rr.Header().Get("Content-Disposition"), fmt.Sprintf(`attachement; filename="%s"`, file.Name))
}

func TestGetOneShotFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	upload.InitializeForTests()
	upload.OneShot = true
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(t, ctx, upload)

	data := "data"
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, data, string(respBody), "invalid file content")

	require.NotEmpty(t, rr.Header().Get("Cache-Control"))
	require.NotEmpty(t, rr.Header().Get("Pragma"))
	require.NotEmpty(t, rr.Header().Get("Expires"))

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err, "unable to get file metadata")
	require.Equal(t, common.FileRemoved, f.Status, "invalid file status")
}

func TestGetStreamingFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := data_test.NewBackend()
	ctx.SetDataBackend(backend)
	ctx.SetStreamBackend(backend)

	upload := &common.Upload{Stream: true}
	upload.InitializeForTests()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploading
	createTestUpload(t, ctx, upload)

	data := "data"
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, data, string(respBody), "invalid file content")

	require.NotEmpty(t, rr.Header().Get("Cache-Control"))
	require.NotEmpty(t, rr.Header().Get("Pragma"))
	require.NotEmpty(t, rr.Header().Get("Expires"))
}

func TestGetRemovedFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileRemoved
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	context.TestNotFound(t, rr, fmt.Sprintf("file %s (%s) is not available : removed", file.Name, file.ID))
}

func TestGetDeletedFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileDeleted
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	context.TestNotFound(t, rr, fmt.Sprintf("file %s (%s) is not available : deleted", file.Name, file.ID))
}

func TestGetFileInvalidDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)
	config.DownloadDomain = "http://download.domain"

	err := config.Initialize()
	require.NoError(t, err, "Unable to initialize config")

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	require.Equal(t, 301, rr.Code, "handler returned wrong status code")
}

func TestGetFileMissingUpload(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	context.TestPanic(t, rr, "missing upload from context", func() {
		GetFile(ctx, rr, req)
	})
}

func TestGetFileMissingFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)
	ctx.SetUpload(&common.Upload{})

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	context.TestPanic(t, rr, "missing file from context", func() {
		GetFile(ctx, rr, req)
	})
}

func TestGetHtmlFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Type = "html"
	file.Status = "uploaded"
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Equal(t, "text/plain", rr.Header().Get("Content-Type"), "invalid content type")
}

func TestGetFileNoType(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Status = "uploaded"
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "invalid content type")
}

func TestGetFileDataBackendError(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	ctx.GetDataBackend().(*data_test.Backend).SetError(errors.New("data backend error"))
	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestInternalServerError(t, rr, "unable to get file from data backend : data backend error")
}

func TestGetFileInvalidStatus(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileMissing
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	ctx.GetDataBackend().(*data_test.Backend).SetError(errors.New("data backend error"))
	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestNotFound(t, rr, "is not available")
}

func TestGetFileInvalidStatusStreaming(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{Stream: true}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileMissing
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("data")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	ctx.GetDataBackend().(*data_test.Backend).SetError(errors.New("data backend error"))
	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestNotFound(t, rr, "is not available")
}
