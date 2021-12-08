package context

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestContext_ConfigureUploadFromContext(t *testing.T) {
	config := common.NewConfiguration()

	upload := &common.Upload{}
	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetSourceIP(net.IPv4(byte(1), byte(1), byte(1), byte(1)))

	err := ctx.ConfigureUploadFromContext(upload)
	require.NoError(t, err, "unable to configure upload from context")
	require.Equal(t, "1.1.1.1", upload.RemoteIP, "invalid source IP address")
}

func TestContext_ConfigureUploadUser(t *testing.T) {
	config := common.NewConfiguration()
	config.Authentication = true

	upload := &common.Upload{}
	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetUser(&common.User{ID: "foo"})

	err := ctx.ConfigureUploadFromContext(upload)
	require.NoError(t, err, "anonymous uploads are disabled")
	require.Equal(t, "foo", upload.User, "invalid upload user")
}

func TestContext_ConfigureUploadUserAuthDisable(t *testing.T) {
	config := common.NewConfiguration()

	upload := &common.Upload{}
	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetUser(&common.User{ID: "foo"})

	err := ctx.ConfigureUploadFromContext(upload)
	require.Error(t, err, "missing authentication is disabled error")
}

func TestContext_NoAnonymousUploads(t *testing.T) {
	config := common.NewConfiguration()
	config.NoAnonymousUploads = true

	upload := &common.Upload{}
	ctx := &Context{}
	ctx.SetConfig(config)

	err := ctx.ConfigureUploadFromContext(upload)
	require.Errorf(t, err, "missing no anonymous uploads error")
}
