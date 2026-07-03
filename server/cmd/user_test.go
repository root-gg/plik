package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func TestUserNotFoundError(t *testing.T) {
	// Non-OIDC providers get a plain message.
	msg := userNotFoundError(common.ProviderLocal, "local:admin")
	require.Equal(t, "User local:admin not found", msg)

	// OIDC users are keyed by the sub claim, so the message must hint at how
	// to find the right identifier.
	msg = userNotFoundError(common.ProviderOIDC, "oidc:test_user")
	require.Contains(t, msg, "User oidc:test_user not found")
	require.Contains(t, msg, "sub")
	require.Contains(t, msg, "plikd user list")
}
