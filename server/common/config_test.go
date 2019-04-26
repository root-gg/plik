/*
 * Charles-Antoine Mathieu <charles-antoine.mathieu@ovh.net>
 */

package common

import (
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

func TestInitializeConfigYubikey(t *testing.T) {
	config := NewConfiguration()
	config.YubikeyEnabled = true
	err := config.Initialize()
	require.NoError(t, err, "unable to initialize invalid config")
	require.NotNil(t, config.GetYubiAuth())
}

func TestInitializeConfigInvalid(t *testing.T) {
	config := NewConfiguration()
	config.YubikeyEnabled = true
	config.YubikeyAPIKey = "key"
	config.YubikeyAPISecret = "secret"

	err := config.Initialize()
	require.Error(t, err, "unable to initialize invalid config")
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

func TestConfigIsUserAdmin(t *testing.T) {
	config := NewConfiguration()

	user := NewUser()
	user.ID = "admin"

	require.False(t, config.IsUserAdmin(user), "invalid admin status")

	config.Admins = append(config.Admins, "admin")

	require.True(t, config.IsUserAdmin(user), "invalid admin status")
	require.True(t, config.IsUserAdmin(user), "invalid admin status")
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
