package handlers

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestRemoveFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"

	upload := &common.Upload{IsAdmin: true}
	file1 := upload.NewFile()
	file1.Name = "file1"
	file1.Status = "uploaded"

	upload.InitializeForTests()

	err := createTestFile(ctx, file1, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 1")

	file2 := upload.NewFile()
	file2.Name = "file2"
	file2.Status = "uploaded"

	err = createTestFile(ctx, file2, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 2")

	createTestUpload(t, ctx, upload)

	ctx.SetUpload(upload)
	ctx.SetFile(file1)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, "ok", string(respBody))

	file1, err = ctx.GetMetadataBackend().GetFile(file1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(upload.Files), "invalid upload files count")
	require.Equal(t, common.FileRemoved, upload.GetFile(file1.ID).Status, "invalid removed file status")
}

func TestRemoveFileNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"

	upload := &common.Upload{}
	file1 := upload.NewFile()
	file1.Name = "file1"
	file1.Status = "uploaded"
	upload.InitializeForTests()

	err := createTestFile(ctx, file1, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 1")

	createTestUpload(t, ctx, upload)

	ctx.SetUpload(upload)
	ctx.SetFile(file1)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveFile(ctx, rr, req)

	context.TestForbidden(t, rr, "you are not allowed to remove files from this upload")
}

func TestRemoveRemovedFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"

	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file1"
	file.Status = common.FileRemoved
	upload.InitializeForTests()

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file 1")

	createTestUpload(t, ctx, upload)

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveFile(ctx, rr, req)
	context.TestOK(t, rr)
}

//func TestRemoveLastFile(t *testing.T) {
//	ctx := newTestingContext(common.NewConfiguration())
//	ctx.SetUploadAdmin(true)
//
//	data := "data"
//
//	upload := &common.Upload{}
//	upload.Create()
//
//	file1 := upload.NewFile()
//	file1.Name = "file1"
//	file1.Status = "uploaded"
//
//	err := createTestFile(ctx, upload, file1, bytes.NewBuffer([]byte(data)))
//	require.NoError(t, err, "unable to create test file 1")
//
//	createTestUpload(t, ctx, upload)
//
//	ctx.SetUpload(upload)
//	ctx.SetFile(file1)
//
//	req, err := http.NewRequest("DELETE", "/file/"+upload.ID+"/"+file1.ID+"/"+file1.Name, bytes.NewBuffer([]byte{}))
//	require.NoError(t, err, "unable to create new request")
//
//	rr := ctx.NewRecorder(req)
//	RemoveFile(ctx, rr, req)
//	context.TestOK(t, rr)
//
//	respBody, err := ioutil.ReadAll(rr.Body)
//	require.NoError(t, err, "unable to read response body")
//	require.Equal(t, "ok", string(respBody))
//
//	u, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
//	require.NoError(t, err, "removed upload still exists")
//	require.Nil(t, u, "removed upload still exists")
//
//	_, err = ctx.GetDataBackend().GetFile(upload, file1.ID)
//	require.Error(t, err, "removed file still exists")
//}

func TestRemoveFileNoUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveFile(ctx, rr, req)
	context.TestInternalServerError(t, rr, "missing upload from context")
}

func TestRemoveFileNoFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true}
	ctx.SetUpload(upload)

	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	RemoveFile(ctx, rr, req)
	context.TestInternalServerError(t, rr, "missing file from context")
}

//
//func TestRemoveFileMetadataBackendError(t *testing.T) {
//	ctx := newTestingContext(common.NewConfiguration())
//	ctx.SetUploadAdmin(true)
//
//	data := "data"
//
//	upload := &common.Upload{}
//	upload.Create()
//
//	file1 := upload.NewFile()
//	file1.Name = "file1"
//	file1.Status = "uploaded"
//
//	err := createTestFile(ctx, upload, file1, bytes.NewBuffer([]byte(data)))
//	require.NoError(t, err, "unable to create test file 1")
//
//	createTestUpload(t, ctx, upload)
//
//	ctx.SetUpload(upload)
//	ctx.SetFile(file1)
//
//	req, err := http.NewRequest("DELETE", "/file/uploadID/fileID/fileName", bytes.NewBuffer([]byte{}))
//	require.NoError(t, err, "unable to create new request")
//
//	ctx.GetMetadataBackend().(*metadata_test.Backend).SetError(errors.New("metadata backend error"))
//
//	rr := ctx.NewRecorder(req)
//	RemoveFile(ctx, rr, req)
//	context.TestInternalServerError(t, rr, "unable to update upload metadata : metadata backend error")
//}
