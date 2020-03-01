package middleware

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

func TestSourceIPInvalid(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")

	req.RemoteAddr = "invalid_ip_address"
	rr := ctx.NewRecorder(req)

	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)
	context.TestInternalServerError(t, rr, "unable to parse source IP address")
}

func TestSourceIPInvalidFromHeader(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().SourceIPHeader = "IP"

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("IP", "invalid_ip_address")

	rr := ctx.NewRecorder(req)
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	context.TestBadRequest(t, rr, "invalid IP address")
}

func TestSourceIP(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")
	req.RemoteAddr = "1.1.1.1:1111"

	rr := ctx.NewRecorder(req)
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	ip := ctx.GetSourceIP()
	require.Equal(t, "1.1.1.1", ip.String(), "invalid source ip from context")
}

func TestSourceIPFromHeader(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().SourceIPHeader = "IP"

	req, err := http.NewRequest("GET", "url", &bytes.Buffer{})
	require.NoError(t, err, "unable to create new request")
	req.Header.Set("IP", "1.1.1.1")

	rr := ctx.NewRecorder(req)
	SourceIP(ctx, common.DummyHandler).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "invalid handler response status code")

	ip := ctx.GetSourceIP()
	require.Equal(t, "1.1.1.1", ip.String(), "invalid source ip from context")
}
