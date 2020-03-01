package context

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestIsWhitelistedAlreadyInContext(t *testing.T) {
	ctx := &Context{}

	ctx.SetWhitelisted(false)
	require.False(t, ctx.IsWhitelisted(), "invalid whitelisted status")

	ctx.SetWhitelisted(true)
	require.True(t, ctx.IsWhitelisted(), "invalid whitelisted status")
}

func TestIsWhitelistedNoWhitelist(t *testing.T) {
	config := common.NewConfiguration()
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")

	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetSourceIP(net.ParseIP("1.1.1.1"))

	require.True(t, ctx.IsWhitelisted(), "invalid whitelisted status")
}

func TestIsWhitelistedNoIp(t *testing.T) {
	config := common.NewConfiguration()
	config.UploadWhitelist = append(config.UploadWhitelist, "1.1.1.1")
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")

	ctx := &Context{}
	ctx.SetConfig(config)

	require.False(t, ctx.IsWhitelisted(), "invalid whitelisted status")
}

func TestIsWhitelisted(t *testing.T) {
	config := common.NewConfiguration()
	config.UploadWhitelist = append(config.UploadWhitelist, "1.1.1.1")
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")

	ctx := &Context{}
	ctx.SetConfig(config)
	ctx.SetSourceIP(net.ParseIP("1.1.1.1"))

	require.True(t, ctx.IsWhitelisted(), "invalid whitelisted status")
}
