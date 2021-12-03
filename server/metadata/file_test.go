package metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestBackend_CreateFile(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}

	createUpload(t, b, upload)

	file := upload.NewFile()
	err := b.CreateFile(file)
	require.NoError(t, err, "create file error")
}

func TestBackend_CreateFile_UploadNotFound(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.ID = "nope"

	file := upload.NewFile()
	file.GenerateID()

	err := b.CreateFile(file)
	require.Error(t, err, "no create file error")
}

func TestBackend_GetFile(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()

	createUpload(t, b, upload)

	result, err := b.GetFile(file.ID)
	require.NoError(t, err, "create file error")

	require.NotNil(t, file, "missing file")
	require.Equal(t, file.ID, result.ID, "invalid file id")
}

func TestBackend_GetFile_NotFound(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	file, err := b.GetFile("not found")
	require.NoError(t, err, "get file error")
	require.Nil(t, file, "file not nil")
}

func TestBackend_GetFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	// To spice the test
	upload := &common.Upload{}
	_ = upload.NewFile()
	createUpload(t, b, upload)

	upload = &common.Upload{}
	_ = upload.NewFile()
	_ = upload.NewFile()
	createUpload(t, b, upload)

	files, err := b.GetFiles(upload.ID)
	require.NoError(t, err, "create file error")
	require.Len(t, files, 2, "missing files")
}

func TestBackend_UpdateFile(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()

	createUpload(t, b, upload)

	file.Status = common.FileUploaded
	file.Name = "name"
	file.Md5 = "md5"
	err := b.UpdateFile(file, common.FileMissing)
	require.NoError(t, err, "update file error")

	result, err := b.GetFile(file.ID)
	require.NoError(t, err, "get file error")

	require.NotNil(t, file, "missing file")
	require.Equal(t, file.ID, result.ID, "invalid file id")
	require.Equal(t, file.Name, result.Name, "invalid file name")
	require.Equal(t, file.Md5, result.Md5, "invalid file md5")
	require.Equal(t, file.Status, result.Status, "invalid file md5")

	err = b.UpdateFile(file, common.FileMissing)
	require.Error(t, err, "update file error expected")
}

func TestBackend_UpdateFileStatus(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()
	createUpload(t, b, upload)

	err := b.UpdateFileStatus(file, common.FileMissing, common.FileUploaded)
	require.NoError(t, err, "update file status error")

	f, err := b.GetFile(file.ID)
	require.NoError(t, err, "get file error")
	require.NotNil(t, f, "missing file")
	require.Equal(t, common.FileUploaded, f.Status, "invalid file status")

	err = b.UpdateFileStatus(file, common.FileMissing, common.FileUploaded)
	require.Error(t, err, "update file status error expected")
}

func TestBackend_RemoveFile(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()

	err := b.RemoveFile(file)
	require.Error(t, err, "remove file error expected")

	// File status Uploaded
	file.Status = common.FileUploaded
	createUpload(t, b, upload)

	err = b.RemoveFile(file)
	require.NoError(t, err, "remove file error")

	f, err := b.GetFile(file.ID)
	require.NoError(t, err, "get file error")
	require.NotNil(t, f, "missing file")
	require.Equal(t, common.FileRemoved, f.Status, "invalid file status")

	// File status Missing
	err = b.UpdateFileStatus(file, common.FileRemoved, common.FileMissing)
	require.NoError(t, err, "update file status error")

	err = b.RemoveFile(file)
	require.NoError(t, err, "remove file error")

	f, err = b.GetFile(file.ID)
	require.NoError(t, err, "get file error")
	require.NotNil(t, f, "missing file")
	require.Equal(t, common.FileDeleted, f.Status, "invalid file status")

	// File status Uploading
	err = b.UpdateFileStatus(file, common.FileDeleted, common.FileUploading)
	require.NoError(t, err, "update file status error")

	err = b.RemoveFile(file)
	require.NoError(t, err, "remove file error")

	f, err = b.GetFile(file.ID)
	require.NoError(t, err, "get file error")
	require.NotNil(t, f, "missing file")
	require.Equal(t, common.FileRemoved, f.Status, "invalid file status")
}

func TestBackend_ForEachUploadFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.NewFile()
	upload.NewFile()
	createUpload(t, b, upload)

	var files []*common.File
	f := func(file *common.File) error {
		files = append(files, file)
		return nil
	}

	err := b.ForEachUploadFiles(upload.ID, f)
	require.NoError(t, err, "for each upload file error")
	require.Len(t, files, 2, "file count mismatch")

	f = func(file *common.File) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachUploadFiles(upload.ID, f)
	require.Error(t, err, "for each upload file error expected")
}

func TestBackend_ForEachRemovedFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	upload.NewFile()
	upload.NewFile().Status = common.FileRemoved
	upload.NewFile().Status = common.FileRemoved
	createUpload(t, b, upload)

	var files []*common.File
	f := func(file *common.File) error {
		files = append(files, file)
		return nil
	}

	err := b.ForEachRemovedFile(f)
	require.NoError(t, err, "for each upload file error")
	require.Len(t, files, 2, "file count mismatch")

	f = func(file *common.File) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachRemovedFile(f)
	require.Error(t, err, "for each upload file error expected")
}

func TestBackend_CountUploadFiles(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	_ = upload.NewFile()

	createUpload(t, b, upload)

	count, err := b.CountUploadFiles(upload.ID)
	require.NoError(t, err, "count upload files error")
	require.Equal(t, 1, count, "count upload files mismatch")
}

func TestBackend_ForEachFile(t *testing.T) {
	b := newTestMetadataBackend()
	defer shutdownTestMetadataBackend(b)

	upload := &common.Upload{}
	file := upload.NewFile()
	file.Size = 42
	createUpload(t, b, upload)

	count := 0
	f := func(file *common.File) error {
		count++
		require.Equal(t, int64(42), file.Size, "invalid file size")
		return nil
	}
	err := b.ForEachFile(f)
	require.NoError(t, err, "for each file error : %s", err)
	require.Equal(t, 1, count, "invalid file count")

	f = func(file *common.File) error {
		return fmt.Errorf("expected")
	}
	err = b.ForEachFile(f)
	require.Errorf(t, err, "expected")
}
