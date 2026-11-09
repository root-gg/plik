package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/iotest"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func newResumableTestUpload(t *testing.T, ctx *context.Context) (upload *common.Upload, file *common.File) {
	ctx.GetConfig().ChunkDirectory = t.TempDir()

	upload = &common.Upload{IsAdmin: true}
	file = upload.NewFile()
	file.Name = "file"
	createTestUpload(t, ctx, upload)
	return upload, file
}

func appendFile(t *testing.T, ctx *context.Context, upload *common.Upload, file *common.File, offset int, complete bool, body io.Reader) *httptest.ResponseRecorder {
	req, err := http.NewRequest("PATCH", "/resumable/"+upload.ID+"/"+file.ID+"/"+file.Name, body)
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("Content-Type", "application/partial-upload")
	req.Header.Set("Upload-Offset", strconv.Itoa(offset))
	if complete {
		req.Header.Set("Upload-Complete", "?1")
	} else {
		req.Header.Set("Upload-Complete", "?0")
	}

	ctx.SetFile(file)
	rr := ctx.NewRecorder(req)
	AppendFile(ctx, rr, req)
	return rr
}

func getFileOffset(t *testing.T, ctx *context.Context, upload *common.Upload, file *common.File) *httptest.ResponseRecorder {
	req, err := http.NewRequest("HEAD", "/resumable/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err, "unable to create new request")

	ctx.SetFile(file)
	rr := ctx.NewRecorder(req)
	GetFileOffset(ctx, rr, req)
	return rr
}

func addResumableFile(t *testing.T, ctx *context.Context, upload *common.Upload, complete bool, body io.Reader) *httptest.ResponseRecorder {
	req, err := http.NewRequest("POST", "/resumable/"+upload.ID+"/file?reference=ref", body)
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("Upload-Length", strconv.Itoa(len(content)))
	if complete {
		req.Header.Set("Upload-Complete", "?1")
	} else {
		req.Header.Set("Upload-Complete", "?0")
	}
	req = mux.SetURLVars(req, map[string]string{"uploadID": upload.ID, "filename": "file"})

	rr := ctx.NewRecorder(req)
	AddResumableFile(ctx, rr, req)
	return rr
}

func requireUploadedFile(t *testing.T, ctx *context.Context, rr *httptest.ResponseRecorder, file *common.File) {
	context.TestOK(t, rr)
	require.Equal(t, "?1", rr.Header().Get("Upload-Complete"), "upload should be complete")

	var fileResult = &common.File{}
	err := json.NewDecoder(rr.Body).Decode(fileResult)
	require.NoError(t, err, "unable to unmarshal response body")
	require.Equal(t, common.FileUploaded, fileResult.Status, "invalid file status")
	require.Equal(t, contentMD5, fileResult.Md5, "invalid file md5")
	require.Equal(t, int64(len(content)), fileResult.Size, "invalid file size")

	reader, err := ctx.GetDataBackend().GetFile(file)
	require.NoError(t, err, "unable to get file from data backend")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, content, string(data), "invalid file content")
}

func TestAppendFileResumeAfterInterruption(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFileSize = 1000
	upload, file := newResumableTestUpload(t, ctx)

	rr := getFileOffset(t, ctx, upload, file)
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")
	require.Equal(t, "0", rr.Header().Get("Upload-Offset"), "invalid offset")
	require.Equal(t, "?0", rr.Header().Get("Upload-Complete"), "upload should not be complete")
	require.Equal(t, "max-size=1000", rr.Header().Get("Upload-Limit"), "invalid upload limit")
	require.Equal(t, "no-store", rr.Header().Get("Cache-Control"), "invalid cache control")

	rr = appendFile(t, ctx, upload, file, 0, false, bytes.NewBufferString(content[:5]))
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")
	require.Equal(t, "5", rr.Header().Get("Upload-Offset"), "invalid offset")

	// The connection drops in the middle of the next append
	rr = appendFile(t, ctx, upload, file, 5, true, io.MultiReader(bytes.NewBufferString(content[5:7]), iotest.ErrReader(errors.New("connection lost"))))
	require.Equal(t, http.StatusInternalServerError, rr.Code, "invalid status code")
	require.Equal(t, "?0", rr.Header().Get("Upload-Complete"), "upload should not be complete")

	// The client asks where to resume
	rr = getFileOffset(t, ctx, upload, file)
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")
	require.Equal(t, "5", rr.Header().Get("Upload-Offset"), "invalid offset")

	rr = appendFile(t, ctx, upload, file, 5, false, bytes.NewBufferString(content[5:]))
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")

	// Every byte is staged, an empty append completes the upload
	rr = getFileOffset(t, ctx, upload, file)
	require.Equal(t, strconv.Itoa(len(content)), rr.Header().Get("Upload-Offset"), "invalid offset")
	require.Equal(t, "?0", rr.Header().Get("Upload-Complete"), "upload should not be complete")

	rr = appendFile(t, ctx, upload, file, len(content), true, http.NoBody)
	requireUploadedFile(t, ctx, rr, file)

	_, err := PurgeStaleChunks(ctx.GetConfig().ChunkDirectory, ctx.GetMetadataBackend())
	require.NoError(t, err, "unable to clean completed upload")
	entries, err := os.ReadDir(ctx.GetConfig().ChunkDirectory)
	require.NoError(t, err, "unable to read chunk directory")
	require.Empty(t, entries, "staged data left behind")
}

