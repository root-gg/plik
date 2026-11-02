package data_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/root-gg/logger"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	data_testing "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
)

// newTestMeta creates a fresh SQLite-backed metadata backend for data migration tests.
func newTestMeta(t *testing.T) *metadata.Backend {
	t.Helper()
	cfg := &metadata.Config{
		Driver:           "sqlite3",
		ConnectionString: "/tmp/plik.data.migrate.test.db",
		EraseFirst:       true,
	}
	b, err := metadata.NewBackend(cfg, logger.NewLogger())
	require.NoError(t, err, "unable to create test metadata backend")
	t.Cleanup(func() { _ = b.Shutdown() })
	return b
}

// seedUploadedFile creates an upload + file record with status uploaded, and puts content in the data backend.
// Note: CreateUpload cascades file creation via GORM associations, so we must not call CreateFile separately.
func seedUploadedFile(t *testing.T, mb *metadata.Backend, src data.Backend, content string) (*common.Upload, *common.File) {
	t.Helper()
	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	file.Size = int64(len(content))
	upload.InitializeForTests()
	err := mb.CreateUpload(upload)
	require.NoError(t, err)
	// File is already created by CreateUpload via GORM cascading (upload.Files)
	err = src.AddFile(file, strings.NewReader(content))
	require.NoError(t, err)
	return upload, file
}

func TestMigrateFiles_Basic(t *testing.T) {
	mb := newTestMeta(t)
	src := data_testing.NewBackend()
	dst := data_testing.NewBackend()

	_, file := seedUploadedFile(t, mb, src, "hello migration")

	stats, err := data.MigrateFiles(src, dst, mb, nil)
	require.NoError(t, err)

	require.Equal(t, int64(1), stats.Copied, "expected 1 file copied")
	require.Equal(t, int64(0), stats.Errors)
	require.Equal(t, int64(len("hello migration")), stats.Bytes)

	// Verify data in destination
	reader, err := dst.GetFile(file)
	require.NoError(t, err)
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	require.Equal(t, "hello migration", string(buf[:n]))
}

func TestMigrateFiles_CopiesRemovedFiles(t *testing.T) {
	mb := newTestMeta(t)
	src := data_testing.NewBackend()
	dst := data_testing.NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileRemoved // still has data in backend
	file.Size = int64(len("removed data"))
	upload.InitializeForTests()
	err := mb.CreateUpload(upload)
	require.NoError(t, err)
	// File is already created by CreateUpload via GORM cascading
	err = src.AddFile(file, strings.NewReader("removed data"))
	require.NoError(t, err)

	stats, err := data.MigrateFiles(src, dst, mb, nil)
	require.NoError(t, err)

	require.Equal(t, int64(1), stats.Copied, "removed file should be copied")
	require.Equal(t, int64(0), stats.Skipped)
}

func TestMigrateFiles_SkipsMissingAndDeleted(t *testing.T) {
	mb := newTestMeta(t)
	src := data_testing.NewBackend()
	dst := data_testing.NewBackend()

	for _, status := range []string{common.FileMissing, common.FileDeleted, common.FileUploading} {
		upload := &common.Upload{}
		f := upload.NewFile()
		f.Status = status
		upload.InitializeForTests()
		err := mb.CreateUpload(upload)
		require.NoError(t, err)
		// No data added to src — these files have no backing storage
	}

	stats, err := data.MigrateFiles(src, dst, mb, nil)
	require.NoError(t, err)

	require.Equal(t, int64(0), stats.Copied)
	require.Equal(t, int64(3), stats.Skipped, "missing/deleted/uploading should be skipped")
}

func TestMigrateFiles_DryRun(t *testing.T) {
	mb := newTestMeta(t)
	src := data_testing.NewBackend()
	dst := data_testing.NewBackend()

	_, _ = seedUploadedFile(t, mb, src, "dry run content")

	opts := &data.MigrateOptions{DryRun: true}
	stats, err := data.MigrateFiles(src, dst, mb, opts)
	require.NoError(t, err)

	require.Equal(t, int64(1), stats.Copied, "dry-run should still count")
	// Destination should be empty — nothing was actually written
	require.Empty(t, dst.GetFiles(), "dry-run must not write to destination")
}

func TestMigrateFiles_IgnoreErrors(t *testing.T) {
	mb := newTestMeta(t)
	src := data_testing.NewBackend()
	dst := data_testing.NewBackend()

	// Seed two files
	_, file1 := seedUploadedFile(t, mb, src, "file one")
	_, file2 := seedUploadedFile(t, mb, src, "file two")

	// Simulate src error for file1 by overwriting its content with a sentinel
	// Use a backend that errors on GetFile for file1 specifically
	// Instead: pre-add file1 to dst so AddFile returns "file exists" error
	err := dst.AddFile(file1, bytes.NewReader([]byte("pre-exists")))
	require.NoError(t, err)

	// Without ignore-errors, should fail
	_, err = data.MigrateFiles(src, dst, mb, &data.MigrateOptions{IgnoreErrors: false, Workers: 1})
	require.Error(t, err)

	// Reset dst (file2 should be written, file1 conflicts)
	dst = data_testing.NewBackend()
	err = dst.AddFile(file1, bytes.NewReader([]byte("pre-exists")))
	require.NoError(t, err)

	// With ignore-errors, should succeed
	stats, err := data.MigrateFiles(src, dst, mb, &data.MigrateOptions{IgnoreErrors: true, Workers: 1})
	require.NoError(t, err)

	require.Equal(t, int64(1), stats.Copied, "file2 should be copied")
	require.Equal(t, int64(1), stats.Errors, "file1 conflict should be counted")

	// Verify file2 was copied
	reader, err := dst.GetFile(file2)
	require.NoError(t, err)
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	require.Equal(t, "file two", string(buf[:n]))
}

func TestMigrateFiles_MultipleWorkers(t *testing.T) {
	mb := newTestMeta(t)
	src := data_testing.NewBackend()
	dst := data_testing.NewBackend()

	for i := range 20 {
		content := strings.Repeat("x", i+1)
		seedUploadedFile(t, mb, src, content)
	}

	stats, err := data.MigrateFiles(src, dst, mb, &data.MigrateOptions{Workers: 8})
	require.NoError(t, err)
	require.Equal(t, int64(20), stats.Copied)
	require.Equal(t, int64(0), stats.Errors)
}
