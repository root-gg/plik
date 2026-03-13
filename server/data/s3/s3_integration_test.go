package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

// newTestBackend creates a real S3 backend from PLIKD_CONFIG.
// Skips the test if the config is absent or not an S3 backend.
func newTestBackend(t *testing.T) *Backend {
	t.Helper()

	configPath := os.Getenv("PLIKD_CONFIG")
	if configPath == "" {
		t.Skip("PLIKD_CONFIG not set, skipping S3 integration test")
	}

	config, err := common.LoadConfiguration(configPath)
	require.NoError(t, err, "unable to load config")

	if config.DataBackend != "s3" {
		t.Skip("data backend is not s3, skipping S3 integration test")
	}

	backend, err := NewBackend(NewConfig(config.DataBackendConfig))
	require.NoError(t, err, "unable to create S3 backend")

	return backend
}

// statObject is a test helper that checks whether an object exists in S3.
func statObject(t *testing.T, b *Backend, objectName string) bool {
	t.Helper()
	_, err := b.client.StatObject(context.TODO(), b.config.Bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false
		}
		t.Fatalf("unexpected StatObject error: %s", err)
	}
	return true
}

func TestRemoveFileNewFormat(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	content := "test data for removal"
	err := b.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to add file")

	// Verify the object exists at the storage level
	objectName := b.getObjectName(file.UploadID, file.ID)
	require.True(t, statObject(t, b, objectName), "object should exist after AddFile")

	// Verify we can read it back
	reader, err := b.GetFile(file)
	require.NoError(t, err, "unable to get file")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, content, string(data), "file content mismatch")
	_ = reader.Close()

	// Remove the file
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	// Verify the object is gone at the storage level
	require.False(t, statObject(t, b, objectName), "object should not exist after RemoveFile")
}

func TestRemoveFileLegacyFormat(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	content := "legacy test data"

	// Manually upload with legacy naming ({fileID} only)
	legacyName := b.getObjectNameLegacy(file.ID)
	_, err := b.client.PutObject(
		context.TODO(),
		b.config.Bucket,
		legacyName,
		bytes.NewBufferString(content),
		int64(len(content)),
		b.newPutObjectOptions("application/octet-stream"),
	)
	require.NoError(t, err, "unable to put legacy object")

	// Verify the legacy object exists
	require.True(t, statObject(t, b, legacyName), "legacy object should exist")

	// Verify the new-format key does NOT exist
	newName := b.getObjectName(file.UploadID, file.ID)
	require.False(t, statObject(t, b, newName), "new-format object should not exist")

	// RemoveFile should fall back to the legacy name and delete it
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove file with legacy name")

	// Verify the legacy object is gone
	require.False(t, statObject(t, b, legacyName), "legacy object should not exist after RemoveFile")
}

func TestRemoveFileTwice(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	err := b.AddFile(file, bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to add file")

	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	// Removing again should not error (interface contract: "should not fail if the file is not found")
	err = b.RemoveFile(file)
	require.NoError(t, err, "error removing already-removed file")
}

func TestRemoveFileNotFound(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	// File was never added — RemoveFile should not error
	err := b.RemoveFile(file)
	require.NoError(t, err, "error removing non-existent file")
}

func TestRemoveFileWithPrefix(t *testing.T) {
	b := newTestBackend(t)

	if b.config.Prefix == "" {
		t.Skip("S3 prefix not configured, skipping prefix test")
	}

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	content := "prefixed data"
	err := b.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to add file")

	// Verify the object exists with the prefix
	objectName := b.getObjectName(file.UploadID, file.ID)
	require.Contains(t, objectName, b.config.Prefix+"/", "object name should contain prefix")
	require.True(t, statObject(t, b, objectName), "prefixed object should exist")

	// Without the prefix, object should not exist
	bareObjectName := fmt.Sprintf("%s.%s", file.UploadID, file.ID)
	require.False(t, statObject(t, b, bareObjectName), "bare object should not exist")

	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove prefixed file")

	require.False(t, statObject(t, b, objectName), "prefixed object should not exist after RemoveFile")
}

// TestAddFileEmpty verifies that zero-byte files are handled correctly.
// This exercises the edge case where io.CopyN returns (0, io.EOF).
func TestAddFileEmpty(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	err := b.AddFile(file, bytes.NewReader(nil))
	require.NoError(t, err, "unable to add empty file")

	// Verify we can read it back (should be zero bytes)
	reader, err := b.GetFile(file)
	require.NoError(t, err, "unable to get empty file")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read empty file")
	require.Empty(t, data, "empty file should have no content")
	_ = reader.Close()

	// Clean up
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove empty file")
}

// TestAddFileSmall verifies that files smaller than PartSize are uploaded
// via a single PUT request (buffer-then-decide: EOF before buffer fills).
func TestAddFileSmall(t *testing.T) {
	b := newTestBackend(t)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	// Small content — well under the default 16MiB PartSize
	content := "hello, small file"
	err := b.AddFile(file, bytes.NewBufferString(content))
	require.NoError(t, err, "unable to add small file")

	// Verify we can read it back correctly
	reader, err := b.GetFile(file)
	require.NoError(t, err, "unable to get small file")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read small file")
	require.Equal(t, content, string(data), "small file content mismatch")
	_ = reader.Close()

	// Verify the object exists at the expected key
	objectName := b.getObjectName(file.UploadID, file.ID)
	require.True(t, statObject(t, b, objectName), "small file object should exist")

	// Clean up
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove small file")
}

// TestAddFileLarge verifies that files larger than PartSize are uploaded
// via multipart upload (buffer-then-decide: buffer fills, falls through to multipart).
func TestAddFileLarge(t *testing.T) {
	b := newTestBackend(t)

	// Use a small PartSize so we don't need to allocate 16MiB+ in tests
	b.config.PartSize = 5 * 1024 * 1024 // 5MiB (S3 minimum)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	// Create content larger than PartSize to trigger multipart
	contentSize := int(b.config.PartSize) + 1024 // PartSize + 1KiB
	content := make([]byte, contentSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	err := b.AddFile(file, bytes.NewReader(content))
	require.NoError(t, err, "unable to add large file")

	// Verify we can read it back correctly
	reader, err := b.GetFile(file)
	require.NoError(t, err, "unable to get large file")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read large file")
	require.Equal(t, content, data, "large file content mismatch")
	_ = reader.Close()

	// Clean up
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove large file")
}

// TestAddFileLargeParallel verifies that parallel multipart uploads work
// when PartUploadConcurrency > 1.
func TestAddFileLargeParallel(t *testing.T) {
	b := newTestBackend(t)

	// Use small PartSize and enable parallel uploads
	b.config.PartSize = 5 * 1024 * 1024 // 5MiB
	b.config.PartUploadConcurrency = 4

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Status = common.FileUploaded
	upload.InitializeForTests()

	// Create content spanning multiple parts (3 × PartSize to exercise parallelism)
	contentSize := int(b.config.PartSize) * 3
	content := make([]byte, contentSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	err := b.AddFile(file, bytes.NewReader(content))
	require.NoError(t, err, "unable to add large file with parallel upload")

	// Verify we can read it back correctly
	reader, err := b.GetFile(file)
	require.NoError(t, err, "unable to get large file")
	data, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read large file")
	require.Equal(t, content, data, "large file content mismatch with parallel upload")
	_ = reader.Close()

	// Clean up
	err = b.RemoveFile(file)
	require.NoError(t, err, "unable to remove large file")
}
