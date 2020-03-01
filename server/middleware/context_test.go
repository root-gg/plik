package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"

	"github.com/root-gg/plik/server/context"
)

func TestContext(t *testing.T) {

	setup := func(ctx *context.Context) {}

	ok := false
	handler := func(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
		require.Equal(t, resp, ctx.GetResp(), "missing response")
		require.Equal(t, req, ctx.GetReq(), "missing request")
		ok = true
	}

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	context.NewChain(Context(setup)).Then(handler).ServeHTTP(rr, req)
	require.True(t, ok, "handler was not called")
}

func TestPanic(t *testing.T) {

	config := &common.Configuration{}
	setup := func(ctx *context.Context) { ctx.SetConfig(config) }

	handler := func(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
		panic("defuk")
	}

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	context.NewChain(Context(setup)).Then(handler).ServeHTTP(rr, req)
	context.TestInternalServerError(t, rr, "internal server error")
}

func TestPanicDebug(t *testing.T) {

	config := &common.Configuration{Debug: true}
	setup := func(ctx *context.Context) { ctx.SetConfig(config) }

	handler := func(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
		panic("defuk")
	}

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	context.NewChain(Context(setup)).Then(handler).ServeHTTP(rr, req)
	context.TestInternalServerError(t, rr, "panic : defuk")
}
