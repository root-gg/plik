package handlers

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
)

func TestGetArchive(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	file.Md5 = "12345"
	file.Type = "type"
	file.Size = int64(len(data))

	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestOK(t, rr)

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

func TestGetArchiveStreaming(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{Stream: true}
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestBadRequest(t, rr, "archive feature is not available in stream mode")
}

func TestGetArchiveNoFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestBadRequest(t, rr, "nothing to archive")
}

func TestGetArchiveFileNameTooLong(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	ctx.SetUpload(upload)

	archiveName := ""
	for i := 0; i < 10240; i++ {
		archiveName += "x"
	}
	archiveName += ".zip"

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+archiveName, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": archiveName,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestBadRequest(t, rr, "archive name too long")
}

func TestGetArchiveInvalidDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)
	config.DownloadDomain = "http://download.domain"

	err := config.Initialize()
	require.NoError(t, err, "Unable to initialize config")

	req, err := http.NewRequest("GET", "/archive/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)
	require.Equal(t, 301, rr.Code, "handler returned wrong status code")
}

func TestGetArchiveMissingUpload(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	req, err := http.NewRequest("GET", "/archive/", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	context.TestPanic(t, rr, "missing upload from context", func() {
		GetArchive(ctx, rr, req)
	})
}

func TestGetArchiveOneShot(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"
	upload := &common.Upload{}
	upload.OneShot = true
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestOK(t, rr)

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

	file, err = ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err, "get file error")
	require.Equal(t, common.FileRemoved, file.Status, "get file error")

}

func TestGetArchiveNoArchiveName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"
	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestBadRequest(t, rr, "missing archive name")
}

func TestGetArchiveInvalidArchiveName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"
	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.tar",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestBadRequest(t, rr, "invalid archive name, missing .zip extension")
}

func TestGetArchiveDataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"
	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = "uploaded"
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	ctx.GetDataBackend().(*data_test.Backend).SetError(errors.New("data backend error"))

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)
	context.TestInternalServerError(t, rr, "unable to get file from data backend : data backend error")
}

//
//func TestGetArchiveMetadataBackendError(t *testing.T) {
//	ctx := newTestingContext(common.NewConfiguration())
//
//	data := "data"
//	upload := &common.Upload{}
//	upload.OneShot = true
//	file := upload.NewFile()
//	file.Name = "file"
//	file.Status = "uploaded"
//	createTestUpload(t, ctx, upload)
//
//	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
//	require.NoError(t, err, "unable to create test file")
//
//	ctx.SetUpload(upload)
//
//	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
//	require.NoError(t, err, "unable to create new request")
//
//	// Fake gorilla/mux vars
//	vars := map[string]string{
//		"filename": "archive.zip",
//	}
//	req = mux.SetURLVars(req, vars)
//
//	ctx.GetMetadataBackend().(*metadata_test.Backend).SetError(errors.New("metadata backend error"))
//
//	rr := ctx.NewRecorder(req)
//	GetArchive(ctx, rr, req)
//	context.TestInternalServerError(t, rr, "unable to update upload metadata : metadata backend error")
//}
