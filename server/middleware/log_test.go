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

	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestLogInfo(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	log := context.GetLogger(ctx)
	log.SetMinLevel(logger.INFO)

	buffer := &bytes.Buffer{}
	log.SetOutput(buffer)

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.RequestURI = "path"

	rr := httptest.NewRecorder()
	Log(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Contains(t, string(buffer.Bytes()), "GET path", "invalid log message")
}

func TestLogDebug(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	log := context.GetLogger(ctx)
	log.SetMinLevel(logger.DEBUG)

	buffer := &bytes.Buffer{}
	log.SetOutput(buffer)

	req, err := http.NewRequest("GET", "/version", bytes.NewBuffer([]byte("request body")))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Log(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Contains(t, string(buffer.Bytes()), "GET /version HTTP/1.1", "invalid log message")
	require.Contains(t, string(buffer.Bytes()), "request body", "invalid log message")
}

func TestLogDebugNoBody(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	log := context.GetLogger(ctx)
	log.SetMinLevel(logger.DEBUG)

	buffer := &bytes.Buffer{}
	log.SetOutput(buffer)

	req, err := http.NewRequest("POST", "/file", bytes.NewBuffer([]byte("request body")))
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	Log(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Contains(t, string(buffer.Bytes()), "POST /file HTTP/1.1", "invalid log message")
	require.NotContains(t, string(buffer.Bytes()), "request body", "invalid log message")
}
