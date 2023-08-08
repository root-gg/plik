package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsFeatureValid(t *testing.T) {
	require.NoError(t, ValidateFeatureFlag(FeatureDisabled))
	require.NoError(t, ValidateFeatureFlag(FeatureEnabled))
	require.NoError(t, ValidateFeatureFlag(FeatureDefault))
	require.NoError(t, ValidateFeatureFlag(FeatureForced))
	RequireError(t, ValidateFeatureFlag(""), "Invalid feature flag value")
	RequireError(t, ValidateFeatureFlag("invalid"), "Invalid feature flag value")
}

func TestIsFeatureAvailable(t *testing.T) {
	require.False(t, IsFeatureAvailable(FeatureDisabled))
	require.True(t, IsFeatureAvailable(FeatureEnabled))
	require.True(t, IsFeatureAvailable(FeatureDefault))
	require.True(t, IsFeatureAvailable(FeatureForced))
	require.False(t, IsFeatureAvailable(""))
	require.False(t, IsFeatureAvailable("invalid"))
}

func TestIsFeatureDefault(t *testing.T) {
	require.False(t, IsFeatureDefault(FeatureDisabled))
	require.False(t, IsFeatureDefault(FeatureEnabled))
	require.True(t, IsFeatureDefault(FeatureDefault))
	require.True(t, IsFeatureDefault(FeatureForced))
	require.False(t, IsFeatureDefault(""))
	require.False(t, IsFeatureDefault("invalid"))
}

