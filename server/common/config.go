package common

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/root-gg/utils"

	"github.com/BurntSushi/toml"
	"github.com/dustin/go-humanize"
	"github.com/iancoleman/strcase"
	str2duration "github.com/xhit/go-str2duration/v2"

	"github.com/root-gg/logger"
)

const envPrefix = "PLIKD_"

// Configuration object
type Configuration struct {
	Debug         bool      `json:"-"`
	DebugRequests bool      `json:"-"`
	LogLevel      string    `json:"-"`
	LogOutput     io.Writer `json:"-"` // Destination for server logs (default: os.Stdout)

	ListenAddress  string `json:"-"`
	ListenPort     int    `json:"-"`
	MetricsAddress string `json:"-"`
	MetricsPort    int    `json:"-"`
	Path           string `json:"-"`

	MaxFileSizeStr   string `json:"-"`
	MaxFileSize      int64  `json:"maxFileSize"`
	MaxUserSizeStr   string `json:"-"`
	MaxUserSize      int64  `json:"maxUserSize"`
	MaxFilePerUpload int    `json:"maxFilePerUpload"`

	DefaultTTLStr string `json:"-"`
	DefaultTTL    int    `json:"defaultTTL"`
	MaxTTLStr     string `json:"-"`
	MaxTTL        int    `json:"maxTTL"`

	SslEnabled bool   `json:"-"`
	SslCert    string `json:"-"`
	SslKey     string `json:"-"`
	TlsVersion string `json:"-"`

	NoWebInterface      bool     `json:"-"`
	PlikDomain          string   `json:"plikDomain"`
	DownloadDomain      string   `json:"downloadDomain"`
	DownloadDomainAlias []string `json:"downloadDomainAlias"`
	DownloadURL         string   `json:"downloadURL,omitempty" toml:"-"` // Computed: DownloadDomain + Path
	EnhancedWebSecurity bool     `json:"-"`
	SessionTimeout      string   `json:"-"`
	StreamTimeoutStr    string   `json:"-"`
	StreamTimeout       int      `json:"streamTimeout"`
	AbuseContact        string   `json:"abuseContact"`
	WebappDirectory     string   `json:"-"`
	ClientsDirectory    string   `json:"-"`
	ChangelogDirectory  string   `json:"-"`

	EnableArchiveCompression bool `json:"-"`

	SourceIPHeader  string   `json:"-"`
	UploadWhitelist []string `json:"-"`

	// Feature Flags
	FeatureAuthentication string `json:"feature_authentication"`
	FeatureLocalLogin     string `json:"feature_local_login"`
	FeatureDeleteAccount  string `json:"feature_delete_account"`
	FeatureOneShot        string `json:"feature_one_shot"`
	FeatureRemovable      string `json:"feature_removable"`
	FeatureStream         string `json:"feature_stream"`
	FeaturePassword       string `json:"feature_password"`
	FeatureComments       string `json:"feature_comments"`
	FeatureSetTTL         string `json:"feature_set_ttl"`
	FeatureExtendTTL      string `json:"feature_extend_ttl"`
	FeatureClients        string `json:"feature_clients"`
	FeatureApiTokens      string `json:"feature_api_tokens"`
	FeatureGithub         string `json:"feature_github"`
	FeatureText           string `json:"feature_text"`
	FeatureE2EE           string `json:"feature_e2ee"`

	// Deprecated Feature Flags
	Authentication      bool `json:"authentication"`      // Deprecated: >1.3.6
	NoAnonymousUploads  bool `json:"noAnonymousUploads"`  // Deprecated: >1.3.6
	OneShot             bool `json:"oneShot"`             // Deprecated: >1.3.6
	Removable           bool `json:"removable"`           // Deprecated: >1.3.6
	Stream              bool `json:"stream"`              // Deprecated: >1.3.6
	ProtectedByPassword bool `json:"protectedByPassword"` // Deprecated: >1.3.6

	GoogleAuthentication bool     `json:"googleAuthentication"`
	GoogleAPISecret      string   `json:"-"`
	GoogleAPIClientID    string   `json:"-"`
	GoogleValidDomains   []string `json:"-"`
	OvhAuthentication    bool     `json:"ovhAuthentication"`
	OvhAPIEndpoint       string   `json:"ovhApiEndpoint"`
	OvhAPIKey            string   `json:"-"`
	OvhAPISecret         string   `json:"-"`

	LocalAuthentication bool `json:"-"`

	OIDCAuthentication       bool     `json:"oidcAuthentication"`
	OIDCClientID             string   `json:"-"`
	OIDCClientSecret         string   `json:"-"`
	OIDCProviderURL          string   `json:"-"`
	OIDCProviderName         string   `json:"oidcProviderName"`
	OIDCValidDomains         []string `json:"-"`
	OIDCRequireVerifiedEmail bool     `json:"-"`

	GitHubAuthentication     bool     `json:"githubAuthentication"`
	GitHubAPIClientID        string   `json:"-"`
	GitHubAPISecret          string   `json:"-"`
	GitHubValidOrganizations []string `json:"-"`

	DefaultAdminLogin    string `json:"-"`
	DefaultAdminPassword string `json:"-"`

	MetadataBackendConfig map[string]any `json:"-"`

	DataBackend       string         `json:"-"`
	DataBackendConfig map[string]any `json:"-"`

	plikDomainURL          *url.URL
	downloadDomainURL      *url.URL
	downloadDomainURLAlias []*url.URL
	uploadWhitelist        []*net.IPNet
	clean                  bool
	sessionTimeout         int
}

