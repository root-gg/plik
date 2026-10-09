package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	data_test "github.com/root-gg/plik/server/data/testing"
	"github.com/root-gg/plik/server/metadata"
	"github.com/root-gg/plik/server/middleware"
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

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result *common.BuildInfo
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err, "unable to unmarshal response body")

	// Non-admin: BuildInfo is sanitized — version is kept, build metadata is stripped
	require.NotEmpty(t, result.Version, "build info version should be present")
	require.Empty(t, result.GoVersion, "build info GoVersion should be sanitized")
	require.Empty(t, result.GitFullRevision, "build info GitFullRevision should be sanitized")
	require.Empty(t, result.GitShortRevision, "build info GitShortRevision should be sanitized")
	require.Zero(t, result.Date, "build info Date should be sanitized")
	require.False(t, result.IsMint, "build info IsMint should be sanitized")
	require.False(t, result.IsRelease, "build info IsRelease should be sanitized")
	require.Empty(t, result.Host, "build info Host should be sanitized")
	require.Empty(t, result.User, "build info User should be sanitized")
}

func TestGetVersionAdmin(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	user := common.NewUser(common.ProviderLocal, "admin")
	user.IsAdmin = true
	ctx.SetUser(user)

	req, err := http.NewRequest("GET", "/version", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetVersion(ctx, rr, req)

	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
	require.NoError(t, err, "unable to read response body")

	var result *common.BuildInfo
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err, "unable to unmarshal response body")

	// Admin: BuildInfo is NOT sanitized — response matches the raw build info
	require.NotEmpty(t, result.Version, "build info version should be present")

	// Compare against unsanitized build info to confirm nothing was stripped
	expected := common.GetBuildInfo()
	require.Equal(t, expected.GoVersion, result.GoVersion, "admin should see GoVersion")
	require.Equal(t, expected.IsRelease, result.IsRelease, "admin should see IsRelease")
	require.Equal(t, expected.IsMint, result.IsMint, "admin should see IsMint")
	require.Equal(t, expected.Date, result.Date, "admin should see Date")
	require.Equal(t, expected.Host, result.Host, "admin should see Host")
	require.Equal(t, expected.User, result.User, "admin should see User")
	require.Equal(t, expected.GitShortRevision, result.GitShortRevision, "admin should see GitShortRevision")
	require.Equal(t, expected.GitFullRevision, result.GitFullRevision, "admin should see GitFullRevision")
}

