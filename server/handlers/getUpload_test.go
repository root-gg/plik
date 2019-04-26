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

package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestGetUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	upload := common.NewUpload()
	upload.Create()
	upload.Login = "secret"
	upload.Password = "secret"
	createTestUpload(ctx, upload)
	ctx.Set("upload", upload)

	req, err := http.NewRequest("GET", "/upload/"+upload.ID, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUpload(ctx, rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var uploadResult = &common.Upload{}
	err = json.Unmarshal(respBody, uploadResult)
	require.NoError(t, err, "unable to unmarshal response body")

	require.Equal(t, upload.ID, uploadResult.ID, "invalid upload id")
	require.Equal(t, upload.Creation, uploadResult.Creation, "invalid upload creation date")
	require.Equal(t, upload.UploadToken, uploadResult.UploadToken, "invalid upload token")
	require.Equal(t, "", uploadResult.Login, "invalid upload login")
	require.Equal(t, "", uploadResult.Password, "invalid upload password")
}

func TestGetUploadMissingUpload(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/upload/uploadID", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	GetUpload(ctx, rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Internal error")
}