// NewConfiguration creates a new configuration
// object with default values
func NewConfiguration() (config *Configuration) {
	config = new(Configuration)
	config.LogLevel = "INFO"

	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8080
	config.MetricsAddress = "0.0.0.0"
	config.MetricsPort = 0
	config.EnhancedWebSecurity = false
	config.SessionTimeout = "365d"
	config.StreamTimeoutStr = "5m"

	config.MaxFileSize = 10000000000 // 10GB
	config.MaxUserSize = -1          // Default max size per user (-1 for unlimited)
	config.MaxFilePerUpload = 1000

	config.DefaultTTL = 2592000 // 30 days
	config.MaxTTL = 2592000     // 30 days

	// Deprecated feature flags default values to ensure backward compatibility <1.3.6
	// New FeatureFlags default values are defined in feature_flags.go initialization functions
	config.OneShot = true
	config.Removable = true
	config.Stream = true
	config.ProtectedByPassword = true

	config.OvhAPIEndpoint = "https://eu.api.ovh.com/1.0"

	config.OIDCProviderName = "OpenID"

	config.EnableArchiveCompression = true

	config.DataBackend = "file"

	config.WebappDirectory = "../webapp/dist"
	config.ClientsDirectory = "../clients"
	config.ChangelogDirectory = "../changelog"

	config.LogOutput = os.Stdout
	config.clean = true
	return
}

