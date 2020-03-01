package common

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test new configuration
func TestNewConfig(t *testing.T) {
	config := NewConfiguration()
	require.NotNil(t, config, "invalid config")
}

// Test loading the default configuration
func TestLoadConfig(t *testing.T) {
	_, err := LoadConfiguration("../plikd.cfg")
	require.NoError(t, err, "unable to load config")
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := LoadConfiguration("invalid_config_path")
	require.Error(t, err, "unable to load config")
}

func TestInitializeConfigUploadWhitelist(t *testing.T) {
	config := NewConfiguration()
	config.UploadWhitelist = []string{"1.1.1.1", "127.0.0.0/24", "127.0.0.10/24"}

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize invalid config")

	require.Equal(t, len(config.UploadWhitelist), len(config.GetUploadWhitelist()), "invalid parsed upload whitelist length")
	require.Equal(t, config.UploadWhitelist[0]+"/32", config.uploadWhitelist[0].String(), "invalid parsed upload IP")
	require.Equal(t, config.UploadWhitelist[1], config.uploadWhitelist[1].String(), "invalid parsed upload IP")
	require.Equal(t, config.UploadWhitelist[1], config.uploadWhitelist[2].String(), "invalid parsed upload IP")
}

func TestIsWhitelisted(t *testing.T) {
	config := NewConfiguration()

	require.True(t, config.IsWhitelisted(nil), "no whitelist should be always ok")
	require.True(t, config.IsWhitelisted(net.ParseIP("1.2.3.4").To4()), "no whitelist should be always ok")
	require.True(t, config.IsWhitelisted(net.ParseIP("1234::1").To16()), "no whitelist should be always ok")

	config.UploadWhitelist = []string{"1.1.1.1", "127.0.0.0/24", "1234::/64"}
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize invalid config")

	require.False(t, config.IsWhitelisted(nil), "should not be whitelisted")
	require.False(t, config.IsWhitelisted(net.ParseIP("1.2.3.4").To4()), "should not be whitelisted")
	require.False(t, config.IsWhitelisted(net.ParseIP("666::").To16()), "should not be whitelisted")

	require.True(t, config.IsWhitelisted(net.ParseIP("1.1.1.1").To4()), "no be whitelisted")
	require.True(t, config.IsWhitelisted(net.ParseIP("127.0.0.42").To4()), "no be whitelisted")
	require.True(t, config.IsWhitelisted(net.ParseIP("1234::42").To16()), "no be whitelisted")
}

func TestInitializeConfigAuthentication(t *testing.T) {
	config := NewConfiguration()
	config.GoogleAPIClientID = "google_api_client_id"
	config.GoogleAPISecret = "google_api_secret"
	config.OvhAPIKey = "ovh_api_key"
	config.OvhAPISecret = "ovh_api_secret"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")
}

func TestInitializeConfigDownloadDomain(t *testing.T) {
	config := NewConfiguration()
	config.DownloadDomain = "https://dl.plik.root.gg"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")
	require.Equal(t, config.DownloadDomain, config.GetDownloadDomain().String(), "invalid download domain")
}

func TestInitializeConfigInvalidDownloadDomain(t *testing.T) {
	config := NewConfiguration()
	config.DownloadDomain = ":/invalid"

	err := config.Initialize()
	require.Error(t, err, "able to initialize invalid config")
}

func TestDisableAutoClean(t *testing.T) {
	config := NewConfiguration()
	require.True(t, config.IsAutoClean(), "invalid auto clean status")
	config.AutoClean(false)
	require.False(t, config.IsAutoClean(), "invalid auto clean status")
}

func TestGetServerUrl(t *testing.T) {
	config := NewConfiguration()
	require.Equal(t, "http://127.0.0.1:8080", config.GetServerURL().String(), "invalid server url")
	config.SslEnabled = true
	require.Equal(t, "https://127.0.0.1:8080", config.GetServerURL().String(), "invalid server url")
	config.ListenAddress = "1.1.1.1"
	require.Equal(t, "https://1.1.1.1:8080", config.GetServerURL().String(), "invalid server url")
	config.Path = "/root"
	require.Equal(t, "https://1.1.1.1:8080/root", config.GetServerURL().String(), "invalid server url")
}
