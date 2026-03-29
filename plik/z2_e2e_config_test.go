package plik

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestPath(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().Path = "/root"

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	// Set URL without path prefix - requests should fail with 404
	pc.URL = fmt.Sprintf("http://127.0.0.1:%d", ps.GetConfig().ListenPort)

	bi, err := pc.GetServerVersion()
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "404 page not found", "invalid error")

	pc.URL += "/root"
	bi, err = pc.GetServerVersion()
	require.NoError(t, err, "unable to get plik server version")
	require.Equal(t, common.GetBuildInfo().Version, bi.Version, "unable to get plik server version")
}

func TestMaxFileSize(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().MaxFileSize = 10

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	_, _, err = pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to upload file")

	_, file, err := pc.UploadReader("filename", bytes.NewBufferString("data data data"))
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "failed to upload at least one file", "invalid error message")
	require.Contains(t, file.Error().Error(), "file too big", "invalid error message")
}

func TestMaxFilePerUploadCreate(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().MaxFilePerUpload = 1

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Create()
	require.NotNil(t, err, "missing error")
	require.Contains(t, err.Error(), "too many files", "invalid error message")
}

func TestMaxFilePerUploadAdd(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().MaxFilePerUpload = 1

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload, _, err := pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to upload file")

	file := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))

	err = upload.Upload()
	common.RequireError(t, err, "failed to upload at least one file")
	common.RequireError(t, file.Error(), "maximum number file per upload reached")

}

func TestAnonymousUploadDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureAuthentication = common.FeatureForced

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	user := common.NewUser("ovh", "id")
	token := user.NewToken()
	err = ps.GetMetadataBackend().CreateUser(user)
	require.NoError(t, err, "unable to start plik server")

	err = pc.NewUpload().Create()
	require.Error(t, err, "should not be able to create anonymous upload")
	require.Contains(t, err.Error(), "anonymous uploads are disabled", "invalid error")

	upload := pc.NewUpload()
	upload.Token = token.Token
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.NotZero(t, upload.ID(), "invalid upload id")
}

func TestDefaultTTL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().DefaultTTL = 26

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.TTL = 0
	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.Equal(t, 26, upload.Metadata().TTL, "invalid upload ttl")
}

func TestTTLNoLimit(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().MaxTTL = -1

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.TTL = -1
	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.Equal(t, -1, upload.Metadata().TTL, "invalid upload ttl")
}

func TestTTLNoLimitDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().MaxTTL = 26

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.TTL = -1
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "cannot set infinite TTL", "invalid error")
}

func TestPasswordDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeaturePassword = common.FeatureDisabled

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.Login = "login"
	upload.Password = "password"
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "upload password protection is disabled", "invalid error")
}

func TestOneShotDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureOneShot = common.FeatureDisabled

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.OneShot = true
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "one shot uploads are disabled", "invalid error")
}

func TestRemovableDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureRemovable = common.FeatureDisabled

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.Removable = true
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "removable uploads are disabled", "invalid error")
}

func TestCommentsForced(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureComments = common.FeatureForced

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Empty comments should be rejected
	upload := pc.NewUpload()
	err = upload.Create()
	require.Error(t, err, "should not be able to create upload without comments")
	require.Contains(t, err.Error(), "upload comments are required", "invalid error")

	// Non-empty comments should be accepted
	upload = pc.NewUpload()
	upload.Comments = "test comment"
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.Equal(t, "test comment", upload.Metadata().Comments, "invalid comments")
}

func TestCommentsDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().FeatureComments = common.FeatureDisabled

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Comments should be stripped when disabled
	upload := pc.NewUpload()
	upload.Comments = "should be stripped"
	upload.AddFileFromReader("filename", bytes.NewBufferString("data"))
	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.Empty(t, upload.Metadata().Comments, "comments should be empty when disabled")
}

func TestValidDownloadDomain(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Create upload before setting download domain — when both PlikDomain and
	// DownloadDomain are set, the middleware blocks non-file requests on the download domain
	_, file, err := pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to create upload")

	// Set download domain after upload creation so the ephemeral port is known
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	// Download should work — /file/ endpoints are allowed on the download domain
	_, err = file.Download()
	require.NoError(t, err, "unable to download file")
}

func TestDownloadDomainAlias(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Create upload before setting download domain — when both PlikDomain and
	// DownloadDomain are set, the middleware blocks non-file requests on the download domain
	_, file, err := pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to create upload")

	// Set download domain + alias after upload creation
	ps.GetConfig().DownloadDomain = "https://plik.root.gg"
	ps.GetConfig().DownloadDomainAlias = []string{fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)}
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	// Download via alias should work — /file/ endpoints are allowed on download domain aliases
	_, err = file.Download()
	require.NoError(t, err, "unable to download file")
}

func TestInvalidDownloadDomain(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().DownloadDomain = "https://plik.root.gg"
	err := ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	err = startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.Equal(t, ps.GetConfig().DownloadDomain, upload.Metadata().DownloadDomain, "invalid upload ttl")

	_, err = file.Download()
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), "Invalid download domain")
}

func TestDownloadDomainOnlyPassesThrough(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Set only DownloadDomain (no PlikDomain) — backward compat: no UI/API blocking
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	// GET /version (API endpoint) should pass through when PlikDomain is not set
	_, err = pc.GetServerVersion()
	require.NoError(t, err, "API should be accessible when only DownloadDomain is set (backward compat)")
}

