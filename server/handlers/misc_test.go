package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
)

func newTestingContext(config *common.Configuration) (ctx *context.Context) {
	ctx = &context.Context{}
	config.Debug = true
	ctx.SetConfig(config)
	ctx.SetLogger(config.NewLogger())
	ctx.SetWhitelisted(true)
	ctx.SetDataBackend(data_test.NewBackend())
	ctx.SetStreamBackend(data_test.NewBackend())
	ctx.SetAuthenticator(&common.SessionAuthenticator{SignatureKey: "sigkey"})

	metadataBackendConfig := &metadata.Config{Driver: "sqlite3", ConnectionString: "/tmp/plik.test.db", EraseFirst: true}
	metadataBackend, err := metadata.NewBackend(metadataBackendConfig, config.NewLogger())
	if err != nil {
		panic(fmt.Errorf("unable to initialize metadata backend : %s", err))
	}
	ctx.SetMetadataBackend(metadataBackend)

	return ctx
}

func TestGetVersion(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/version", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetVersion(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result *common.BuildInfo
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err, "unable to unmarshal response body")

	require.EqualValues(t, common.GetBuildInfo(), result, "invalid build info")
}

func TestGetVersionEnhancedWebSecurity(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	ctx.GetConfig().EnhancedWebSecurity = true

	req, err := http.NewRequest("GET", "/version", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetVersion(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result *common.BuildInfo
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err, "unable to unmarshal response body")

	require.NotEmpty(t, result.Version, "invalid build info")
	require.Empty(t, result.GoVersion, result, "invalid build info")
	require.Empty(t, result.GitFullRevision, result, "invalid build info")
	require.Empty(t, result.GitShortRevision, result, "invalid build info")
	require.Zero(t, result.Date, result, "invalid build info")
	require.False(t, result.IsMint, result, "invalid build info")
	require.False(t, result.IsRelease, result, "invalid build info")
	require.Empty(t, result.Host, result, "invalid build info")
	require.Empty(t, result.User, result, "invalid build info")
}

func TestGetConfiguration(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/version", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetConfiguration(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result *common.Configuration
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err, "unable to unmarshal response body")
}

func TestGetQrCode(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/qrcode?url="+url.QueryEscape("https://root.gg"), bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetQrCode(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")
	require.Equal(t, "image/png", rr.Header().Get("Content-Type"), "invalid response content type")
}

func TestGetQrCodeWithSize(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/qrcode?url="+url.QueryEscape("https://root.gg")+"&size=100", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetQrCode(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")
	require.NotEqual(t, 0, len(respBody), "invalid empty response body")
	require.Equal(t, "image/png", rr.Header().Get("Content-Type"), "invalid response content type")
}

func TestGetQrCodeWithInvalidSize(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/qrcode?url="+url.QueryEscape("https://root.gg")+"&size=10000", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetQrCode(ctx, rr, req)

	context.TestBadRequest(t, rr, "QRCode size must be lower than 1000")
}

func TestGetQrCodeWithInvalidSize2(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/qrcode?url="+url.QueryEscape("https://root.gg")+"&size=-1", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetQrCode(ctx, rr, req)

	context.TestBadRequest(t, rr, "QRCode size must be positive")
}

func TestLogout(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/logout", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	Logout(ctx, rr, req)
	context.TestOK(t, rr)
}

func TestGetRedirectionURL(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/auth", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetReq(req)

	// Without referer
	redirectURL, err := getRedirectURL(ctx, "/callback")
	require.Error(t, err, "missing no referrer error")

	// With invalid referer
	req.Header.Set("referer", ":::foo:::")
	redirectURL, err = getRedirectURL(ctx, "/callback")
	require.Error(t, err, "missing invalid referrer error")

	// With trailing slash
	req.Header.Set("referer", "https://plik.root.gg/")
	redirectURL, err = getRedirectURL(ctx, "/callback")
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg/callback", redirectURL)

	// Without trailing slash
	req.Header.Set("referer", "https://plik.root.gg")
	redirectURL, err = getRedirectURL(ctx, "/callback")
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg/callback", redirectURL)
}

func TestGetRedirectionURLWithPath(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	ctx.GetConfig().Path = "/path"

	req, err := http.NewRequest("GET", "/logout", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx.SetReq(req)

	// With trailing slash
	req.Header.Set("referer", "https://plik.root.gg/")
	redirectURL, err := getRedirectURL(ctx, "/callback")
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg/path/callback", redirectURL)

	// Without trailing slash
	req.Header.Set("referer", "https://plik.root.gg")
	redirectURL, err = getRedirectURL(ctx, "/callback")
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg/path/callback", redirectURL)
}
