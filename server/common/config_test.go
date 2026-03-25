package common

import (
	"crypto/tls"
	"net"
	"os"
	"testing"

	"github.com/root-gg/logger"

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
	config.FeatureAuthentication = FeatureEnabled
	config.GoogleAPIClientID = "google_api_client_id"
	config.GoogleAPISecret = "google_api_secret"
	config.OvhAPIKey = "ovh_api_key"
	config.OvhAPISecret = "ovh_api_secret"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")
}

func TestInitializeConfigAuthenticationNoMethod(t *testing.T) {
	// Auth enabled but local login disabled and no OAuth configured → should fail
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.FeatureLocalLogin = FeatureDisabled

	err := config.Initialize()
	RequireError(t, err, "no authentication method is available")

	// Auth forced but local login disabled and no OAuth configured → should fail
	config = NewConfiguration()
	config.FeatureAuthentication = FeatureForced
	config.FeatureLocalLogin = FeatureDisabled

	err = config.Initialize()
	RequireError(t, err, "no authentication method is available")

	// Auth disabled → should be fine regardless
	config = NewConfiguration()
	config.FeatureAuthentication = FeatureDisabled
	config.FeatureLocalLogin = FeatureDisabled

	err = config.Initialize()
	require.NoError(t, err, "should be able to initialize with auth disabled")

	// Auth enabled + local login enabled → should be fine
	config = NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.FeatureLocalLogin = FeatureEnabled

	err = config.Initialize()
	require.NoError(t, err, "should be able to initialize with local login")
}

func TestInitializeConfigDefaultAdminValid(t *testing.T) {
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.DefaultAdminLogin = "admin"
	config.DefaultAdminPassword = "s3cr3tpass"

	err := config.Initialize()
	require.NoError(t, err, "valid default admin config should initialize successfully")
}

func TestInitializeConfigDefaultAdminNoAuth(t *testing.T) {
	// DefaultAdminLogin set but FeatureAuthentication is disabled → should fail
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureDisabled
	config.DefaultAdminLogin = "admin"

	err := config.Initialize()
	RequireError(t, err, "DefaultAdminLogin is set but FeatureAuthentication is disabled")
}

func TestInitializeConfigDefaultAdminLocalLoginDisabled(t *testing.T) {
	// DefaultAdminLogin set but FeatureLocalLogin is disabled → should fail
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.FeatureLocalLogin = FeatureDisabled
	config.GoogleAPIClientID = "id"
	config.GoogleAPISecret = "secret"
	config.DefaultAdminLogin = "admin"

	err := config.Initialize()
	RequireError(t, err, "DefaultAdminLogin is set but FeatureLocalLogin is disabled")
}

func TestInitializeConfigDefaultAdminLoginTooShort(t *testing.T) {
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.DefaultAdminLogin = "adm" // 3 chars, min is 4

	err := config.Initialize()
	RequireError(t, err, "DefaultAdminLogin is too short")
}

func TestInitializeConfigDefaultAdminPasswordTooShort(t *testing.T) {
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.DefaultAdminLogin = "admin"
	config.DefaultAdminPassword = "short" // 5 chars, min is 8

	err := config.Initialize()
	RequireError(t, err, "DefaultAdminPassword is too short")
}

func TestInitializeConfigDefaultAdminPasswordEmpty(t *testing.T) {
	// Empty password is allowed (will be auto-generated at runtime)
	config := NewConfiguration()
	config.FeatureAuthentication = FeatureEnabled
	config.DefaultAdminLogin = "admin"
	// DefaultAdminPassword intentionally left empty

	err := config.Initialize()
	require.NoError(t, err, "empty password should be allowed (auto-generated at startup)")
}

func TestInitializeConfigPlikDomain(t *testing.T) {
	config := NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")
	require.NotNil(t, config.GetPlikDomain())
	require.Equal(t, "plik.root.gg", config.GetPlikDomain().Host)
}

func TestInitializeConfigInvalidPlikDomain(t *testing.T) {
	config := NewConfiguration()
	config.PlikDomain = ":/invalid"

	err := config.Initialize()
	require.Error(t, err, "able to initialize invalid config")
}

