package file

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Testing Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

func TestGetFiles(t *testing.T) {
	backend := NewBackend()
	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	files := backend.GetFiles()
	require.NotNil(t, files, "missing file map")
	require.Lenf(t, files, 1, "empty file map")
	require.NotNil(t, files[file.ID], "missing file")
}

func TestAddFileError(t *testing.T) {
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestAddFileReaderError(t *testing.T) {
	backend := NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()
	reader := common.NewErrorReader(errors.New("io error"))

	err := backend.AddFile(file, reader)
	require.Error(t, err, "missing error")
	require.Equal(t, "io error", err.Error(), "invalid error message")
}

func TestAddFile(t *testing.T) {
	backend := NewBackend()
	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")
}

func TestGetFileError(t *testing.T) {
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := &common.Upload{}
	file := upload.NewFile()

	_, err := backend.GetFile(file)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestGetFile(t *testing.T) {
	backend := NewBackend()
	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(file)
	require.NoError(t, err, "unable to get file")
}

func TestRemoveFileError(t *testing.T) {
	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.RemoveFile(file)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestRemoveFile(t *testing.T) {
	backend := NewBackend()

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(file)
	require.NoError(t, err, "unable to get file")

	err = backend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	_, err = backend.GetFile(file)
	require.Error(t, err, "unable to get file")
	require.Equal(t, "file not found", err.Error(), "invalid error message")
}
