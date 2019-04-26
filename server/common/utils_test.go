/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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

package common

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripPrefixNoPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/prefix", req.URL.Path, "invalid request url")
}

func TestStripPrefixNoExactPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, 301, rr.Code, "invalid handler response status code")
	require.Equal(t, "/prefix/", rr.Result().Header.Get("Location"), "invalid location header")
}

func TestStripPrefix(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/path", req.URL.Path, "invalid location header")
}

func TestStripPrefixNotFound(t *testing.T) {
	req, err := http.NewRequest("GET", "/invalid/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code, "invalid handler response status code")
}

func TestStripPrefixRootSlash(t *testing.T) {
	req, err := http.NewRequest("GET", "/prefix/path", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	rr := httptest.NewRecorder()
	StripPrefix("/prefix/", DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")
	require.Equal(t, "/path", req.URL.Path, "invalid location header")
}