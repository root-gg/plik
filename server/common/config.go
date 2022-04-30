package common

import (
	"fmt"
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
	Debug         bool   `json:"-"`
	DebugRequests bool   `json:"-"`
	LogLevel      string `json:"-"`

	ListenAddress string `json:"-"`
	ListenPort    int    `json:"-"`
	Path          string `json:"-"`

	MaxFileSizeStr   string `json:"-"`
	MaxFileSize      int64  `json:"maxFileSize"`
	MaxFilePerUpload int    `json:"maxFilePerUpload"`

	DefaultTTLStr string `json:"-"`
	DefaultTTL    int    `json:"defaultTTL"`
	MaxTTLStr     string `json:"-"`
	MaxTTL        int    `json:"maxTTL"`

	SslEnabled bool   `json:"-"`
	SslCert    string `json:"-"`
	SslKey     string `json:"-"`

	NoWebInterface      bool   `json:"-"`
	DownloadDomain      string `json:"downloadDomain"`
	EnhancedWebSecurity bool   `json:"-"`
	AbuseContact        string `json:"abuseContact"`
	WebappDirectory     string `json:"-"`
	ClientsDirectory    string `json:"-"`
	ChangelogDirectory  string `json:"-"`

	SourceIPHeader  string   `json:"-"`
	UploadWhitelist []string `json:"-"`

	// Feature Flags
	FeatureAuthentication string `json:"feature_authentication"`
	FeatureOneShot        string `json:"feature_one_shot"`
	FeatureRemovable      string `json:"feature_removable"`
	FeatureStream         string `json:"feature_stream"`
	FeaturePassword       string `json:"feature_password"`
	FeatureComments       string `json:"feature_comments"`
	FeatureSetTTL         string `json:"feature_set_ttl"`
	FeatureExtendTTL      string `json:"feature_extend_ttl"`

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

	MetadataBackendConfig map[string]interface{} `json:"-"`

	DataBackend       string                 `json:"-"`
	DataBackendConfig map[string]interface{} `json:"-"`

	downloadDomainURL *url.URL
	uploadWhitelist   []*net.IPNet
	clean             bool
}

// NewConfiguration creates a new configuration
// object with default values
func NewConfiguration() (config *Configuration) {
	config = new(Configuration)
	config.LogLevel = "INFO"

	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8080
	config.EnhancedWebSecurity = false

	config.MaxFileSize = 10000000000 // 10GB
	config.MaxFilePerUpload = 1000

	config.DefaultTTL = 2592000 // 30 days
	config.MaxTTL = 2592000     // 30 days

	config.FeatureAuthentication = FeatureDisabled
	config.FeatureOneShot = FeatureEnabled
	config.FeatureRemovable = FeatureEnabled
	config.FeatureStream = FeatureEnabled
	config.FeaturePassword = FeatureEnabled
	config.FeatureComments = FeatureEnabled
	config.FeatureSetTTL = FeatureEnabled
	config.FeatureExtendTTL = FeatureDisabled

	config.OvhAPIEndpoint = "https://eu.api.ovh.com/1.0"

	config.DataBackend = "file"

	config.WebappDirectory = "../webapp/dist"
	config.ClientsDirectory = "../clients"
	config.ChangelogDirectory = "../changelog"

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

	err = config.Initialize()
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

// Initialize config internal parameters
func (config *Configuration) Initialize() (err error) {

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

	if config.DownloadDomain != "" {
		strings.Trim(config.DownloadDomain, "/ ")
		var err error
		if config.downloadDomainURL, err = url.Parse(config.DownloadDomain); err != nil {
			return fmt.Errorf("invalid download domain URL %s : %s", config.DownloadDomain, err)
		}
	}

	if config.MaxFileSizeStr != "" {
		maxFileSize, err := humanize.ParseBytes(config.MaxFileSizeStr)
		if err != nil {
			return err
		}
		config.MaxFileSize = int64(maxFileSize)
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

	return nil
}

// NewLogger returns a new logger instance
func (config *Configuration) NewLogger() (log *logger.Logger) {
	level := config.LogLevel
	if config.Debug {
		level = "DEBUG"
	}
	return logger.NewLogger().SetMinLevelFromString(level).SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel)
}

// GetUploadWhitelist return the parsed IP upload whitelist
func (config *Configuration) GetUploadWhitelist() []*net.IPNet {
	return config.uploadWhitelist
}

// GetDownloadDomain return the parsed download domain URL
func (config *Configuration) GetDownloadDomain() *url.URL {
	return config.downloadDomainURL
}

// AutoClean enable or disables the periodical upload cleaning goroutine.
// This needs to be called before Plik server starts to have effect
func (config *Configuration) AutoClean(value bool) {
	config.clean = value
}

// IsAutoClean return weather or not to start the cleaning goroutine
func (config *Configuration) IsAutoClean() bool {
	return config.clean
}

// IsWhitelisted return weather or not the IP matches of the config upload whitelist
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

// GetServerURL is a helper to get the server HTTP URL
func (config *Configuration) GetServerURL() *url.URL {
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

func (config *Configuration) String() string {
	str := ""
	if config.DownloadDomain != "" {
		str += fmt.Sprintf("Download domain : %s\n", config.DownloadDomain)
	}

	str += fmt.Sprintf("Maximum file size : %s\n", humanize.Bytes(uint64(config.MaxFileSize)))
	str += fmt.Sprintf("Maximum files per upload : %d\n", config.MaxFilePerUpload)

	if config.DefaultTTL > 0 {
		str += fmt.Sprintf("Default upload TTL : %s\n", HumanDuration(time.Duration(config.DefaultTTL)*time.Second))
	} else {
		str += fmt.Sprintf("Default upload TTL : unlimited\n")
	}

	if config.MaxTTL > 0 {
		str += fmt.Sprintf("Maximum upload TTL : %s\n", HumanDuration(time.Duration(config.MaxTTL)*time.Second))
	} else {
		str += fmt.Sprintf("Maximum upload TTL : unlimited\n")
	}

	str += fmt.Sprintf("One shot upload : %s\n", config.FeatureOneShot)
	str += fmt.Sprintf("Removable upload : %s\n", config.FeatureRemovable)
	str += fmt.Sprintf("Streaming upload : %s\n", config.FeatureStream)
	str += fmt.Sprintf("Upload password : %s\n", config.FeaturePassword)
	str += fmt.Sprintf("Upload comments : %s\n", config.FeatureComments)
	str += fmt.Sprintf("Upload set TTL : %s\n", config.FeatureSetTTL)
	str += fmt.Sprintf("Upload extend TTL : %s\n", config.FeatureExtendTTL)

	str += fmt.Sprintf("Authentication : %s\n", config.FeatureAuthentication)
	if config.FeatureAuthentication != FeatureDisabled {
		if config.GoogleAuthentication {
			str += fmt.Sprintf("Google authentication : enabled\n")
		} else {
			str += fmt.Sprintf("Google authentication : disabled\n")
		}

		if config.OvhAuthentication {
			str += fmt.Sprintf("OVH authentication : enabled\n")
			if config.OvhAPIEndpoint != "" {
				str += fmt.Sprintf("OVH API endpoint : %s\n", config.OvhAPIEndpoint)
			}
		} else {
			str += fmt.Sprintf("OVH authentication : disabled\n")
		}
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