func TestInitializeConfigDownloadDomain(t *testing.T) {
	config := NewConfiguration()
	config.DownloadDomain = "https://dl.plik.root.gg"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize config")
	require.Equal(t, config.DownloadDomain, config.GetDownloadDomain().String(), "invalid download domain")
}

func TestInitializeConfigPlikDomainEqualsDownloadDomain(t *testing.T) {
	config := NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://plik.root.gg"

	err := config.Initialize()
	RequireError(t, err, "PlikDomain and DownloadDomain must be different domains")
}

func TestInitializeConfigPlikDomainEqualsDownloadDomainAlias(t *testing.T) {
	config := NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	config.DownloadDomainAlias = []string{"https://plik.root.gg"}

	err := config.Initialize()
	RequireError(t, err, "PlikDomain and DownloadDomain must be different domains")
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

func TestInitializeMaxFileSizeUnlimited(t *testing.T) {
	config := NewConfiguration()
	config.MaxFileSizeStr = "unlimited"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, int64(-1), config.MaxFileSize, "invalid max file size")

	config = NewConfiguration()
	config.MaxFileSizeStr = "-1"

	err = config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, int64(-1), config.MaxFileSize, "invalid max file size")
}

func TestInitializeMaxUserSizeString(t *testing.T) {
	config := NewConfiguration()
	config.MaxUserSizeStr = "100 MB"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, int64(100*1000*1000), config.MaxUserSize, "invalid max file size")
}

func TestInitializeMaxUserSizeUnlimited(t *testing.T) {
	config := NewConfiguration()
	config.MaxUserSizeStr = "unlimited"

	err := config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, int64(-1), config.MaxUserSize, "invalid max file size")

	config = NewConfiguration()
	config.MaxUserSizeStr = "-1"

	err = config.Initialize()
	require.NoError(t, err, "unable to initialize valid config")

	require.Equal(t, int64(-1), config.MaxUserSize, "invalid max file size")
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

func TestGetServerUrlWithPlikDomain(t *testing.T) {
	config := NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	err := config.Initialize()
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg", config.GetServerURL().String())

	config.Path = "/sub"
	require.Equal(t, "https://plik.root.gg/sub", config.GetServerURL().String())
}

