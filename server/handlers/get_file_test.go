package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/middleware"
)

func createTestFile(ctx *context.Context, file *common.File, reader io.Reader) (err error) {
	dataBackend := ctx.GetDataBackend()
	err = dataBackend.AddFile(file, reader)
	return err
}

func TestGetFile(t *testing.T) {
	config := common.NewConfiguration()
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
	require.NotEmpty(t, rr.Header().Get("X-Frame-Options"))
	require.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	require.Equal(t, rr.Header().Get("Content-Disposition"), fmt.Sprintf(`attachment; filename="%s"`, file.Name))

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.DownloadCount)
	u, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), u.DownloadCount)
}

func TestGetFileDownloadStatsErrorDoesNotBlockDownload(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	data := "data"
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBufferString(data))
	require.NoError(t, err)

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	err = ctx.GetMetadataBackend().Shutdown()
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err)
	require.Equal(t, data, string(respBody))
}

// dailyDownloadStatsFor sums the daily rollup downloads and bytes recorded for a
// given entity (upload or file), so download-egress tests can assert both the
// event count and the bytes served.
func dailyDownloadStatsFor(t *testing.T, ctx *context.Context, entityType string, entityID string) (downloads int64, bytesServed int64, found bool) {
	t.Helper()
	err := ctx.GetMetadataBackend().ForEachDownloadStatsDaily(func(s *common.DownloadStatsDaily) error {
		if s.EntityType == entityType && s.EntityID == entityID {
			downloads += s.Downloads
			bytesServed += s.Bytes
			found = true
		}
		return nil
	})
	require.NoError(t, err)
	return downloads, bytesServed, found
}

// TestGetFileFullGetRecordsBytesAndEvent pins the semantics for a full GET: the
// bytes served equal the file size and the download event is counted once, on
// both the file and upload rollups.
func TestGetFileFullGetRecordsBytesAndEvent(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "0123456789ABCDEF"
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)
	require.NoError(t, createTestFile(ctx, file, bytes.NewBufferString(data)))

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)
	require.Equal(t, data, rr.Body.String())

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.DownloadCount, "full GET counts one event")

	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found)
	require.Equal(t, int64(1), fileDownloads, "file rollup event count")
	require.Equal(t, file.Size, fileBytes, "file rollup bytes must equal file size")

	uploadDownloads, uploadBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, found)
	require.Equal(t, int64(1), uploadDownloads, "upload rollup event count")
	require.Equal(t, file.Size, uploadBytes, "upload rollup bytes must equal file size")
}

// TestGetFileStartRangeRecordsBytesAndEvent pins Range: bytes=0-49 -> 50 bytes
// served, one event counted (byte-0 range is a download event).
func TestGetFileStartRangeRecordsBytesAndEvent(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := strings.Repeat("x", 100)
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)
	require.NoError(t, createTestFile(ctx, file, bytes.NewBufferString(data)))

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=0-49")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusPartialContent, rr.Code)
	require.Equal(t, 50, rr.Body.Len())

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.DownloadCount, "byte-0 range counts one event")

	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found)
	require.Equal(t, int64(1), fileDownloads)
	require.Equal(t, int64(50), fileBytes, "only the served range bytes are recorded")
}

// TestGetFileMultiRangeRecordsSummedRangeBytesNotFraming pins that a multi-range
// GET records only the summed requested range bytes as egress, not the inflated
// multipart/byteranges total (MIME boundaries + per-part headers) that
// http.ServeContent streams. Range bytes=0-49,60-79 over a 100-byte file selects
// 50+20 = 70 file bytes; the framed body is larger, but only 70 must be recorded
// on the upload row and both rollups.
func TestGetFileMultiRangeRecordsSummedRangeBytesNotFraming(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := strings.Repeat("z", 100)
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)
	require.NoError(t, createTestFile(ctx, file, bytes.NewBufferString(data)))

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=0-49,60-79")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusPartialContent, rr.Code)
	require.Contains(t, rr.Header().Get("Content-Type"), "multipart/byteranges", "multi-range is served as multipart")
	require.Greater(t, rr.Body.Len(), 70, "the framed multipart body is larger than the raw range bytes")

	const summedRangeBytes = int64(70)

	// The first range starts at 0, so the multi-range GET counts one event.
	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.DownloadCount)

	u, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, summedRangeBytes, u.DownloadedBytes, "upload egress excludes multipart framing")

	uploadDownloads, uploadBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, found)
	require.Equal(t, int64(1), uploadDownloads)
	require.Equal(t, summedRangeBytes, uploadBytes, "upload rollup bytes = summed range bytes, not the framed total")

	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found)
	require.Equal(t, int64(1), fileDownloads)
	require.Equal(t, summedRangeBytes, fileBytes, "file rollup bytes = summed range bytes, not the framed total")
}

// TestGetFileMidRangeRecordsBytesNotEvent pins Range: bytes=100-199 (mid-range):
// 100 bytes served but NO download event (downloads unchanged everywhere).
func TestGetFileMidRangeRecordsBytesNotEvent(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := strings.Repeat("y", 200)
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)
	require.NoError(t, createTestFile(ctx, file, bytes.NewBufferString(data)))

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=100-199")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusPartialContent, rr.Code)
	require.Equal(t, 100, rr.Body.Len())

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), f.DownloadCount, "mid-range GET is not a download event")

	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found, "a bytes-only recording still creates a rollup row")
	require.Equal(t, int64(0), fileDownloads, "mid-range must not count an event")
	require.Equal(t, int64(100), fileBytes, "mid-range still records the bytes served")

	uploadDownloads, uploadBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, found)
	require.Equal(t, int64(0), uploadDownloads)
	require.Equal(t, int64(100), uploadBytes)
}