func TestGetConfiguration(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())

	req, err := http.NewRequest("GET", "/version", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	rr := ctx.NewRecorder(req)
	GetConfiguration(ctx, rr, req)

	// Check the status code is what we expect.
	context.TestOK(t, rr)

	respBody, err := io.ReadAll(rr.Body)
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

	respBody, err := io.ReadAll(rr.Body)
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

	respBody, err := io.ReadAll(rr.Body)
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

func TestLogoutNoAuthenticator(t *testing.T) {
	ctx := newTestingContext(common.NewConfiguration())
	// Clear the authenticator to simulate auth being disabled
	ctx.SetAuthenticator(nil)

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

func TestCheckDownloadDomain(t *testing.T) {
	config := common.NewConfiguration()
	config.DownloadDomain = "https://plik.root.gg"
	config.DownloadDomainAlias = []string{"https://dl.root.gg"}
	require.NoError(t, config.Initialize())

	ctx := newTestingContext(config)

	req, err := http.NewRequest("GET", "/files/my.file", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	req.Host = "plik.root.gg"
	rr := ctx.NewRecorder(req)
	checkDownloadDomain(ctx)
	context.TestOK(t, rr)

	req.Host = "dl.root.gg"
	rr = ctx.NewRecorder(req)
	checkDownloadDomain(ctx)
	context.TestOK(t, rr)

	req.Host = "invalid.domain"
	rr = ctx.NewRecorder(req)
	checkDownloadDomain(ctx)
	context.TestBadRequest(t, rr, "Invalid download domain invalid.domain")
}

func TestGetRedirectionURLWithPlikDomain(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	require.NoError(t, config.Initialize())

	ctx := newTestingContext(config)

	req, err := http.NewRequest("GET", "/auth", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	ctx.SetReq(req)

	// Should use PlikDomain, even without Referer header
	redirectURL, err := getRedirectURL(ctx, "/callback")
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg/callback", redirectURL)
}

func TestGetRedirectionURLWithPlikDomainAndPath(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.Path = "/sub"
	require.NoError(t, config.Initialize())

	ctx := newTestingContext(config)

	req, err := http.NewRequest("GET", "/auth", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	ctx.SetReq(req)

	redirectURL, err := getRedirectURL(ctx, "/callback")
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg/sub/callback", redirectURL)
}

func TestCORSHeaders(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	ctx := newTestingContext(config)

	data := "data"
	upload := &common.Upload{}
	file := upload.NewFile()
	file.Name = "file"
	file.Status = common.FileUploaded
	createTestUpload(t, ctx, upload)

	err := createTestFile(ctx, file, bytes.NewBuffer([]byte(data)))
	require.NoError(t, err, "unable to create test file")

	ctx.SetUpload(upload)
	ctx.SetFile(file)

	// Request with Origin header → should get CORS headers
	req, err := http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.Host = "dl.plik.root.gg"
	req.Header.Set("Origin", "https://plik.root.gg")

	rr := ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Equal(t, "https://plik.root.gg", rr.Header().Get("Access-Control-Allow-Origin"))

	// Request without Origin header → should NOT get CORS headers
	req, err = http.NewRequest("GET", "/file/"+upload.ID+"/"+file.ID+"/"+file.Name, bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.Host = "dl.plik.root.gg"

	rr = ctx.NewRecorder(req)
	GetFile(ctx, rr, req)
	context.TestOK(t, rr)

	require.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSPreflight(t *testing.T) {
	config := common.NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	require.NoError(t, config.Initialize())

	ctx := newTestingContext(config)

	req, err := http.NewRequest("OPTIONS", "/file/uploadID/fileID/filename", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	req.Host = "dl.plik.root.gg"
	req.Header.Set("Origin", "https://plik.root.gg")

	// The middleware should short-circuit without needing upload/file in context
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	rr := ctx.NewRecorder(req)
	middleware.CORSPreflight(ctx, next).ServeHTTP(rr, req)

	require.False(t, nextCalled, "middleware should short-circuit OPTIONS requests")
	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Equal(t, "https://plik.root.gg", rr.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestHealth(t *testing.T) {
	config := common.NewConfiguration()
	require.NoError(t, config.Initialize())

	req, err := http.NewRequest("GET", "/health", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")

	ctx := newTestingContext(config)
	rr := ctx.NewRecorder(req)
	Health(ctx, rr, req)
	context.TestOK(t, rr)
}

func TestIsUploadSort(t *testing.T) {
	require.True(t, isUploadSort(""))
	require.True(t, isUploadSort("date"))
	require.True(t, isUploadSort("size"))
	require.True(t, isUploadSort("downloads"))
	require.True(t, isUploadSort("downloadedBytes"))
	require.False(t, isUploadSort("lifetimeSize"))
	require.False(t, isUploadSort("wat"))
}

func TestParseTrendingWindow(t *testing.T) {
	for _, path := range []string{
		"/stats/trending/uploads",
		"/stats/trending/uploads?window=all",
		"/stats/trending/uploads?window=1d",
		"/stats/trending/uploads?window=7d",
		"/stats/trending/uploads?window=30d",
	} {
		req, err := http.NewRequest("GET", path, bytes.NewBuffer([]byte{}))
		require.NoError(t, err, "unable to create new request")
		_, err = parseTrendingWindow(req)
		require.NoError(t, err)
	}

	req, err := http.NewRequest("GET", "/stats/trending/uploads?window=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	_, err = parseTrendingWindow(req)
	require.Error(t, err)
}

func TestParseTrendingLimit(t *testing.T) {
	req, err := http.NewRequest("GET", "/stats/trending/uploads", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	limit, err := parseTrendingLimit(req)
	require.NoError(t, err)
	require.Equal(t, 20, limit)

	req, err = http.NewRequest("GET", "/stats/trending/uploads?limit=5", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	limit, err = parseTrendingLimit(req)
	require.NoError(t, err)
	require.Equal(t, 5, limit)

	req, err = http.NewRequest("GET", "/stats/trending/uploads?limit=101", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	limit, err = parseTrendingLimit(req)
	require.NoError(t, err)
	require.Equal(t, 100, limit)

	req, err = http.NewRequest("GET", "/stats/trending/uploads?limit=0", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	_, err = parseTrendingLimit(req)
	require.Error(t, err)

	req, err = http.NewRequest("GET", "/stats/trending/uploads?limit=wat", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	_, err = parseTrendingLimit(req)
	require.Error(t, err)
}

func TestParseTrendingSort(t *testing.T) {
	for _, path := range []string{
		"/stats/trending/uploads",
		"/stats/trending/uploads?sort=downloads",
		"/stats/trending/uploads?sort=downloadedBytes",
	} {
		req, err := http.NewRequest("GET", path, bytes.NewBuffer([]byte{}))
		require.NoError(t, err, "unable to create new request")
		_, err = parseTrendingSort(req)
		require.NoError(t, err)
	}

	req, err := http.NewRequest("GET", "/stats/trending/uploads?sort=bad", bytes.NewBuffer([]byte{}))
	require.NoError(t, err, "unable to create new request")
	_, err = parseTrendingSort(req)
	require.Error(t, err)
}