func TestString(t *testing.T) {
	config := NewConfiguration()
	require.NotEmpty(t, config.String())

	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "download.domain"
	config.DefaultTTL = -1
	config.MaxTTL = -1
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
	require.EqualValues(t, map[string]any{"path": "files"}, config.MetadataBackendConfig)
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

func TestConfiguration_GetSessionTimeout(t *testing.T) {
	config := NewConfiguration()
	require.Equal(t, 0, config.GetSessionTimeout())

	err := config.Initialize()
	require.NoError(t, err)
	require.Equal(t, 365*24*60*60, config.GetSessionTimeout())

	config = NewConfiguration()
	config.SessionTimeout = "30d"
	err = config.Initialize()
	require.NoError(t, err)
	require.Equal(t, 30*24*60*60, config.GetSessionTimeout())

	config = NewConfiguration()
	config.SessionTimeout = ""
	err = config.Initialize()
	RequireError(t, err, "unable to parse SessionTimeout")

	config = NewConfiguration()
	config.SessionTimeout = "-1"
	err = config.Initialize()
	RequireError(t, err, "invalid negative or zero value for SessionTimeout")

	config = NewConfiguration()
	config.SessionTimeout = "azerty"
	err = config.Initialize()
	RequireError(t, err, "unable to parse SessionTimeout")
}

func TestConfiguration_GetStreamTimeout(t *testing.T) {
	config := NewConfiguration()
	require.Equal(t, 0, config.GetStreamTimeout())

	err := config.Initialize()
	require.NoError(t, err)
	require.Equal(t, 5*60, config.GetStreamTimeout()) // default "5m"

	config = NewConfiguration()
	config.StreamTimeoutStr = "10m"
	err = config.Initialize()
	require.NoError(t, err)
	require.Equal(t, 10*60, config.GetStreamTimeout())

	config = NewConfiguration()
	config.StreamTimeoutStr = "0"
	err = config.Initialize()
	require.NoError(t, err)
	require.Equal(t, 0, config.GetStreamTimeout()) // disabled

	config = NewConfiguration()
	config.StreamTimeoutStr = "azerty"
	err = config.Initialize()
	RequireError(t, err, "unable to parse StreamTimeout")

	config = NewConfiguration()
	config.StreamTimeoutStr = "-1"
	err = config.Initialize()
	RequireError(t, err, "invalid negative value for StreamTimeout")
}

func TestConfiguration_GetPath(t *testing.T) {
	config := NewConfiguration()
	require.Equal(t, "/", config.GetPath())
	config.Path = "/path"
	require.Equal(t, "/path", config.GetPath())
}

func TestConfiguration_IsValidDownloadDomain(t *testing.T) {
	config := NewConfiguration()
	err := config.Initialize()
	require.NoError(t, err)

	require.Nil(t, config.downloadDomainURL)
	require.True(t, config.IsValidDownloadDomain("plik.root.gg"))
	require.True(t, config.IsValidDownloadDomain("invalid.domain"))

	config = NewConfiguration()
	config.DownloadDomain = "https://plik.root.gg"
	err = config.Initialize()
	require.NoError(t, err)

	require.NotNil(t, config.downloadDomainURL)
	require.True(t, config.IsValidDownloadDomain("plik.root.gg"))
	require.False(t, config.IsValidDownloadDomain("invalid.domain"))

	config = NewConfiguration()
	config.DownloadDomain = "https://plik.root.gg"
	config.DownloadDomainAlias = []string{"https://dl.root.gg"}
	err = config.Initialize()
	require.NoError(t, err)

	require.NotNil(t, config.downloadDomainURL)
	require.NotEmpty(t, config.downloadDomainURLAlias)
	require.True(t, config.IsValidDownloadDomain("plik.root.gg"))
	require.True(t, config.IsValidDownloadDomain("dl.root.gg"))
	require.False(t, config.IsValidDownloadDomain("invalid.domain"))
}

func TestConfiguration_GetCORSOrigin(t *testing.T) {
	// No domains → no CORS
	config := NewConfiguration()
	err := config.Initialize()
	require.NoError(t, err)
	require.Equal(t, "", config.GetCORSOrigin())

	// PlikDomain only → no CORS
	config = NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	err = config.Initialize()
	require.NoError(t, err)
	require.Equal(t, "", config.GetCORSOrigin())

	// Both → CORS returns PlikDomain
	config = NewConfiguration()
	config.PlikDomain = "https://plik.root.gg"
	config.DownloadDomain = "https://dl.plik.root.gg"
	err = config.Initialize()
	require.NoError(t, err)
	require.Equal(t, "https://plik.root.gg", config.GetCORSOrigin())
}

func TestGetTlsVersionDefault(t *testing.T) {
	config := NewConfiguration()
	require.Equal(t, uint16(tls.VersionTLS12), config.GetTlsVersion(), "default TLS version should be TLS 1.2")
}

func TestGetTlsVersionAllValues(t *testing.T) {
	config := NewConfiguration()

	config.TlsVersion = "tlsv10"
	require.Equal(t, uint16(tls.VersionTLS10), config.GetTlsVersion())

	config.TlsVersion = "tlsv11"
	require.Equal(t, uint16(tls.VersionTLS11), config.GetTlsVersion())

	config.TlsVersion = "tlsv12"
	require.Equal(t, uint16(tls.VersionTLS12), config.GetTlsVersion())

	config.TlsVersion = "tlsv13"
	require.Equal(t, uint16(tls.VersionTLS13), config.GetTlsVersion())

	// Unknown value should default to TLS 1.2
	config.TlsVersion = "invalid"
	require.Equal(t, uint16(tls.VersionTLS12), config.GetTlsVersion())
}
