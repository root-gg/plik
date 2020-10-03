package common

import (
	"fmt"
	"strings"
	"testing"

	"github.com/root-gg/plik/version"

	"github.com/stretchr/testify/require"
)

func TestGetBuildInfo(t *testing.T) {
	buildInfo := GetBuildInfo()
	require.NotNil(t, buildInfo, "missing build info")
	require.NotZero(t, buildInfo.Version, "missing build info version")
}

func TestGetBuildInfoString(t *testing.T) {
	buildInfo := GetBuildInfo()
	buildInfo.GitShortRevision = "foobar"
	v := buildInfo.String()
	require.NotZero(t, v, "empty build string")
	require.True(t, strings.Contains(v, "foobar"), "invalid version string")
}

func TestGetBuildInfoStringSanitize(t *testing.T) {
	buildInfo := GetBuildInfo()
	buildInfo.Sanitize()
	v := buildInfo.String()
	require.Equal(t, fmt.Sprintf("v%s", version.Get()), v, "invalid build string")
}