func TestAppendFileLostResponses(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	upload, file := newResumableTestUpload(t, ctx)

	rr := appendFile(t, ctx, upload, file, 0, false, bytes.NewBufferString(content[:5]))
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")

	// The response was lost and the client sends the same data again
	rr = appendFile(t, ctx, upload, file, 0, false, bytes.NewBufferString(content[:5]))
	require.Equal(t, http.StatusConflict, rr.Code, "invalid status code")
	require.Equal(t, "5", rr.Header().Get("Upload-Offset"), "invalid offset")
	require.Equal(t, "?0", rr.Header().Get("Upload-Complete"), "upload should not be complete")
	require.Equal(t, "application/problem+json", rr.Header().Get("Content-Type"), "invalid content type")

	var problem map[string]any
	err := json.NewDecoder(rr.Body).Decode(&problem)
	require.NoError(t, err, "unable to unmarshal problem")
	require.Equal(t, "https://iana.org/assignments/http-problem-types#mismatching-upload-offset", problem["type"], "invalid problem type")
	require.Equal(t, float64(5), problem["expected-offset"], "invalid expected offset")
	require.Equal(t, float64(0), problem["provided-offset"], "invalid provided offset")

	rr = appendFile(t, ctx, upload, file, 5, true, bytes.NewBufferString(content[5:]))
	requireUploadedFile(t, ctx, rr, file)

	// The response completing the upload was lost too
	rr = getFileOffset(t, ctx, upload, file)
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")
	require.Equal(t, "?1", rr.Header().Get("Upload-Complete"), "upload should be complete")
	require.Equal(t, strconv.Itoa(len(content)), rr.Header().Get("Upload-Offset"), "invalid offset")

	rr = appendFile(t, ctx, upload, file, len(content), true, http.NoBody)
	requireUploadedFile(t, ctx, rr, file)
}

func TestAppendFileStatusRemoved(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	upload, file := newResumableTestUpload(t, ctx)

	rr := appendFile(t, ctx, upload, file, 0, false, bytes.NewBufferString(content[:5]))
	require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")

	file, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err, "unable to get file")
	err = ctx.GetMetadataBackend().RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	rr = appendFile(t, ctx, upload, file, 5, true, bytes.NewBufferString(content[5:]))
	context.TestBadRequest(t, rr, "invalid file status removed")
}

func TestAppendFileInvalidHeaders(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	upload, file := newResumableTestUpload(t, ctx)

	for name, test := range map[string]struct {
		header  string
		value   string
		status  int
		message string
	}{
		"missing offset":     {"Upload-Offset", "", http.StatusBadRequest, "invalid Upload-Offset header"},
		"negative offset":    {"Upload-Offset", "-1", http.StatusBadRequest, "invalid Upload-Offset header"},
		"missing complete":   {"Upload-Complete", "", http.StatusBadRequest, "invalid Upload-Complete header"},
		"invalid complete":   {"Upload-Complete", "true", http.StatusBadRequest, "invalid Upload-Complete header"},
		"other content type": {"Content-Type", "application/octet-stream", http.StatusUnsupportedMediaType, "application/partial-upload"},
		"content coding":     {"Content-Encoding", "gzip", http.StatusUnsupportedMediaType, "application/partial-upload"},
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest("PATCH", "/resumable/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBufferString(content))
			require.NoError(t, err, "unable to create new request")
			req.Header.Set("Content-Type", "application/partial-upload")
			req.Header.Set("Upload-Offset", "0")
			req.Header.Set("Upload-Complete", "?1")
			req.Header.Set(test.header, test.value)

			rr := ctx.NewRecorder(req)
			AppendFile(ctx, rr, req)
			context.TestFail(t, rr, test.status, test.message)
		})
	}
}

func TestAppendFileTooBig(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFileSize = 5
	upload, file := newResumableTestUpload(t, ctx)

	rr := appendFile(t, ctx, upload, file, 0, true, bytes.NewBufferString(content))
	context.TestBadRequest(t, rr, "file too big")
	require.Equal(t, "max-size=5", rr.Header().Get("Upload-Limit"), "invalid upload limit")
}

func TestAppendFileStream(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload := &common.Upload{IsAdmin: true, Stream: true}
	file := upload.NewFile()
	file.Name = "file"
	createTestUpload(t, ctx, upload)

	rr := appendFile(t, ctx, upload, file, 0, true, bytes.NewBufferString(content))
	context.TestBadRequest(t, rr, "not available for stream uploads")
}

func TestAppendFileNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	upload, file := newResumableTestUpload(t, ctx)
	upload.IsAdmin = false

	rr := appendFile(t, ctx, upload, file, 0, true, bytes.NewBufferString(content))
	context.TestForbidden(t, rr, "you are not allowed to add file to this upload")
}

