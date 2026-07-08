package handlers

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
	data_test "github.com/root-gg/plik/server/data/testing"
)

type failingArchiveBackend struct{}

func (*failingArchiveBackend) AddFile(_ *common.File, reader io.Reader) error {
	_, err := io.Copy(io.Discard, reader)
	return err
}

func (*failingArchiveBackend) GetFile(_ *common.File) (io.ReadSeekCloser, error) {
	return &failingArchiveReader{data: []byte("partial")}, nil
}

func (*failingArchiveBackend) RemoveFile(_ *common.File) error {
	return nil
}

type failingArchiveReader struct {
	data []byte
	off  int
}

func (r *failingArchiveReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, errors.New("read error")
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, errors.New("read error")
}

func (r *failingArchiveReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.off = int(offset)
	case io.SeekCurrent:
		r.off += int(offset)
	case io.SeekEnd:
		r.off = len(r.data) + int(offset)
	}
	return int64(r.off), nil
}

func (*failingArchiveReader) Close() error {
	return nil
}

// secondFileGetFileErrorBackend wraps a real data backend and fails GetFile
// only for one designated file ID, so a multi-file archive can stream its
// first file successfully before a later file's GetFile call fails — pinning
// the archive handler's uniform bytes-only policy on its other internal-error
// early returns (GetFile / CreateHeader), which otherwise would drop
// already-served bytes exactly like the archiveFailed path used to before
// this policy applied uniformly.
type secondFileGetFileErrorBackend struct {
	data.Backend
	failFileID string
}

func (b *secondFileGetFileErrorBackend) GetFile(file *common.File) (io.ReadSeekCloser, error) {
	if file.ID == b.failFileID {
		return nil, errors.New("second file error")
	}
	return b.Backend.GetFile(file)
}

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

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	z, err := zip.NewReader(bytes.NewReader(respBody), int64(len(respBody)))
	require.NoError(t, err, "unable to unzip response body")

	require.Equal(t, len(upload.Files), len(z.File), "invalid archive file count")
	require.Equal(t, file.Name, z.File[0].Name, "invalid archived file name")
	require.Equal(t, zip.Deflate, z.File[0].Method, "archive should use zip.Deflate by default (EnableArchiveCompression=true)")

	fileReader, err := z.File[0].Open()
	require.NoError(t, err, "unable to open archived file")

	content, err := io.ReadAll(fileReader)
	require.NoError(t, err, "unable to read archived file")
	require.Equal(t, data, string(content), "invalid archived file content")

	gotUpload, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotUpload.DownloadCount)
	gotFile, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotFile.DownloadCount)

	// The whole-archive egress (zip stream size) is attributed to the upload
	// rollup; each included file records its download event with 0 bytes because
	// a single zip stream cannot be split across its files.
	uploadDownloads, uploadBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, found)
	require.Equal(t, int64(1), uploadDownloads)
	require.Equal(t, int64(len(respBody)), uploadBytes, "upload rollup bytes must equal the zip stream size served")

	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found)
	require.Equal(t, int64(1), fileDownloads, "each included file counts one event")
	require.Equal(t, int64(0), fileBytes, "per-file archive bytes are not attributable")
}

func TestGetArchiveCopyErrorDoesNotCount(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.SetDataBackend(&failingArchiveBackend{})

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = 7

	createTestUpload(t, ctx, upload)
	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req = mux.SetURLVars(req, map[string]string{"filename": "archive.zip"})

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	gotUpload, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), gotUpload.DownloadCount, "a mid-stream archive failure must not count a download event")
	require.Positive(t, gotUpload.DownloadedBytes, "a mid-stream archive failure must still record the bytes already streamed to the client")
	gotFile, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), gotFile.DownloadCount, "a mid-stream archive failure must not touch the file's download event")

	// A mid-stream failure still records its already-served bytes as a
	// bytes-only event (downloads=0) on the upload rollup, matching the
	// single-file egress policy; the per-file rollup stays untouched since
	// there is no event to attribute to it.
	uploadDownloads, uploadBytes, uploadFound := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, uploadFound, "failed archive must still record its upload rollup bytes")
	require.Equal(t, int64(0), uploadDownloads, "failed archive must not count an upload rollup download event")
	require.Positive(t, uploadBytes, "failed archive must record its already-served bytes on the upload rollup")
	_, _, fileFound := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.False(t, fileFound, "failed archive must not record a file rollup (event-gated, no event occurred)")
}

