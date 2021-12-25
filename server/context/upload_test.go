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

func TestContext_ConfigureUploadFromContextTTL(t *testing.T) {
	config := common.NewConfiguration()
	config.MaxTTL = 10

	upload := &common.Upload{TTL: 60}
	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetSourceIP(net.IPv4(byte(1), byte(1), byte(1), byte(1)))

	err := ctx.ConfigureUploadFromContext(upload)
	common.RequireError(t, err, "invalid TTL")
}

func TestContext_ConfigureUploadFromContextUserTTL(t *testing.T) {
	config := common.NewConfiguration()
	config.Authentication = true

	upload := &common.Upload{TTL: 60}
	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetSourceIP(net.IPv4(byte(1), byte(1), byte(1), byte(1)))
	ctx.SetUser(&common.User{ID: "user", MaxTTL: 10})

	err := ctx.ConfigureUploadFromContext(upload)
	common.RequireError(t, err, "invalid TTL")
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

func TestContext_GetMaxFileSize(t *testing.T) {
	config := common.NewConfiguration()
	config.MaxFileSize = 10

	ctx := &Context{}
	ctx.SetConfig(config)

	require.Equal(t, int64(10), ctx.GetMaxFileSize(), "invalid max file size")

	ctx.SetUser(&common.User{MaxFileSize: 0})
	require.Equal(t, int64(10), ctx.GetMaxFileSize(), "invalid max file size")

	ctx.SetUser(&common.User{MaxFileSize: 100})
	require.Equal(t, int64(100), ctx.GetMaxFileSize(), "invalid max file size")
}

func TestContext_GetMaxTTL(t *testing.T) {
	config := common.NewConfiguration()
	config.MaxTTL = 10

	ctx := &Context{}
	ctx.SetConfig(config)

	require.Equal(t, 10, ctx.GetMaxTTL(), "invalid max TTL")

	ctx.SetUser(&common.User{MaxTTL: 0})
	require.Equal(t, 10, ctx.GetMaxTTL(), "invalid max TTL")

	ctx.SetUser(&common.User{MaxTTL: 100})
	require.Equal(t, 100, ctx.GetMaxTTL(), "invalid max TTL")
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