func Test_initializeFeatureAuthentication(t *testing.T) {
	config := NewConfiguration()
	config.FeatureAuthentication = "invalid"
	RequireError(t, config.initializeFeatureAuthentication(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureAuthentication = ""
	require.NoError(t, config.initializeFeatureAuthentication())
	require.Equal(t, FeatureDisabled, config.FeatureAuthentication)

	config = NewConfiguration()
	config.FeatureAuthentication = ""
	config.Authentication = true
	require.NoError(t, config.initializeFeatureAuthentication())
	require.Equal(t, FeatureEnabled, config.FeatureAuthentication)

	config = NewConfiguration()
	config.FeatureAuthentication = ""
	config.Authentication = true
	config.NoAnonymousUploads = true
	require.NoError(t, config.initializeFeatureAuthentication())
	require.Equal(t, FeatureForced, config.FeatureAuthentication)

	config = NewConfiguration()
	config.FeatureAuthentication = FeatureDisabled
	require.NoError(t, config.initializeFeatureAuthentication())
	require.False(t, config.Authentication)
	require.False(t, config.NoAnonymousUploads)

	config = NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	require.NoError(t, config.initializeFeatureAuthentication())
	require.True(t, config.Authentication)
	require.False(t, config.NoAnonymousUploads)

	config = NewConfiguration()
	config.FeatureAuthentication = FeatureDefault
	require.NoError(t, config.initializeFeatureAuthentication())
	require.True(t, config.Authentication)
	require.False(t, config.NoAnonymousUploads)

	config = NewConfiguration()
	config.FeatureAuthentication = FeatureForced
	require.NoError(t, config.initializeFeatureAuthentication())
	require.True(t, config.Authentication)
	require.True(t, config.NoAnonymousUploads)
}

func Test_initializeFeatureOneShot(t *testing.T) {
	config := NewConfiguration()
	config.FeatureOneShot = "invalid"
	RequireError(t, config.initializeFeatureOneShot(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureOneShot = ""
	config.OneShot = false
	require.NoError(t, config.initializeFeatureOneShot())
	require.Equal(t, FeatureDisabled, config.FeatureOneShot)

	config = NewConfiguration()
	config.FeatureOneShot = ""
	config.OneShot = true
	require.NoError(t, config.initializeFeatureOneShot())
	require.Equal(t, FeatureEnabled, config.FeatureOneShot)

	config = NewConfiguration()
	config.FeatureOneShot = FeatureDisabled
	require.NoError(t, config.initializeFeatureOneShot())
	require.False(t, config.OneShot)

	config = NewConfiguration()
	config.FeatureOneShot = FeatureEnabled
	require.NoError(t, config.initializeFeatureOneShot())
	require.True(t, config.OneShot)

	config = NewConfiguration()
	config.FeatureOneShot = FeatureDefault
	require.NoError(t, config.initializeFeatureOneShot())
	require.True(t, config.OneShot)

	config = NewConfiguration()
	config.FeatureOneShot = FeatureForced
	require.NoError(t, config.initializeFeatureOneShot())
	require.True(t, config.OneShot)
}

func Test_initializeFeatureRemovable(t *testing.T) {
	config := NewConfiguration()
	config.FeatureRemovable = "invalid"
	RequireError(t, config.initializeFeatureRemovable(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureRemovable = ""
	config.Removable = false
	require.NoError(t, config.initializeFeatureRemovable())
	require.Equal(t, FeatureDisabled, config.FeatureRemovable)

	config = NewConfiguration()
	config.FeatureRemovable = ""
	config.Removable = true
	require.NoError(t, config.initializeFeatureRemovable())
	require.Equal(t, FeatureEnabled, config.FeatureRemovable)

	config = NewConfiguration()
	config.FeatureRemovable = FeatureDisabled
	require.NoError(t, config.initializeFeatureRemovable())
	require.False(t, config.Removable)

	config = NewConfiguration()
	config.FeatureRemovable = FeatureEnabled
	require.NoError(t, config.initializeFeatureRemovable())
	require.True(t, config.Removable)

	config = NewConfiguration()
	config.FeatureRemovable = FeatureDefault
	require.NoError(t, config.initializeFeatureRemovable())
	require.True(t, config.Removable)

	config = NewConfiguration()
	config.FeatureRemovable = FeatureForced
	require.NoError(t, config.initializeFeatureRemovable())
	require.True(t, config.Removable)
}

func Test_initializeFeatureStream(t *testing.T) {
	config := NewConfiguration()
	config.FeatureStream = "invalid"
	RequireError(t, config.initializeFeatureStream(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureStream = ""
	config.Stream = false
	require.NoError(t, config.initializeFeatureStream())
	require.Equal(t, FeatureDisabled, config.FeatureStream)

	config = NewConfiguration()
	config.FeatureStream = ""
	config.Stream = true
	require.NoError(t, config.initializeFeatureStream())
	require.Equal(t, FeatureEnabled, config.FeatureStream)

	config = NewConfiguration()
	config.FeatureStream = FeatureDisabled
	require.NoError(t, config.initializeFeatureStream())
	require.False(t, config.Stream)

	config = NewConfiguration()
	config.FeatureStream = FeatureEnabled
	require.NoError(t, config.initializeFeatureStream())
	require.True(t, config.Stream)

	config = NewConfiguration()
	config.FeatureStream = FeatureDefault
	require.NoError(t, config.initializeFeatureStream())
	require.True(t, config.Stream)

	config = NewConfiguration()
	config.FeatureStream = FeatureForced
	require.NoError(t, config.initializeFeatureStream())
	require.True(t, config.Stream)
}

func Test_initializeFeaturePassword(t *testing.T) {
	config := NewConfiguration()
	config.FeaturePassword = "invalid"
	RequireError(t, config.initializeFeaturePassword(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeaturePassword = ""
	config.ProtectedByPassword = false
	require.NoError(t, config.initializeFeaturePassword())
	require.Equal(t, FeatureDisabled, config.FeaturePassword)

	config = NewConfiguration()
	config.FeaturePassword = ""
	config.ProtectedByPassword = true
	require.NoError(t, config.initializeFeaturePassword())
	require.Equal(t, FeatureEnabled, config.FeaturePassword)

	config = NewConfiguration()
	config.FeaturePassword = FeatureDisabled
	require.NoError(t, config.initializeFeaturePassword())
	require.False(t, config.ProtectedByPassword)

	config = NewConfiguration()
	config.FeaturePassword = FeatureEnabled
	require.NoError(t, config.initializeFeaturePassword())
	require.True(t, config.ProtectedByPassword)

	config = NewConfiguration()
	config.FeaturePassword = FeatureDefault
	require.NoError(t, config.initializeFeaturePassword())
	require.True(t, config.ProtectedByPassword)

	config = NewConfiguration()
	config.FeaturePassword = FeatureForced
	require.NoError(t, config.initializeFeaturePassword())
	require.True(t, config.ProtectedByPassword)
}

func Test_initializeFeatureComments(t *testing.T) {
	config := NewConfiguration()
	config.FeatureComments = "invalid"
	RequireError(t, config.initializeFeatureComments(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureComments = ""
	require.NoError(t, config.initializeFeatureComments())
	require.Equal(t, FeatureEnabled, config.FeatureComments)

	config = NewConfiguration()
	config.FeatureComments = FeatureForced
	require.NoError(t, config.initializeFeatureComments())
	require.Equal(t, FeatureForced, config.FeatureComments)
}

func Test_initializeFeatureSetTTL(t *testing.T) {
	config := NewConfiguration()
	config.FeatureSetTTL = "invalid"
	RequireError(t, config.initializeFeatureSetTTL(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureSetTTL = ""
	require.NoError(t, config.initializeFeatureSetTTL())
	require.Equal(t, FeatureEnabled, config.FeatureSetTTL)

	config = NewConfiguration()
	config.FeatureSetTTL = FeatureForced
	require.NoError(t, config.initializeFeatureSetTTL())
	require.Equal(t, FeatureForced, config.FeatureSetTTL)
}

func Test_initializeFeatureExtendTTL(t *testing.T) {
	config := NewConfiguration()
	config.FeatureExtendTTL = "invalid"
	RequireError(t, config.initializeFeatureExtendTTL(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureExtendTTL = ""
	require.NoError(t, config.initializeFeatureExtendTTL())
	require.Equal(t, FeatureDisabled, config.FeatureExtendTTL)

	config = NewConfiguration()
	config.FeatureExtendTTL = FeatureForced
	require.NoError(t, config.initializeFeatureExtendTTL())
	require.Equal(t, FeatureForced, config.FeatureExtendTTL)
}

func Test_initializeFeatureClients(t *testing.T) {
	config := NewConfiguration()
	config.FeatureClients = "invalid"
	RequireError(t, config.initializeFeatureClients(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureClients = ""
	require.NoError(t, config.initializeFeatureClients())
	require.Equal(t, FeatureEnabled, config.FeatureClients)

	config = NewConfiguration()
	config.FeatureClients = FeatureDisabled
	require.NoError(t, config.initializeFeatureClients())
	require.Equal(t, FeatureDisabled, config.FeatureClients)

	config = NewConfiguration()
	config.FeatureClients = FeatureForced
	RequireError(t, config.initializeFeatureClients(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureClients = FeatureDefault
	RequireError(t, config.initializeFeatureClients(), "Invalid feature flag value")
}

func Test_initializeFeatureGithub(t *testing.T) {
	config := NewConfiguration()
	config.FeatureGithub = "invalid"
	RequireError(t, config.initializeFeatureGithub(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureGithub = ""
	require.NoError(t, config.initializeFeatureGithub())
	require.Equal(t, FeatureEnabled, config.FeatureGithub)

	config = NewConfiguration()
	config.FeatureGithub = FeatureDisabled
	require.NoError(t, config.initializeFeatureGithub())
	require.Equal(t, FeatureDisabled, config.FeatureGithub)

	config = NewConfiguration()
	config.FeatureGithub = FeatureForced
	RequireError(t, config.initializeFeatureGithub(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureClients = FeatureDefault
	RequireError(t, config.initializeFeatureClients(), "Invalid feature flag value")
}

func Test_initializeFeatureText(t *testing.T) {
	config := NewConfiguration()
	config.FeatureText = "invalid"
	RequireError(t, config.initializeFeatureText(), "Invalid feature flag value")

	config = NewConfiguration()
	config.FeatureText = ""
	require.NoError(t, config.initializeFeatureText())
	require.Equal(t, FeatureEnabled, config.FeatureText)

	config = NewConfiguration()
	config.FeatureText = FeatureDisabled
	require.NoError(t, config.initializeFeatureText())
	require.Equal(t, FeatureDisabled, config.FeatureText)

	config = NewConfiguration()
	config.FeatureText = FeatureForced
	require.NoError(t, config.initializeFeatureText())
	require.Equal(t, FeatureForced, config.FeatureText)

	config = NewConfiguration()
	config.FeatureText = FeatureDefault
	require.NoError(t, config.initializeFeatureText())
	require.Equal(t, FeatureDefault, config.FeatureText)
}

func Test_initializeFeatureFlags(t *testing.T) {
	config := NewConfiguration()
	require.NoError(t, config.initializeFeatureFlags())

	require.NoError(t, ValidateFeatureFlag(config.FeatureAuthentication))
	require.NoError(t, ValidateFeatureFlag(config.FeatureOneShot))
	require.NoError(t, ValidateFeatureFlag(config.FeatureRemovable))
	require.NoError(t, ValidateFeatureFlag(config.FeatureStream))
	require.NoError(t, ValidateFeatureFlag(config.FeatureComments))
	require.NoError(t, ValidateFeatureFlag(config.FeatureSetTTL))
	require.NoError(t, ValidateFeatureFlag(config.FeatureExtendTTL))
	require.NoError(t, ValidateFeatureFlag(config.FeatureGithub))
	require.NoError(t, ValidateFeatureFlag(config.FeatureClients))
	require.NoError(t, ValidateFeatureFlag(config.FeatureText))

	config = NewConfiguration()
	config.FeatureOneShot = "invalid"
	RequireError(t, config.initializeFeatureFlags(), "Invalid feature flag value")
}
