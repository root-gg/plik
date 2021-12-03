package plik

import (
	"bytes"
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

	err := start(ps)
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

	err := start(ps)
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

	err := start(ps)
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

	ps.GetConfig().Authentication = true
	ps.GetConfig().NoAnonymousUploads = true

	err := start(ps)
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

	err := start(ps)
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

	err := start(ps)
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

	err := start(ps)
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

	ps.GetConfig().ProtectedByPassword = false

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.Login = "login"
	upload.Password = "password"
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "password protection is not enabled", "invalid error")
}

func TestOneShotDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().OneShot = false

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.OneShot = true
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "one shot uploads are not enabled", "invalid error")
}

func TestRemovableDisabled(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().Removable = false

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.Removable = true
	err = upload.Create()
	require.Error(t, err, "unable to create upload")
	require.Contains(t, err.Error(), "removable uploads are not enabled", "invalid error")
}

func TestDownloadDomain(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().DownloadDomain = "http://127.0.0.1:13425"
	err := ps.GetConfig().Initialize()
	require.NoError(t, err, "unable to initialize config")

	err = start(ps)
	require.NoError(t, err, "unable to start plik server")

	handler := http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("error_ok"))
	})
	cancel, err := common.StartAPIMockServerCustomPort(13425, handler)
	defer cancel()

	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.Equal(t, ps.GetConfig().DownloadDomain, upload.Metadata().DownloadDomain, "invalid upload ttl")

	_, err = file.Download()
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), "error_ok", "invalid error")
}

func TestUploadWhitelistOK(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	ps.GetConfig().UploadWhitelist = append(ps.GetConfig().UploadWhitelist, "127.0.0.1")
	err := ps.GetConfig().Initialize()
	require.NoError(t, err, "unable to initialize config")

	err = start(ps)
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
	err := ps.GetConfig().Initialize()
	require.NoError(t, err, "unable to initialize config")

	err = start(ps)
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
	err := ps.GetConfig().Initialize()
	require.NoError(t, err, "unable to initialize config")

	err = start(ps)
	require.NoError(t, err, "unable to start plik server")

	var req *http.Request
	req, err = http.NewRequest("POST", pc.URL+"/upload", &bytes.Buffer{})
	require.NoError(t, err, "unable to create request")

	_, err = pc.MakeRequest(req)
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "untrusted source IP address", "invalid error")

	req.Header.Set("X-Remote-Ip", "1.1.1.1")

	_, err = pc.MakeRequest(req)
	require.NoError(t, err, "unable to create upload")

}
