package context

import (
	"fmt"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/metadata"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewRecorder create a new response recorder for testing
func (ctx *Context) NewRecorder(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	ctx.SetReq(req)
	ctx.SetResp(rr)
	return rr
}

// TestMissingParameter is a helper to test a httptest.ResponseRecorder status
func TestMissingParameter(t *testing.T, resp *httptest.ResponseRecorder, parameter string) {
	TestFail(t, resp, http.StatusBadRequest, fmt.Sprintf("missing %s", parameter))
}

// TestInvalidParameter is a helper to test a httptest.ResponseRecorder status
func TestInvalidParameter(t *testing.T, resp *httptest.ResponseRecorder, parameter string) {
	TestFail(t, resp, http.StatusBadRequest, fmt.Sprintf("invalid %s", parameter))
}

// TestNotFound is a helper to test a httptest.ResponseRecorder status
func TestNotFound(t *testing.T, resp *httptest.ResponseRecorder, message string) {
	TestFail(t, resp, http.StatusNotFound, message)
}

// TestForbidden is a helper to test a httptest.ResponseRecorder status
func TestForbidden(t *testing.T, resp *httptest.ResponseRecorder, message string) {
	TestFail(t, resp, http.StatusForbidden, message)
}

// TestUnauthorized is a helper to test a httptest.ResponseRecorder status
func TestUnauthorized(t *testing.T, resp *httptest.ResponseRecorder, message string) {
	TestFail(t, resp, http.StatusUnauthorized, message)
}

// TestBadRequest is a helper to test a httptest.ResponseRecorder status
func TestBadRequest(t *testing.T, resp *httptest.ResponseRecorder, message string) {
	TestFail(t, resp, http.StatusBadRequest, message)
}

// TestInternalServerError is a helper to test a httptest.ResponseRecorder status
func TestInternalServerError(t *testing.T, resp *httptest.ResponseRecorder, message string) {
	TestFail(t, resp, http.StatusInternalServerError, message)
}

// TestFail is a helper to test a httptest.ResponseRecorder status
func TestFail(t *testing.T, resp *httptest.ResponseRecorder, status int, message string) {
	require.Equal(t, status, resp.Code, "handler returned wrong status code")

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, err, 0, len(respBody), "empty response body")

	if message != "" {
		require.Contains(t, string(respBody), message, "invalid response error message")
	}
}

// TestOK is a helper to test a httptest.ResponseRecorder status
func TestOK(t *testing.T, resp *httptest.ResponseRecorder) {
	if resp.Code != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		require.Equal(t, http.StatusOK, resp.Code, fmt.Sprintf("handler error %s", string(respBody)))
	}
}

// TestPanic is a helper to test a httptest.ResponseRecorder status
func TestPanic(t *testing.T, resp *httptest.ResponseRecorder, message string, handler func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("the code did not panic")
		}
	}()
	handler()
}

// Setup metadata backend
var metadataBackendConfig = &metadata.Config{Driver: "sqlite3", ConnectionString: "/tmp/plik.test.db", EraseFirst: true, Debug: false}

func newTestMetadataBackend() *metadata.Backend {
	b, err := metadata.NewBackend(metadataBackendConfig, logger.NewLogger())
	if err != nil {
		panic(fmt.Sprintf("unable to create metadata backend : %s", err))
	}

	return b
}

func shutdownTestMetadataBackend(b *metadata.Backend) {
	err := b.Shutdown()
	if err != nil {
		fmt.Printf("Unable to shutdown metadata backend : %s\n", err)
	}
}

func setupNewMetadataBackend(ctx *Context) (cancel func()) {
	meta := newTestMetadataBackend()
	ctx.SetMetadataBackend(meta)
	return func() { shutdownTestMetadataBackend(meta) }
}
