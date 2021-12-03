package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestFileNoUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)

	File(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestInternalServerError(t, rr, "missing upload from context")
}

func TestFileNoFileID(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUpload(&common.Upload{})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestMissingParameter(t, rr, "file ID")
}

func TestFileNoFileName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUpload(&common.Upload{})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": "fileID",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestMissingParameter(t, rr, "file name")
}

func TestFileNotFound(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetUpload(&common.Upload{})

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   "fileID",
		"filename": "filename",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestNotFound(t, rr, "file fileID not found")
}

func TestFileInvalidFileName(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "filename"
	ctx.SetUpload(upload)

	upload.InitializeForTests()
	err := ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err, "create upload error")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   file.ID,
		"filename": "invalid_file_name",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestInvalidParameter(t, rr, "file name")
}

func TestFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "filename"
	ctx.SetUpload(upload)

	upload.InitializeForTests()
	err := ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err, "create upload error")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   file.ID,
		"filename": file.Name,
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	f := ctx.GetFile()
	require.NotNil(t, f, "missing file from context")
	require.Equal(t, file.ID, f.ID, "invalid file from context")
}

func TestFileMetadataBackendError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "filename"
	ctx.SetUpload(upload)

	upload.InitializeForTests()
	err := ctx.GetMetadataBackend().CreateUpload(upload)
	require.NoError(t, err, "create upload error")

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   file.ID,
		"filename": file.Name,
	}
	req = mux.SetURLVars(req, vars)

	err = ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err, "unable to shutdown metadata backend")

	rr := ctx.NewRecorder(req)
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestInternalServerError(t, rr, "database is closed")
}
