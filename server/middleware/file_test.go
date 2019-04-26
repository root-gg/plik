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
package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestFileNoUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}

func TestFileNoFileID(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("upload", common.NewUpload())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing file id")
}

func TestFileNoFileName(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("upload", common.NewUpload())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID": "fileID",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusBadRequest, "Missing file name")
}

func TestFileNotFound(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	ctx.Set("upload", common.NewUpload())

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   "fileID",
		"filename": "filename",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "File fileID not found")
}

func TestFileInvalidFileName(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "filename"

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   file.ID,
		"filename": "invalid_file_name",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusNotFound, "File invalid_file_name not found")
}

func TestFile(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	file := upload.NewFile()
	file.Name = "filename"

	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	// Fake gorilla/mux vars
	vars := map[string]string{
		"fileID":   file.ID,
		"filename": file.Name,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	File(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	f, ok := ctx.Get("file")
	require.True(t, ok, "missing file from context")
	require.Equal(t, file, f, "invalid file from context")
}