// LoadConfiguration creates a new empty configuration
// and try to load specified file with toml library to
// override default params
func LoadConfiguration(path string) (config *Configuration, err error) {
	config = NewConfiguration()

	if path != "" {
		if _, err := toml.DecodeFile(path, config); err != nil {
			return nil, fmt.Errorf("unable to load config file %s : %s", path, err)
		}
	}

	err = config.EnvironmentOverride()
	if err != nil {
		return nil, err
	}

	// Use a minimal bootstrap logger so domain path-stripping warnings reach stdout
	bootstrapLog := config.NewLogger()
	err = config.Initialize(bootstrapLog)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// EnvironmentOverride override config from environment variables
// Environment variables must match config params in screaming snake case ( DebugRequests -> PLIKD_DEBUG_REQUESTS )
func (config *Configuration) EnvironmentOverride() (err error) {
	getEnvOverride := func(fieldName string) (string, bool) {
		return os.LookupEnv(envPrefix + strcase.ToScreamingSnake(fieldName))
	}
	return utils.AssignStrings(config, getEnvOverride)
}

// Initialize config internal parameters.
// log is used to emit warnings for misconfigured domain options (e.g. path components
// in PlikDomain / DownloadDomain). Pass nil to suppress warnings (e.g. in tests).
func (config *Configuration) Initialize(log *logger.Logger) (err error) {

	// For backward compatibility
	if config.LogLevel == "DEBUG" {
		config.Debug = true
		config.DebugRequests = true
	}

	config.Path = strings.TrimSuffix(config.Path, "/")

	// UploadWhitelist is only parsed once at startup time
	for _, cidr := range config.UploadWhitelist {
		if !strings.Contains(cidr, "/") {
			cidr += "/32"
		}
		if _, cidr, err := net.ParseCIDR(cidr); err == nil {
			config.uploadWhitelist = append(config.uploadWhitelist, cidr)
		} else {
			return fmt.Errorf("failed to parse upload whitelist : %s", cidr)
		}
	}

	err = config.initializeFeatureFlags()
	if err != nil {
		return err
	}

	config.GoogleAuthentication = config.FeatureAuthentication != FeatureDisabled && config.GoogleAPIClientID != "" && config.GoogleAPISecret != ""
	config.OvhAuthentication = config.FeatureAuthentication != FeatureDisabled && config.OvhAPIKey != "" && config.OvhAPISecret != ""
	config.OIDCAuthentication = config.FeatureAuthentication != FeatureDisabled && config.OIDCClientID != "" && config.OIDCClientSecret != "" && config.OIDCProviderURL != ""
	config.GitHubAuthentication = config.FeatureAuthentication != FeatureDisabled && config.GitHubAPIClientID != "" && config.GitHubAPISecret != ""
	config.LocalAuthentication = config.FeatureAuthentication != FeatureDisabled && config.FeatureLocalLogin != FeatureDisabled

	if config.DefaultAdminLogin != "" {
		if config.FeatureAuthentication == FeatureDisabled {
			return fmt.Errorf("DefaultAdminLogin is set but FeatureAuthentication is disabled")
		}
		if config.FeatureLocalLogin == FeatureDisabled {
			return fmt.Errorf("DefaultAdminLogin is set but FeatureLocalLogin is disabled")
		}
		if len(config.DefaultAdminLogin) < 4 {
			return fmt.Errorf("DefaultAdminLogin is too short (min 4 chars)")
		}
		if config.DefaultAdminPassword != "" && len(config.DefaultAdminPassword) < 8 {
			return fmt.Errorf("DefaultAdminPassword is too short (min 8 chars)")
		}
	}

	// Validate that at least one authentication method is available when authentication is enabled
	if config.FeatureAuthentication != FeatureDisabled &&
		!config.LocalAuthentication && !config.GoogleAuthentication && !config.OvhAuthentication && !config.OIDCAuthentication && !config.GitHubAuthentication {
		return fmt.Errorf("authentication is enabled but no authentication method is available, enable at least one of : FeatureLocalLogin, Google, OVH, OIDC, or GitHub")
	}

	if config.PlikDomain != "" {
		config.PlikDomain = strings.Trim(config.PlikDomain, "/ ")
		var err error
		if config.plikDomainURL, err = url.Parse(config.PlikDomain); err != nil {
			return fmt.Errorf("invalid plik domain URL %s : %s", config.PlikDomain, err)
		}
		if config.plikDomainURL.Path != "" && config.plikDomainURL.Path != "/" {
			if log != nil {
				log.Warningf("PlikDomain %q contains a path component %q which will be ignored — use the Path config option instead", config.PlikDomain, config.plikDomainURL.Path)
			}
			config.plikDomainURL.Path = ""
			config.PlikDomain = config.plikDomainURL.String()
		}
	}

	if config.DownloadDomain != "" {
		config.DownloadDomain = strings.Trim(config.DownloadDomain, "/ ")
		var err error
		if config.downloadDomainURL, err = url.Parse(config.DownloadDomain); err != nil {
			return fmt.Errorf("invalid download domain URL %s : %s", config.DownloadDomain, err)
		}
		if config.downloadDomainURL.Path != "" && config.downloadDomainURL.Path != "/" {
			if log != nil {
				log.Warningf("DownloadDomain %q contains a path component %q which will be ignored — use the Path config option instead", config.DownloadDomain, config.downloadDomainURL.Path)
			}
			config.downloadDomainURL.Path = ""
			config.DownloadDomain = config.downloadDomainURL.String()
		}

		for _, alias := range config.DownloadDomainAlias {
			domainAlias, err := url.Parse(alias)
			if err != nil {
				return fmt.Errorf("invalid download domain URL %s : %s", domainAlias, err)
			}
			if domainAlias.Path != "" && domainAlias.Path != "/" {
				if log != nil {
					log.Warningf("DownloadDomainAlias %q contains a path component %q which will be ignored", alias, domainAlias.Path)
				}
				domainAlias.Path = ""
			}
			config.downloadDomainURLAlias = append(config.downloadDomainURLAlias, domainAlias)
		}

		if config.plikDomainURL != nil && config.IsDownloadDomain(config.plikDomainURL.Host) {
			return fmt.Errorf("PlikDomain and DownloadDomain must be different domains (%s), using the same domain would cause redirect loops", config.plikDomainURL.Host)
		}

		// Compute the download base URL (domain + Path) for use in generated URLs
		u := *config.downloadDomainURL
		u.Path = config.Path
		config.DownloadURL = u.String()
	}

	if config.MaxFileSizeStr == "unlimited" || config.MaxFileSizeStr == "-1" {
		config.MaxFileSize = int64(-1)
	} else if config.MaxFileSizeStr != "" {
		maxFileSize, err := humanize.ParseBytes(config.MaxFileSizeStr)
		if err != nil {
			return err
		}
		config.MaxFileSize = int64(maxFileSize)
	}

	if config.MaxUserSizeStr == "unlimited" || config.MaxUserSizeStr == "-1" {
		config.MaxUserSize = int64(-1)
	} else if config.MaxUserSizeStr != "" {
		maxUserSize, err := humanize.ParseBytes(config.MaxUserSizeStr)
		if err != nil {
			return err
		}
		config.MaxUserSize = int64(maxUserSize)
	}

	if config.DefaultTTLStr != "" {
		config.DefaultTTL, err = ParseTTL(config.DefaultTTLStr)
		if err != nil {
			return err
		}
	}

	if config.MaxTTLStr != "" {
		config.MaxTTL, err = ParseTTL(config.MaxTTLStr)
		if err != nil {
			return err
		}
	}

	if config.MaxTTL > 0 && config.DefaultTTL > config.MaxTTL {
		return fmt.Errorf("DefaultTTL should not be more than MaxTTL")
	}

	config.sessionTimeout, err = ParseTTL(config.SessionTimeout)
	if err != nil {
		return fmt.Errorf("unable to parse SessionTimeout : %s", err)
	}
	if config.sessionTimeout <= 0 {
		return fmt.Errorf("invalid negative or zero value for SessionTimeout")
	}

	if config.StreamTimeoutStr != "" {
		config.StreamTimeout, err = ParseTTL(config.StreamTimeoutStr)
		if err != nil {
			return fmt.Errorf("unable to parse StreamTimeout : %s", err)
		}
		if config.StreamTimeout < 0 {
			return fmt.Errorf("invalid negative value for StreamTimeout")
		}
	}

	return nil
}

// NewLogger returns a new logger instance
func (config *Configuration) NewLogger() (log *logger.Logger) {
	level := config.LogLevel
	if config.Debug {
		level = "DEBUG"
	}
	return logger.NewLogger().SetMinLevelFromString(level).SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel).SetOutput(config.LogOutput)
}

