package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func newBackend(t *testing.T) (backend *Backend, cleanup func()) {
	dir, err := ioutil.TempDir("", "pliktest")
	require.NoError(t, err, "unable to create temp directory")

	backend = NewBackend(&Config{Directory: dir})
	cleanup = func() {
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Println(err)
		}
	}

	return backend, cleanup
}

func TestNewFileBackendConfig(t *testing.T) {
	config := NewConfig(make(map[string]interface{}))
	require.NotNil(t, config, "invalid nil config")
}

func TestAddFileInvalidUploadId(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.Error(t, err, "no error with invalid upload id")
}

func TestAddFileImpossibleToCreateDirectory(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	err := backend.AddFile(file, &bytes.Buffer{})
	require.Error(t, err, "unable to create directory")
}

func TestAddFileInvalidReader(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := common.NewErrorReader(errors.New("io error"))
	err := backend.AddFile(file, reader)
	require.Error(t, err, "unable to create directory")
	require.Contains(t, err.Error(), "io error", "invalid error")
}

func TestAddFile(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := bytes.NewBufferString("data")
	err := backend.AddFile(file, reader)
	require.NoError(t, err, "unable to add file")

	_, path, err := backend.getPathCompat(file)
	require.NoError(t, err, "unable to get file path")

	fh, err := os.Open(path)
	require.NoError(t, err, "unable to open file")
	defer fh.Close()

	read, err := ioutil.ReadAll(fh)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")
}

func TestGetFileInvalidDirectory(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	_, err := backend.GetFile(file)
	require.Error(t, err, "no error with invalid upload directory")
}

func TestGetFileMissingFile(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	_, err := backend.GetFile(file)
	require.Error(t, err, "no error with missing file")
	require.Contains(t, err.Error(), "no such file or directory", "invalid error message")
}

func TestGetFile(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := bytes.NewBufferString("data")
	err := backend.AddFile(file, reader)
	require.NoError(t, err, "unable to add file")

	fileReader, err := backend.GetFile(file)
	require.NoError(t, err, "unable to get file")

	read, err := ioutil.ReadAll(fileReader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")
}

func TestGetFileCompathPath(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := bytes.NewBufferString("data")
	dir := fmt.Sprintf("%s/%s/%s", backend.Config.Directory, file.UploadID[:2], file.UploadID)
	path := fmt.Sprintf("%s/%s", dir, file.ID)

	// Create directory
	err := os.MkdirAll(dir, 0777)
	require.NoError(t, err, "error creating directories")

	// Create file
	out, err := os.Create(path)
	require.NoError(t, err, "error touching file")

	// Write file content
	_, err = io.Copy(out, reader)
	require.NoError(t, err, "error writing file")

	fileReader, err := backend.GetFile(file)
	require.NoError(t, err, "unable to get file")

	read, err := ioutil.ReadAll(fileReader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")
}

func TestRemoveFileInvalidDirectory(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	err := backend.RemoveFile(file)
	require.Error(t, err, "no error with invalid upload id")
}

func TestRemoveFileMissingFile(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	err := backend.RemoveFile(file)
	require.NoError(t, err, "error removing missing file")
}

func TestRemoveFile(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := bytes.NewBufferString("data")
	err := backend.AddFile(file, reader)
	require.NoError(t, err, "unable to add file")

	_, path, err := backend.getPathCompat(file)
	require.NoError(t, err, "unable to get file path")

	fh, err := os.Open(path)
	require.NoError(t, err, "unable to open file")

	read, err := ioutil.ReadAll(fh)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")

	err = backend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	_, err = os.Open(path)
	require.Error(t, err, "able to open removed file")
}

func TestRemoveFileTwice(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := bytes.NewBufferString("data")
	err := backend.AddFile(file, reader)
	require.NoError(t, err, "unable to add file")

	_, path, err := backend.getPathCompat(file)
	require.NoError(t, err, "unable to get file path")

	fh, err := os.Open(path)
	require.NoError(t, err, "unable to open file")

	read, err := ioutil.ReadAll(fh)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")

	err = backend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	_, err = os.Open(path)
	require.Error(t, err, "able to open removed file")

	err = backend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")
}

func TestRemoveFileCompatPath(t *testing.T) {
	backend, clean := newBackend(t)
	defer clean()

	upload := &common.Upload{}
	file := upload.NewFile()
	upload.InitializeForTests()

	reader := bytes.NewBufferString("data")
	dir := fmt.Sprintf("%s/%s/%s", backend.Config.Directory, file.UploadID[:2], file.UploadID)
	path := fmt.Sprintf("%s/%s", dir, file.ID)

	// Create directory
	err := os.MkdirAll(dir, 0777)
	require.NoError(t, err, "error creating directories")

	// Create file
	out, err := os.Create(path)
	require.NoError(t, err, "error touching file")

	// Write file content
	_, err = io.Copy(out, reader)
	require.NoError(t, err, "error writing file")

	fh, err := os.Open(path)
	require.NoError(t, err, "unable to open file")

	read, err := ioutil.ReadAll(fh)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")

	err = backend.RemoveFile(file)
	require.NoError(t, err, "unable to remove file")

	_, err = os.Open(path)
	require.Error(t, err, "able to open removed file")
}