// TestGetFileHeadRecordsNothing pins HEAD: no body served -> no event, no bytes,
// no rollup rows created.
func TestGetFileHeadRecordsNothing(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	data := "data"
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)
	require.NoError(t, createTestFile(ctx, file, bytes.NewBufferString(data)))

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("HEAD", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)
	require.Equal(t, 0, rr.Body.Len(), "HEAD serves no body")

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), f.DownloadCount)

	_, _, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.False(t, found, "HEAD must not record any rollup bytes or events")
}

func TestShouldRecordFileDownload(t *testing.T) {
	req, err := http.NewRequest("GET", "/file", nil)
	require.NoError(t, err)
	require.True(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=0-")
	require.True(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=0-10")
	require.True(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=0-10,20-30")
	require.True(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=00-49")
	require.True(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=000-")
	require.True(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=1-")
	require.False(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=-100")
	require.False(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=0-x")
	require.False(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=0-10,x-y")
	require.False(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=10-0")
	require.False(t, shouldRecordFileDownload(req))

	req.Header.Set("Range", "bytes=0--1")
	require.False(t, shouldRecordFileDownload(req))

	req, err = http.NewRequest("HEAD", "/file", nil)
	require.NoError(t, err)
	require.False(t, shouldRecordFileDownload(req))
}

func TestGetFileNonStartRangeDoesNotCount(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	data := "0123456789"
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBufferString(data))
	require.NoError(t, err)

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=1-")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	require.Equal(t, http.StatusPartialContent, rr.Code)

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), f.DownloadCount)
}

func TestGetFileHeadDoesNotCount(t *testing.T) {
	config := common.NewConfiguration()
	ctx := newTestingContext(config)

	data := "data"
	upload := &common.Upload{IsAdmin: true}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	file.Size = int64(len(data))
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBufferString(data))
	require.NoError(t, err)

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	req, err := http.NewRequest("HEAD", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err)

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), f.DownloadCount)
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
	require.Equal(t, int64(1), f.DownloadCount, "invalid download count")
}

func TestGetOneShotFileRangedRequestCounts(t *testing.T) {
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
	req.Header.Set("Range", "bytes=100-")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.Equal(t, data, string(respBody), "invalid file content")

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err, "unable to get file metadata")
	require.Equal(t, int64(1), f.DownloadCount, "invalid download count")

	// The one-shot/stream branch ignores Range and streams the full file, so the
	// full file content length is recorded as bytes served. file.Size is not set
	// in this fixture, so compare to len(data).
	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found)
	require.Equal(t, int64(1), fileDownloads, "one-shot ranged GET still counts one event")
	require.Equal(t, int64(len(data)), fileBytes, "one-shot branch records the full file bytes served")
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

	err := config.Initialize()
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

	// The webapp-redirect branch (E2EE + X-ClientApp) returns before recording
	// anything; this non-webapp passthrough request is the one that actually
	// streams the file, so it must be the one that counts — exactly once, both
	// the event and the bytes served.
	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.DownloadCount, "the direct E2EE fetch must count exactly one event")
	u, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), u.DownloadCount, "the direct E2EE fetch must count exactly one event on the parent upload")

	fileDownloads, fileBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.True(t, found)
	require.Equal(t, int64(1), fileDownloads, "file rollup event count")
	require.Equal(t, int64(len(data)), fileBytes, "file rollup bytes must equal the encrypted payload size")

	uploadDownloads, uploadBytes, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityUpload, upload.ID)
	require.True(t, found)
	require.Equal(t, int64(1), uploadDownloads, "upload rollup event count")
	require.Equal(t, int64(len(data)), uploadBytes, "upload rollup bytes must equal the encrypted payload size")
}

// TestGetFileRejectedAuthDoesNotRecordDownload pins that a download rejected
// for missing/wrong password credentials never records anything. The check
// happens in middleware.Upload (server/middleware/upload.go), before GetFile
// is ever reached, so this wires the real middleware -> handler chain (unlike
// every other GetFile test in this file, which calls GetFile directly and
// therefore always bypasses the password check).
func TestGetFileRejectedAuthDoesNotRecordDownload(t *testing.T) {
	config := common.NewConfiguration()
	config.FeatureAuthentication = common.FeatureEnabled
	ctx := newTestingContext(config)

	upload := &common.Upload{}
	upload.ProtectedByPassword = true
	upload.Login = "login"
	var err error
	upload.Password, err = common.HashUploadPassword("login", "password")
	require.NoError(t, err, "unable to hash upload credentials")
	upload.InitializeForTests()
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	require.NoError(t, createTestFile(ctx, file, bytes.NewBufferString("data")))

	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, nil)
	require.NoError(t, err, "unable to create new request")
	req = mux.SetURLVars(req, map[string]string{"uploadID": upload.ID})
	// No Authorization header at all -> rejected before GetFile is reached.

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		GetFile(ctx, w, r)
	})

	rr := ctx.NewRecorder(req)
	middleware.Upload(ctx, next).ServeHTTP(rr, req)

	context.TestUnauthorized(t, rr, "please provide valid credentials to access this upload")
	require.False(t, nextCalled, "GetFile must never run for a rejected download")

	f, err := ctx.GetMetadataBackend().GetFile(file.ID)
	require.NoError(t, err)
	require.Zero(t, f.DownloadCount, "a rejected download must not count an event")
	u, err := ctx.GetMetadataBackend().GetUpload(upload.ID)
	require.NoError(t, err)
	require.Zero(t, u.DownloadCount, "a rejected download must not count an event on the parent upload")

	_, _, found := dailyDownloadStatsFor(t, ctx, common.DownloadStatsEntityFile, file.ID)
	require.False(t, found, "a rejected download must not create any rollup row")
}