// GetUploadWhitelist return the parsed IP upload whitelist
func (config *Configuration) GetUploadWhitelist() []*net.IPNet {
	return config.uploadWhitelist
}

// GetPlikDomain return the parsed plik domain URL
func (config *Configuration) GetPlikDomain() *url.URL {
	return config.plikDomainURL
}

// GetDownloadDomain return the parsed download domain URL
func (config *Configuration) GetDownloadDomain() *url.URL {
	return config.downloadDomainURL
}

// GetDownloadDomainAlias return the parsed download domain alias URLs
func (config *Configuration) GetDownloadDomainAlias() []*url.URL {
	return config.downloadDomainURLAlias
}

// GetCORSOrigin returns the Access-Control-Allow-Origin value for download endpoints.
// When both PlikDomain and DownloadDomain are configured, returns the PlikDomain origin
// so the webapp can fetch file content cross-origin. Returns empty string otherwise.
func (config *Configuration) GetCORSOrigin() string {
	if config.plikDomainURL != nil && config.downloadDomainURL != nil {
		return config.PlikDomain
	}
	return ""
}

// IsDownloadDomain returns true if the host matches the configured download domain
// or any of its aliases. Returns false if no download domain is configured.
func (config *Configuration) IsDownloadDomain(host string) bool {
	if config.downloadDomainURL == nil {
		return false
	}

	if config.downloadDomainURL.Host == host {
		return true
	}

	for _, urlAlias := range config.downloadDomainURLAlias {
		if urlAlias.Host == host {
			return true
		}
	}

	return false
}

