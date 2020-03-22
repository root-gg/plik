package plik

import (
	"fmt"
	"testing"

	"github.com/root-gg/plik/server/common"

	"github.com/stretchr/testify/require"
)

func TestGetUploadURL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()

	_, err = upload.GetURL()
	common.RequireError(t, err, "upload has not been created yet")

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")

	uploadURL, err := upload.GetURL()
	require.NoError(t, err, "unable to get upload URL")
	require.Equal(t, pc.URL+"/#/?id="+upload.ID(), uploadURL.String(), "invalid upload URL")
}

func TestGetUploadAdminURL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()

	_, err = upload.GetAdminURL()
	common.RequireError(t, err, "upload has not been created yet")

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")

	uploadURL, err := upload.GetAdminURL()
	require.NoError(t, err, "unable to get upload URL")
	require.Equal(t, fmt.Sprintf("%s/#/?id=%s&uploadToken=%s", pc.URL, upload.ID(), upload.Metadata().UploadToken), uploadURL.String(), "invalid upload URL")
}
