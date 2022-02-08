package common

import (
	"github.com/root-gg/logger"
	"net"
	"os"
	"testing"

	"github.com/iancoleman/strcase"

	"github.com/stretchr/testify/require"
)

func TestToScreamingSnakeCase(t *testing.T) {
	require.Equal(t, "DEBUG_REQUESTS", strcase.ToScreamingSnake("DebugRequests"))
	require.Equal(t, "DEFAULT_TTL", strcase.ToScreamingSnake("DefaultTTL"))
	require.Equal(t, "GOOGLE_API_CLIENT_ID", strcase.ToScreamingSnake("GoogleAPIClientID"))
}

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

	config = NewConfiguration()
	config.UploadWhitelist = []string{"foo", "bar", "baz"}

	err = config.Initialize()
	RequireError(t, err, "failed to parse upload whitelist")
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

func TestInitializeInvalidDefaultTTL(t *testing.T) {
	config := NewConfiguration()
	config.DefaultTTL = 10 * 86400
	config.MaxTTL = 1 * 86400

	err := config.Initialize()
	require.Error(t, err, "able to initialize invalid config")
}

func TestInitializeInfiniteMaxTTL(t *testing.T) {
	config := NewConfiguration()
	config.DefaultTTL = 10 * 86400
	config.MaxTTL = -1

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")
}

func TestInitializeTTLString(t *testing.T) {
	config := NewConfiguration()
	config.DefaultTTLStr = "3d"
	config.MaxTTLStr = "30d"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, 3*86400, config.DefaultTTL, "invalid default TTL")
	require.Equal(t, 30*86400, config.MaxTTL, "invalid max TTL")
}

func TestInitializeMaxFileSizeString(t *testing.T) {
	config := NewConfiguration()
	config.MaxFileSizeStr = "100 MB"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, int64(100*1000*1000), config.MaxFileSize, "invalid max file size")
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

func TestString(t *testing.T) {
	config := NewConfiguration()
	require.NotEmpty(t, config.String())

	config.DownloadDomain = "download.domain"
	config.OneShot = false
	config.Removable = false
	config.Stream = false
	config.ProtectedByPassword = false
	config.DefaultTTL = -1
	config.MaxTTL = -1
	require.NotEmpty(t, config.String())

	config.Authentication = true
	require.NotEmpty(t, config.String())

	config.GoogleAuthentication = true
	config.OvhAuthentication = true
	config.OvhAPIEndpoint = "api.ovh.com"
	require.NotEmpty(t, config.String())
}

func TestConfiguration_EnvironmentOverride(t *testing.T) {
	defer func() {
		_ = os.Unsetenv(envPrefix + "DEBUG")
		_ = os.Unsetenv(envPrefix + "LISTEN_ADDRESS")
		_ = os.Unsetenv(envPrefix + "MAX_FILE_SIZE")
		_ = os.Unsetenv(envPrefix + "UPLOAD_WHITELIST")
		_ = os.Unsetenv(envPrefix + "METADATA_BACKEND_CONFIG")
	}()

	err := os.Setenv(envPrefix+"DEBUG", "true")
	require.NoError(t, err)

	err = os.Setenv(envPrefix+"LISTEN_ADDRESS", "1.2.3.4")
	require.NoError(t, err)

	err = os.Setenv(envPrefix+"MAX_FILE_SIZE", "42")
	require.NoError(t, err)

	err = os.Setenv(envPrefix+"UPLOAD_WHITELIST", "[\"127.0.0.1\"]")
	require.NoError(t, err)

	err = os.Setenv(envPrefix+"METADATA_BACKEND_CONFIG", "{\"path\": \"files\"}")
	require.NoError(t, err)

	config := NewConfiguration()
	err = config.EnvironmentOverride()
	require.NoError(t, err)

	require.True(t, config.Debug)
	require.Equal(t, "1.2.3.4", config.ListenAddress)
	require.Equal(t, int64(42), config.MaxFileSize)
	require.EqualValues(t, []string{"127.0.0.1"}, config.UploadWhitelist)
	require.EqualValues(t, map[string]interface{}{"path": "files"}, config.MetadataBackendConfig)
}

func TestConfiguration_NewLogger(t *testing.T) {
	config := NewConfiguration()
	log := config.NewLogger()
	require.NotNil(t, log, "invalid nil logger")
	require.Equal(t, logger.INFO, log.MinLevel, "invalid logger level")

	config.Debug = true
	log = config.NewLogger()
	require.Equal(t, logger.DEBUG, log.MinLevel, "invalid logger level")
}

func TestNewConfiguration_InitializeDebugCompat(t *testing.T) {
	config := NewConfiguration()
	config.LogLevel = "DEBUG"
	err := config.Initialize()
	require.NoError(t, err, "initialize error")
	require.True(t, config.Debug)
	require.True(t, config.DebugRequests)
}

func TestParseTTL(t *testing.T) {
	TTL, err := ParseTTL("60")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, 60, TTL)

	TTL, err = ParseTTL("60s")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, 60, TTL)

	TTL, err = ParseTTL("30d")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, 86400*30, TTL)

	TTL, err = ParseTTL("720h")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, 3600*720, TTL)

	TTL, err = ParseTTL("4w")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, 86400*28, TTL)

	TTL, err = ParseTTL("-1")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, -1, TTL)

	TTL, err = ParseTTL("-10d")
	require.NoError(t, err, "parse ttl error")
	require.Equal(t, -1, TTL)

	TTL, err = ParseTTL("foo")
	RequireError(t, err, "unable to parse TTL")
}