// IsValidDownloadDomain return whether or not the host is a valid download domain.
// Returns true if no download domain is configured (all hosts are valid).
func (config *Configuration) IsValidDownloadDomain(host string) bool {
	if config.downloadDomainURL == nil {
		return true
	}
	return config.IsDownloadDomain(host)
}

// AutoClean enable or disables the periodical upload cleaning goroutine.
// This needs to be called before Plik server starts to have effect
func (config *Configuration) AutoClean(value bool) {
	config.clean = value
}

// IsAutoClean return whether or not to start the cleaning goroutine
func (config *Configuration) IsAutoClean() bool {
	return config.clean
}

// IsWhitelisted return whether or not the IP matches of the config upload whitelist
func (config *Configuration) IsWhitelisted(ip net.IP) bool {
	if len(config.uploadWhitelist) == 0 {
		// Empty whitelist == accept all
		return true
	}

	// Check if the source IP address is in whitelist
	for _, subnet := range config.uploadWhitelist {
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

// GetServerURL is a helper to get the server HTTP URL.
// When PlikDomain is configured it returns the public-facing URL,
// otherwise falls back to ListenAddress:ListenPort.
func (config *Configuration) GetServerURL() *url.URL {
	if config.plikDomainURL != nil {
		u := *config.plikDomainURL // copy
		if config.Path != "" {
			u.Path = config.Path
		}
		return &u
	}

	URL := &url.URL{}

	if config.SslEnabled {
		URL.Scheme = "https"
	} else {
		URL.Scheme = "http"
	}

	var addr string
	if config.ListenAddress == "0.0.0.0" {
		addr = "127.0.0.1"
	} else {
		addr = config.ListenAddress
	}

	URL.Host = fmt.Sprintf("%s:%d", addr, config.ListenPort)
	URL.Path = config.Path

	return URL
}

// GetDownloadBaseURL returns the base URL for file download links.
// Uses DownloadDomain + Path when configured, otherwise falls back to GetServerURL().
func (config *Configuration) GetDownloadBaseURL() *url.URL {
	if config.downloadDomainURL != nil {
		u := *config.downloadDomainURL
		u.Path = config.Path
		return &u
	}
	return config.GetServerURL()
}

// GetFileURL returns the full download URL for a file.
// When stream is true, uses the /stream/ endpoint instead of /file/.
func (config *Configuration) GetFileURL(uploadID, fileID, fileName string, stream bool) string {
	mode := "file"
	if stream {
		mode = "stream"
	}
	u := config.GetDownloadBaseURL()
	// Set Path (decoded) for correct URL semantics, and RawPath (encoded) so .String()
	// emits exactly one level of percent-encoding (no double-encoding).
	rawSuffix := fmt.Sprintf("/%s/%s/%s/%s", mode, uploadID, fileID, url.PathEscape(fileName))
	decodedSuffix := fmt.Sprintf("/%s/%s/%s/%s", mode, uploadID, fileID, fileName)
	u.RawPath = u.Path + rawSuffix
	u.Path = u.Path + decodedSuffix
	return u.String()
}

// GetArchiveURL returns the full download URL for an upload archive.
func (config *Configuration) GetArchiveURL(uploadID, archiveName string) string {
	u := config.GetDownloadBaseURL()
	rawSuffix := fmt.Sprintf("/archive/%s/%s", uploadID, url.PathEscape(archiveName))
	decodedSuffix := fmt.Sprintf("/archive/%s/%s", uploadID, archiveName)
	u.RawPath = u.Path + rawSuffix
	u.Path = u.Path + decodedSuffix
	return u.String()
}

// GetTlsVersion is a helper to get the TLS version
func (config *Configuration) GetTlsVersion() uint16 {
	if config.TlsVersion == "tlsv10" {
		return tls.VersionTLS10
	}
	if config.TlsVersion == "tlsv11" {
		return tls.VersionTLS11
	}
	if config.TlsVersion == "tlsv12" {
		return tls.VersionTLS12
	}
	if config.TlsVersion == "tlsv13" {
		return tls.VersionTLS13
	}

	return tls.VersionTLS12
}

// GetPath return the web API/UI root path
func (config *Configuration) GetPath() string {
	if config.Path == "" {
		return "/"
	}
	return config.Path
}

// GetSessionTimeout return parsed session timeout
func (config *Configuration) GetSessionTimeout() int {
	return config.sessionTimeout
}

// GetStreamTimeout return parsed stream timeout in seconds (0 = disabled)
func (config *Configuration) GetStreamTimeout() int {
	return config.StreamTimeout
}

func (config *Configuration) String() string {
	str := ""
	if config.PlikDomain != "" {
		str += fmt.Sprintf("Plik domain : %s\n", config.PlikDomain)
	}
	if config.DownloadDomain != "" {
		str += fmt.Sprintf("Download domain : %s\n", config.DownloadDomain)
		if len(config.DownloadDomainAlias) > 0 {
			str += fmt.Sprintf("Download domain alias: %v\n", config.DownloadDomainAlias)
		}
	}

	str += fmt.Sprintf("Maximum file size : %s\n", humanize.Bytes(uint64(config.MaxFileSize)))
	str += fmt.Sprintf("Maximum files per upload : %d\n", config.MaxFilePerUpload)

	if config.DefaultTTL > 0 {
		str += fmt.Sprintf("Default upload TTL : %s\n", HumanDuration(time.Duration(config.DefaultTTL)*time.Second))
	} else {
		str += "Default upload TTL : unlimited\n"
	}

	if config.MaxTTL > 0 {
		str += fmt.Sprintf("Maximum upload TTL : %s\n", HumanDuration(time.Duration(config.MaxTTL)*time.Second))
	} else {
		str += "Maximum upload TTL : unlimited\n"
	}

	str += fmt.Sprintf("One shot upload : %s\n", config.FeatureOneShot)
	str += fmt.Sprintf("Removable upload : %s\n", config.FeatureRemovable)
	str += fmt.Sprintf("Streaming upload : %s\n", config.FeatureStream)
	if config.StreamTimeout > 0 {
		str += fmt.Sprintf("Stream timeout : %s\n", config.StreamTimeoutStr)
	} else {
		str += "Stream timeout : disabled\n"
	}
	str += fmt.Sprintf("Upload password : %s\n", config.FeaturePassword)
	str += fmt.Sprintf("Upload comments : %s\n", config.FeatureComments)
	str += fmt.Sprintf("Upload set TTL : %s\n", config.FeatureSetTTL)
	str += fmt.Sprintf("Upload extend TTL : %s\n", config.FeatureExtendTTL)
	str += fmt.Sprintf("E2E encryption : %s\n", config.FeatureE2EE)
	str += fmt.Sprintf("Delete account : %s\n", config.FeatureDeleteAccount)
	str += fmt.Sprintf("Archive compression : %t\n", config.EnableArchiveCompression)

	str += fmt.Sprintf("Authentication : %s\n", config.FeatureAuthentication)
	if config.FeatureAuthentication != FeatureDisabled {
		if config.GoogleAuthentication {
			str += "Google authentication : enabled\n"
		} else {
			str += "Google authentication : disabled\n"
		}

		if config.OvhAuthentication {
			str += "OVH authentication : enabled\n"
			if config.OvhAPIEndpoint != "" {
				str += fmt.Sprintf("OVH API endpoint : %s\n", config.OvhAPIEndpoint)
			}
		} else {
			str += "OVH authentication : disabled\n"
		}

		if config.OIDCAuthentication {
			str += fmt.Sprintf("OIDC authentication : enabled (%s)\n", config.OIDCProviderName)
			str += fmt.Sprintf("OIDC provider URL : %s\n", config.OIDCProviderURL)
		} else {
			str += "OIDC authentication : disabled\n"
		}

		if config.GitHubAuthentication {
			str += "GitHub authentication : enabled\n"
		} else {
			str += "GitHub authentication : disabled\n"
		}

		str += fmt.Sprintf("Local login : %s\n", config.FeatureLocalLogin)
	}

	return str
}

// ParseTTL string into a number of seconds
func ParseTTL(TTL string) (int, error) {
	// For backward compatibility input without units are in seconds
	_, err := strconv.Atoi(TTL)
	if err == nil {
		TTL += "s"
	}

	duration, err := str2duration.ParseDuration(TTL)
	if err != nil {
		return 0, fmt.Errorf("unable to parse TTL : %s", err)
	}

	if duration < 0 {
		return -1, nil
	}

	return int(duration.Seconds()), nil
}
