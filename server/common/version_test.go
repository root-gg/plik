package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBuildInfo(t *testing.T) {
	buildInfo := GetBuildInfo()
	require.NotNil(t, buildInfo, "missing build info")
	require.NotZero(t, buildInfo.Version, "missing build info version")
}

func TestGetBuildInfoString(t *testing.T) {
	buildInfo := GetBuildInfo()
	version := buildInfo.String()
	require.NotZero(t, version, "invalid build string")
}