func TestDownloadDomainBlocksWebapp(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Set both PlikDomain and DownloadDomain — blocking only activates when both are configured
	ps.GetConfig().PlikDomain = "https://plik.root.gg"
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	// GET / (webapp root) should be blocked on the download domain → redirect to PlikDomain
	req, err := http.NewRequest("GET", pc.URL+"/", &bytes.Buffer{})
	require.NoError(t, err, "unable to create request")

	// Don't follow redirects — we want to assert the 302
	noRedirectClient := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noRedirectClient.Do(req)
	require.NoError(t, err, "unable to make request")
	resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode, "webapp root should be redirected on download domain")
	require.Equal(t, "https://plik.root.gg/", resp.Header.Get("Location"), "should redirect to PlikDomain")

	// GET /version (API endpoint) should also be redirected
	req, err = http.NewRequest("GET", pc.URL+"/version", &bytes.Buffer{})
	require.NoError(t, err, "unable to create request")

	resp, err = noRedirectClient.Do(req)
	require.NoError(t, err, "unable to make request")
	resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode, "API endpoint should be redirected on download domain")
	require.Equal(t, "https://plik.root.gg/version", resp.Header.Get("Location"), "should redirect to PlikDomain")
}

func TestDownloadDomainArchive(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Create upload before setting download domain
	upload, _, err := pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to create upload")

	// Set download domain to match the server's listen address
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	// Archive download should work — /archive/ endpoints are allowed on the download domain
	reader, err := upload.DownloadZipArchive()
	require.NoError(t, err, "unable to download archive on download domain")
	defer reader.Close()

	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read archive")
	require.NotEmpty(t, content, "empty archive")
}

func TestUploadWhitelistOK(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().UploadWhitelist = append(ps.GetConfig().UploadWhitelist, "127.0.0.1")
	err := ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	err = startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
}

func TestUploadWhitelistKO(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().UploadWhitelist = append(ps.GetConfig().UploadWhitelist, "1.1.1.1")
	err := ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	err = startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "untrusted source IP address", "invalid error")
}

func TestSourceIpHeader(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().SourceIPHeader = "X-Remote-Ip"
	ps.GetConfig().UploadWhitelist = append(ps.GetConfig().UploadWhitelist, "1.1.1.1")
	err := ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to initialize config")

	err = startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	var req *http.Request
	req, err = http.NewRequest("POST", pc.URL+"/upload", &bytes.Buffer{})
	require.NoError(t, err, "unable to create request")

	_, err = pc.makeRequest(req)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "untrusted source IP address", "invalid error")

	req.Header.Set("X-Remote-Ip", "1.1.1.1")

	_, err = pc.makeRequest(req)
	require.NoError(t, err, "unable to create upload")

}

// TestDownloadURLMetadata verifies that the upload API response includes both:
//   - DownloadDomain: the raw domain (backward compat)
//   - DownloadURL: the domain + Path (new field)
func TestDownloadURLMetadata(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().Path = "/sub"
	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Set DownloadDomain after server start (port is now known)
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to re-initialize config")

	upload, _, err := pc.UploadReader("filename", bytes.NewBufferString("test data"))
	require.NoError(t, err, "unable to create upload")

	meta := upload.Metadata()
	require.NotNil(t, meta)

	// DownloadDomain must be the raw domain (no path) for backward compat
	require.Equal(t, ps.GetConfig().DownloadDomain, meta.DownloadDomain,
		"DownloadDomain must be the raw domain without Path")

	// DownloadURL must include the Path prefix
	require.Equal(t, ps.GetConfig().DownloadURL, meta.DownloadURL,
		"DownloadURL must equal DownloadDomain + Path")
	require.Contains(t, meta.DownloadURL, "/sub",
		"DownloadURL must contain the Path prefix")
}

// TestDownloadDomainWithPath verifies that GetURL() on a file returns a URL that
// includes the server Path prefix when DownloadURL is set.
func TestDownloadDomainWithPath(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().Path = "/sub"
	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Set DownloadDomain after server start
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://%s:%d", ps.GetConfig().ListenAddress, ps.GetConfig().ListenPort)
	err = ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "unable to re-initialize config")

	_, file, err := pc.UploadReader("test.txt", bytes.NewBufferString("hello"))
	require.NoError(t, err, "unable to upload file")

	u, err := file.GetURL()
	require.NoError(t, err, "unable to get file URL")

	// The URL must contain /sub/file/ — not just /file/
	require.Contains(t, u.Path, "/sub/file/",
		"file URL must include the Path prefix")
}

// TestDownloadDomainWithPathStripping verifies that Initialize() strips a path
// component from DownloadDomain and uses config.Path instead for DownloadURL.
func TestDownloadDomainWithPathStripping(t *testing.T) {
	ps, _ := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().Path = "/correct"
	// Intentionally put a path in DownloadDomain — should be stripped with a warning
	ps.GetConfig().DownloadDomain = "https://dl.example.com/wrong-path"

	err := ps.GetConfig().Initialize(nil)
	require.NoError(t, err, "Initialize must not error when stripping domain path")

	// The raw DownloadDomain is cleaned
	require.Equal(t, "https://dl.example.com", ps.GetConfig().DownloadDomain)
	// DownloadURL uses config.Path, not the path that was in DownloadDomain
	require.Equal(t, "https://dl.example.com/correct", ps.GetConfig().DownloadURL)
}
