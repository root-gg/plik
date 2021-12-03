package context

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestContext_IsAdmin(t *testing.T) {
	ctx := &Context{}
	require.False(t, ctx.IsAdmin())

	ctx.user = &common.User{}
	require.False(t, ctx.IsAdmin())

	ctx.user.IsAdmin = true
	require.True(t, ctx.IsAdmin())
}
