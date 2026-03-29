package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Equal(t, file.Type, rr.Header().Get("Content-Type"), "invalid response content type")
	require.Equal(t, strconv.Itoa(int(file.Size)), rr.Header().Get("Content-Length"), "invalid response content length")

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, data, string(respBody), "invalid file content")
	require.NotEmpty(t, rr.Header().Get("X-Content-Type-Options"))
	require.NotEmpty(t, rr.Header().Get("X-XSS-Protection"))
	require.NotEmpty(t, rr.Header().Get("X-Frame-Options"))
	require.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	require.Equal(t, rr.Header().Get("Content-Disposition"), fmt.Sprintf(`attachment; filename="%s"`, file.Name))
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

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, data, string(respBody), "invalid file content")

	require.NotEmpty(t, rr.Header().Get("Cache-Control"))
	require.NotEmpty(t, rr.Header().Get("Pragma"))
	require.NotEmpty(t, rr.Header().Get("Expires"))

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err, "unable to get file metadata")
	require.Equal(t, common.FileDeleted, f.Status, "invalid file status")
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

	respBody, err := io.ReadAll(rr.Body)
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

	err := config.Initialize(nil)
	require.NoError(t, err, "Unable to initialize config")

	req, err := http.NewRequest("GET", "/file/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.Host = "invalid.domain"

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestBadRequest(t, rr, "Invalid download domain invalid.domain")
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

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "HTML files should be neutralized to octet-stream")
}

func TestGetSvgFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Type = "image/svg+xml"
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

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "SVG files should be neutralized to octet-stream")
}

func TestGetXmlFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Type = "text/xml"
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

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "XML files should be neutralized to octet-stream")
}

func TestGetJavascriptFile(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.InitializeForTests()

	file := upload.NewFile()
	file.Type = "application/javascript"
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

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "JS files should be neutralized to octet-stream")
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

func TestGetFileE2EERedirectWebapp(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{E2EE: "age"}
	upload.InitializeForTests()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("encrypted")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("X-ClientApp", "web_client")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	require.Equal(t, http.StatusTemporaryRedirect, rr.Code, "expected redirect for webapp E2EE download")
	require.Contains(t, rr.Header().Get("Location"), "/#/?id="+upload.ID, "redirect should point to download page")
}

func TestGetFileE2EERedirectWebappWithPath(t *testing.T) {
	config := common.NewConfiguration()
	config.Path = "/sub"
	ctx := newTestingContext(config)

	upload := &common.Upload{E2EE: "age"}
	upload.InitializeForTests()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("encrypted")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("X-ClientApp", "web_client")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	require.Equal(t, http.StatusTemporaryRedirect, rr.Code, "expected redirect for webapp E2EE download")
	require.Equal(t, "/sub/#/?id="+upload.ID, rr.Header().Get("Location"), "E2EE redirect should include the configured Path prefix")
}

func TestGetFileE2EEContentType(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{E2EE: "age"}
	upload.InitializeForTests()
	file := upload.NewFile()
	file.Name = "document.pdf"
	file.Type = "application/pdf"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte("encrypted")))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	// Non-webapp request (e.g. curl) — should get raw bytes with octet-stream
	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"), "E2EE files should be served as octet-stream")

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, "encrypted", string(respBody), "should receive raw encrypted bytes")
}

func TestGetFileE2EENonWebappPassthrough(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	upload := &common.Upload{E2EE: "age"}
	upload.InitializeForTests()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	data := "encrypted-content"
	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	// Request without X-ClientApp header (e.g. curl) — should NOT redirect
	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, data, string(respBody), "CLI should receive raw encrypted bytes")
}
