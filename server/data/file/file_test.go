/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package file

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
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
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	file := upload.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.Error(t, err, "no error with invalid upload id")
}

func TestAddFileImpossibleToCreateDirectory(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.Error(t, err, "unable to create directory")
}

func TestAddFileInvalidReader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	reader := common.NewErrorReader(errors.New("io error"))
	_, err := backend.AddFile(ctx, upload, file, reader)
	require.Error(t, err, "unable to create directory")
	require.Contains(t, err.Error(), "Unable to save file", "invalid error")
}

func TestAddFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	reader := bytes.NewBufferString("data")
	details, err := backend.AddFile(ctx, upload, file, reader)
	require.NoError(t, err, "unable to add file")
	require.NotNil(t, details, "missing backend detail")

	path, ok := details["path"].(string)
	require.True(t, ok, "missing backend detail path")
	require.NotZero(t, path, "missing backend detail path")

	fh, err := os.Open(path)
	require.NoError(t, err, "unable to open file")
	defer fh.Close()

	read, err := ioutil.ReadAll(fh)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")
}

func TestGetFileInvalidDirectory(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	_, err := backend.GetFile(ctx, upload, file.ID)
	require.Error(t, err, "no error with invalid upload directory")
}

func TestGetFileMissingFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	_, err := backend.GetFile(ctx, upload, file.ID)
	require.Error(t, err, "no error with missing file")
	require.Contains(t, err.Error(), "no such file or directory", "invalid error message")
}

func TestGetFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	reader := bytes.NewBufferString("data")
	_, err := backend.AddFile(ctx, upload, file, reader)
	require.NoError(t, err, "unable to add file")

	fileReader, err := backend.GetFile(ctx, upload, file.ID)
	require.NoError(t, err, "unable to get file")

	read, err := ioutil.ReadAll(fileReader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")
}

func TestRemoveFileInvalidDirectory(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	err := backend.RemoveFile(ctx, upload, file.ID)
	require.Error(t, err, "no error with invalid upload id")
}

func TestRemoveFileMissingFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	err := backend.RemoveFile(ctx, upload, file.ID)
	require.Error(t, err, "no error with invalid upload id")
	require.Contains(t, err.Error(), "no such file or directory", "invalid error message")
}

func TestRemoveFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file := upload.NewFile()

	reader := bytes.NewBufferString("data")
	details, err := backend.AddFile(ctx, upload, file, reader)
	require.NoError(t, err, "unable to add file")
	require.NotNil(t, details, "missing backend detail")

	path, ok := details["path"].(string)
	require.True(t, ok, "missing backend detail path")
	require.NotZero(t, path, "missing backend detail path")

	fh, err := os.Open(path)
	require.NoError(t, err, "unable to open file")

	read, err := ioutil.ReadAll(fh)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, "data", string(read), "inavlid file content")

	err = backend.RemoveFile(ctx, upload, file.ID)
	require.NoError(t, err, "unable to remove file")

	_, err = os.Open(path)
	require.Error(t, err, "able to open removed file")
}

func TestRemoveUploadInvalidDirectory(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()

	// null byte looks like a good invalid dirname value ^^
	backend.Config.Directory = string([]byte{0})

	err := backend.RemoveUpload(ctx, upload)
	require.Error(t, err, "no error with invalid upload id")
}

func TestRemoveUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	backend, clean := newBackend(t)
	defer clean()

	upload := common.NewUpload()
	upload.Create()
	file1 := upload.NewFile()
	file2 := upload.NewFile()

	reader1 := bytes.NewBufferString("data")
	details1, err := backend.AddFile(ctx, upload, file1, reader1)
	require.NoError(t, err, "unable to add file")

	path1, ok := details1["path"].(string)
	require.True(t, ok, "missing backend detail path")
	require.NotZero(t, path1, "missing backend detail path")

	fh1, err := os.Open(path1)
	require.NoError(t, err, "unable to open file")
	fh1.Close()

	reader2 := bytes.NewBufferString("data")
	details2, err := backend.AddFile(ctx, upload, file2, reader2)
	require.NoError(t, err, "unable to add file")

	path2, ok := details2["path"].(string)
	require.True(t, ok, "missing backend detail path")
	require.NotZero(t, path2, "missing backend detail path")

	fh2, err := os.Open(path2)
	require.NoError(t, err, "unable to open file")
	fh2.Close()

	err = backend.RemoveUpload(ctx, upload)
	require.NoError(t, err, "unable to remove upload")

	dir, err := backend.getDirectoryFromUploadID(upload.ID)
	require.NoError(t, err, "unable to get upload directory")

	_, err = os.Stat(dir)
	require.Error(t, err, "able to state removed upload directory")

	_, err = os.Open(path1)
	require.Error(t, err, "able to open removed file")

	_, err = os.Open(path2)
	require.Error(t, err, "able to open removed file")
}
