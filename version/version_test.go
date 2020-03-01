package version

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func getVersionRegex() string {
	return `^\d+\.\d+((\.\d+)|(\-RC\d+))$`
}

func validateVersion(t *testing.T, version string, ok bool) {
	matched, err := regexp.Match(getVersionRegex(), []byte(version))
	require.NoError(t, err, "invalid version regex")

	if ok {
		require.True(t, matched, "invalid version regex match")
	} else {
		require.False(t, matched, "invalid version regex match")
	}
}

func TestValidateVersionRegex(t *testing.T) {
	validateVersion(t, "1.1.1", true)
	validateVersion(t, "1.1-RC1", true)
	validateVersion(t, "1.1.1-RC1", false)
	validateVersion(t, "1.1-rc1", false)
}

func TestGet(t *testing.T) {
	version := Get()
	require.NotZero(t, version, "missing version")
	validateVersion(t, version, true)
}