// TestGetArchiveSecondFileGetFileErrorRecordsFirstFileBytes extends the
// archiveFailed test above (TestGetArchiveCopyErrorDoesNotCount) to the
// handler's two OTHER internal-error early returns (GetFile / CreateHeader),
// which used to return immediately without recording anything. Here the
// first file streams fully, then the second file's GetFile call fails: the
// bytes already served to the client for the first file must still be
// recorded as a bytes-only event (no download event), matching the uniform
// "never drop already-served bytes" policy this handler now applies on every
// mid-archive failure path.
//
// Like TestGetArchiveCopyErrorDoesNotCount, this does not assert on the HTTP
// status code: once a prior file's bytes have actually reached the client the
// response has already committed its (200) status, so the later attempt to
// signal a 500 is a no-op superfluous write — an inherent HTTP streaming
// limitation, not something this handler can or should paper over.
func TestGetArchiveSecondFileGetFileErrorRecordsFirstFileBytes(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	firstData := "0123456789"
	upload := &common.Upload{}
	firstFile := upload.NewFile()
	firstFile.Name = "first"
	firstFile.Status = common.FileUploaded
	firstFile.Size = int64(len(firstData))
	secondFile := upload.NewFile()
	secondFile.Name = "second"
	secondFile.Status = common.FileUploaded
	secondFile.Size = 7

	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, firstFile, bytes.NewBufferString(firstData))
	require.NoError(t, err, "unable to create first test file")
	err = createTestFile(ctx, secondFile, bytes.NewBufferString("abcdefg"))
	require.NoError(t, err, "unable to create second test file")

	ctx.SetUpload(upload)
	ctx.SetDataBackend(&secondFileGetFileErrorBackend{Backend: ctx.GetDataBackend(), failFileID: secondFile.ID})

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req = mux.SetURLVars(req, map[string]string{"filename": "archive.zip"})

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	gotUpload, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), gotUpload.DownloadCount, "a mid-archive internal error must not count a download event")
	require.Positive(t, gotUpload.DownloadedBytes, "the first file's already-streamed bytes must still be recorded")

	uploadDownloads, uploadBytes, uploadFound := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, uploadFound, "a GetFile error on the second file must still record the upload rollup bytes")
	require.Equal(t, int64(0), uploadDownloads, "a GetFile error on the second file must not count an upload rollup download event")
	require.Positive(t, uploadBytes, "a GetFile error on the second file must record the first file's already-served bytes")
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

	var archiveName strings.Builder
	for range 10240 {
		archiveName.WriteString("x")
	}
	archiveName.WriteString(".zip")

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+archiveName.String(), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"filename": archiveName.String(),
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
	req.Host = "invalid.domain"

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)
	context.TestBadRequest(t, rr, "Invalid download domain invalid.domain")
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

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	z, err := zip.NewReader(bytes.NewReader(respBody), int64(len(respBody)))
	require.NoError(t, err, "unable to unzip response body")

	require.Equal(t, len(upload.Files), len(z.File), "invalid archive file count")
	require.Equal(t, file.Name, z.File[0].Name, "invalid archived file name")
	require.Equal(t, zip.Deflate, z.File[0].Method, "archive should use zip.Deflate by default (EnableArchiveCompression=true)")

	fileReader, err := z.File[0].Open()
	require.NoError(t, err, "unable to open archived file")

	content, err := io.ReadAll(fileReader)
	require.NoError(t, err, "unable to read archived file")
	require.Equal(t, data, string(content), "invalid archived file content")

	file, err = ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err, "get file error")
	require.Equal(t, common.FileRemoved, file.Status, "get file error")

}

func TestGetArchiveNoCompression(t *testing.T) {
	config := common.NewConfiguration()
	config.EnableArchiveCompression = false
	ctx := newTestingContext(config)

	data := "data data data data data data data" // repetitive data compresses well

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file.txt"
	file.Status = "uploaded"
	file.Size = int64(len(data))

	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)

	req, err := http.NewRequest("GET", "/archive/"+upload.ID+"/"+"archive.zip", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	vars := map[string]string{
		"filename": "archive.zip",
	}
	req = mux.SetURLVars(req, vars)

	rr := ctx.NewRecorder(req)
	GetArchive(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	z, err := zip.NewReader(bytes.NewReader(respBody), int64(len(respBody)))
	require.NoError(t, err, "unable to unzip response body")

	require.Equal(t, len(upload.Files), len(z.File), "invalid archive file count")
	require.Equal(t, file.Name, z.File[0].Name, "invalid archived file name")
	require.Equal(t, zip.Store, z.File[0].Method, "archive should use zip.Store when EnableArchiveCompression is false")

	fileReader, err := z.File[0].Open()
	require.NoError(t, err, "unable to open archived file")

	content, err := io.ReadAll(fileReader)
	require.NoError(t, err, "unable to read archived file")
	require.Equal(t, data, string(content), "invalid archived file content")
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
