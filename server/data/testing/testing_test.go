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
	"testing"

	"github.com/root-gg/juliet"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/stretchr/testify/require"
)

// Ensure Testing Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

func newTestingContext(config *common.Configuration) (ctx *juliet.Context) {
	ctx = juliet.NewContext()
	ctx.Set("config", config)
	ctx.Set("logger", logger.NewLogger())
	return ctx
}

func TestAddFileError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()
	file := upload.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestAddFileReaderError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()

	upload := common.NewUpload()
	file := upload.NewFile()
	reader := common.NewErrorReader(errors.New("io error"))

	_, err := backend.AddFile(ctx, upload, file, reader)
	require.Error(t, err, "missing error")
	require.Equal(t, "io error", err.Error(), "invalid error message")
}

func TestAddFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	upload := common.NewUpload()
	file := upload.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")
}

func TestGetFileError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()
	file := upload.NewFile()

	_, err := backend.GetFile(ctx, upload, file.ID)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestGetFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	backend := NewBackend()
	upload := common.NewUpload()
	file := upload.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(ctx, upload, file.ID)
	require.NoError(t, err, "unable to get file")
}

func TestRemoveFileError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()
	file := upload.NewFile()

	err := backend.RemoveFile(ctx, upload, file.ID)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestRemoveFile(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()

	upload := common.NewUpload()
	file := upload.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(ctx, upload, file.ID)
	require.NoError(t, err, "unable to get file")

	err = backend.RemoveFile(ctx, upload, file.ID)
	require.NoError(t, err, "unable to remove file")

	_, err = backend.GetFile(ctx, upload, file.ID)
	require.Error(t, err, "unable to get file")
	require.Equal(t, "File not found", err.Error(), "invalid error message")
}

func TestRemoveUploadError(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()
	backend.SetError(errors.New("error"))

	upload := common.NewUpload()

	err := backend.RemoveUpload(ctx, upload)
	require.Error(t, err, "missing error")
	require.Equal(t, "error", err.Error(), "invalid error message")
}

func TestRemoveUpload(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	backend := NewBackend()

	upload := common.NewUpload()
	file := upload.NewFile()

	upload2 := common.NewUpload()
	file2 := upload2.NewFile()

	_, err := backend.AddFile(ctx, upload, file, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.AddFile(ctx, upload2, file2, &bytes.Buffer{})
	require.NoError(t, err, "unable to add file")

	_, err = backend.GetFile(ctx, upload, file.ID)
	require.NoError(t, err, "unable to get file")

	_, err = backend.GetFile(ctx, upload, file2.ID)
	require.NoError(t, err, "unable to get file")

	err = backend.RemoveUpload(ctx, upload)
	require.NoError(t, err, "unable to remove file")

	_, err = backend.GetFile(ctx, upload, file.ID)
	require.Error(t, err, "unable to get file")
	require.Equal(t, "File not found", err.Error(), "invalid error message")

	_, err = backend.GetFile(ctx, upload2, file2.ID)
	require.NoError(t, err, "unable to get file")
}
