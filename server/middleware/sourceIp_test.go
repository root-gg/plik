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

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/stretchr/testify/require"
)

func TestSourceIPInvalid(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.RemoteAddr = "invalid_ip_address"
	rr := httptest.NewRecorder()
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to parse source IP address")
}

func TestSourceIPInvalidFromHeader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).SourceIPHeader = "IP"

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("IP", "invalid_ip_address")

	rr := httptest.NewRecorder()
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestFail(t, rr, http.StatusInternalServerError, "Unable to parse source IP address")
}

func TestSourceIP(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")
	req.RemoteAddr = "1.1.1.1:1111"

	rr := httptest.NewRecorder()
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	ip := context.GetSourceIP(ctx)
	require.Equal(t, "1.1.1.1", ip.String(), "invalid source ip from context")
}

func TestSourceIPFromHeader(t *testing.T) {
	ctx := context.NewTestingContext(common.NewConfiguration())
	context.GetConfig(ctx).SourceIPHeader = "IP"

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("IP", "1.1.1.1")

	rr := httptest.NewRecorder()
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	ip := context.GetSourceIP(ctx)
	require.Equal(t, "1.1.1.1", ip.String(), "invalid source ip from context")
}