func TestPurgeStaleChunks(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	upload, abandoned := newResumableTestUpload(t, ctx)
	active := upload.NewFile()
	active.Name = "active"
	removed := upload.NewFile()
	removed.Name = "removed"
	for _, file := range []*common.File{active, removed} {
		file.Status = common.FileMissing
		err := ctx.GetMetadataBackend().CreateFile(file)
		require.NoError(t, err, "unable to create file")
	}

	for _, file := range []*common.File{abandoned, active, removed} {
		rr := appendFile(t, ctx, upload, file, 0, false, bytes.NewBufferString(content[:5]))
		require.Equal(t, http.StatusNoContent, rr.Code, "invalid status code")
	}

	dayAgo := time.Now().Add(-25 * time.Hour)
	err := os.Chtimes(filepath.Join(ctx.GetConfig().ChunkDirectory, abandoned.ID), dayAgo, dayAgo)
	require.NoError(t, err, "unable to age staged file")

	removed, err = ctx.GetMetadataBackend().GetFile(removed.ID)
	require.NoError(t, err, "unable to get file")
	err = ctx.GetMetadataBackend().RemoveFile(removed)
	require.NoError(t, err, "unable to remove file")

	purged, err := PurgeStaleChunks(ctx.GetConfig().ChunkDirectory, ctx.GetMetadataBackend())
	require.NoError(t, err, "unable to purge stale chunks")
	require.Equal(t, 2, purged, "invalid purged count")

	rr := getFileOffset(t, ctx, upload, active)
	require.Equal(t, "5", rr.Header().Get("Upload-Offset"), "active upload should be kept")
	rr = appendFile(t, ctx, upload, active, 5, true, bytes.NewBufferString(content[5:]))
	requireUploadedFile(t, ctx, rr, active)

	file, err := ctx.GetMetadataBackend().GetFile(abandoned.ID)
	require.NoError(t, err, "unable to get file")
	require.Equal(t, common.FileRemoved, file.Status, "abandoned file should be removed")
}

func TestAddResumableFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().ChunkDirectory = t.TempDir()
	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	// The creation carries the beginning of the file
	rr := addResumableFile(t, ctx, upload, false, bytes.NewBufferString(content[:5]))
	require.Equal(t, http.StatusCreated, rr.Code, "invalid status code")
	require.Equal(t, "?0", rr.Header().Get("Upload-Complete"), "upload should not be complete")
	require.Equal(t, "5", rr.Header().Get("Upload-Offset"), "invalid offset")
	location := rr.Header().Get("Location")
	require.Regexp(t, "^/resumable/"+upload.ID+"/[a-zA-Z0-9]+/file$", location, "invalid location")

	files, err := ctx.GetMetadataBackend().GetFiles(upload.ID)
	require.NoError(t, err, "unable to get upload files")
	require.Len(t, files, 1, "invalid file count")
	file := files[0]
	require.Equal(t, "/resumable/"+upload.ID+"/"+file.ID+"/file", location, "location should point to the new file")

	// The response was lost and the client creates the file again
	rr = addResumableFile(t, ctx, upload, false, http.NoBody)
	require.Equal(t, http.StatusConflict, rr.Code, "invalid status code")
	require.Equal(t, location, rr.Header().Get("Location"), "a retried creation should return the same file")
	require.Equal(t, "5", rr.Header().Get("Upload-Offset"), "invalid offset")

	count, err := ctx.GetMetadataBackend().CountUploadFiles(upload.ID)
	require.NoError(t, err, "unable to count upload files")
	require.Equal(t, 1, count, "a retried creation should not add a file")

	rr = appendFile(t, ctx, upload, file, 5, true, bytes.NewBufferString(content[5:]))
	requireUploadedFile(t, ctx, rr, file)
}

func TestAddResumableFileInOneRequest(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().ChunkDirectory = t.TempDir()
	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	rr := addResumableFile(t, ctx, upload, true, bytes.NewBufferString(content))
	files, err := ctx.GetMetadataBackend().GetFiles(upload.ID)
	require.NoError(t, err, "unable to get upload files")
	require.Len(t, files, 1, "invalid file count")
	requireUploadedFile(t, ctx, rr, files[0])
}

func TestAddResumableFileTooBig(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFileSize = 5
	upload := &common.Upload{IsAdmin: true}
	createTestUpload(t, ctx, upload)

	rr := addResumableFile(t, ctx, upload, false, http.NoBody)
	context.TestBadRequest(t, rr, "file is too big")
}

func TestAddResumableFileTooManyFiles(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().MaxFilePerUpload = 1
	upload := &common.Upload{IsAdmin: true}
	upload.NewFile().Name = "other"
	createTestUpload(t, ctx, upload)

	rr := addResumableFile(t, ctx, upload, false, http.NoBody)
	context.TestBadRequest(t, rr, "maximum number file per upload reached")
}

func TestAddResumableFileNotAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	upload := &common.Upload{}
	createTestUpload(t, ctx, upload)

	rr := addResumableFile(t, ctx, upload, true, bytes.NewBufferString(content))
	context.TestForbidden(t, rr, "you are not allowed to add file to this upload")
}
