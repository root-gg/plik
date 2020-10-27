package common

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dustin/go-humanize"

	"github.com/root-gg/logger"
)

// RegistrationOpen user registration enabled
const RegistrationOpen = "open"

// RegistrationClosed user registration disabled
const RegistrationClosed = "closed"

// RegistrationInvite user registration requires an invite
const RegistrationInvite = "invite"

// Configuration object
type Configuration struct {
	Debug         bool   `json:"-"`
	DebugRequests bool   `json:"-"`
	LogLevel      string `json:"-"` // DEPRECATED since 1.3

	ListenAddress  string `json:"-"`
	ListenPort     int    `json:"-"`
	Domain         string `json:"domain"`
	DownloadDomain string `json:"downloadDomain"`
	Path           string `json:"-"`

	MaxFileSize      int64 `json:"maxFileSize"`
	MaxFilePerUpload int   `json:"maxFilePerUpload"`

	DefaultTTL int `json:"defaultTTL"`
	MaxTTL     int `json:"maxTTL"`

	SslEnabled bool   `json:"-"`
	SslCert    string `json:"-"`
	SslKey     string `json:"-"`

	NoWebInterface      bool `json:"-"`
	EnhancedWebSecurity bool `json:"-"`

	SourceIPHeader  string   `json:"-"`
	UploadWhitelist []string `json:"-"`

	OneShot             bool `json:"oneShot"`
	Removable           bool `json:"removable"`
	Stream              bool `json:"stream"`
	ProtectedByPassword bool `json:"protectedByPassword"`

	Authentication       bool     `json:"authentication"`
	NoAnonymousUploads   bool     `json:"noAnonymousUploads"`
	Registration         string   `json:"registration"`
	EmailVerification    bool     `json:"-"`
	EmailValidDomains    []string `json:"-"`
	GoogleAuthentication bool     `json:"googleAuthentication"`
	GoogleAPISecret      string   `json:"-"`
	GoogleAPIClientID    string   `json:"-"`
	GoogleValidDomains   []string `json:"-"` // Deprecated use email valid domains
	OvhAuthentication    bool     `json:"ovhAuthentication"`
	OvhAPIEndpoint       string   `json:"ovhApiEndpoint"`
	OvhAPIKey            string   `json:"-"`
	OvhAPISecret         string   `json:"-"`

	MetadataBackendConfig map[string]interface{} `json:"-"`

	DataBackend       string                 `json:"-"`
	DataBackendConfig map[string]interface{} `json:"-"`

	domainURL         *url.URL
	downloadDomainURL *url.URL
	uploadWhitelist   []*net.IPNet
	clean             bool
}

// NewConfiguration creates a new configuration
// object with default values
func NewConfiguration() (config *Configuration) {
	config = new(Configuration)

	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8080
	config.EnhancedWebSecurity = false

	config.MaxFileSize = 10000000000 // 10GB
	config.MaxFilePerUpload = 1000

	config.DefaultTTL = 2592000 // 30 days
	config.MaxTTL = 2592000     // 30 days

	config.Stream = true
	config.OneShot = true
	config.Removable = true
	config.ProtectedByPassword = true

	config.Registration = RegistrationClosed
	config.EmailVerification = false
	config.OvhAPIEndpoint = "https://eu.api.ovh.com/1.0"

	config.DataBackend = "file"

	config.clean = true
	return
}

// LoadConfiguration creates a new empty configuration
// and try to load specified file with toml library to
// override default params
func LoadConfiguration(file string) (config *Configuration, err error) {
	config = NewConfiguration()

	if _, err := toml.DecodeFile(file, config); err != nil {
		return nil, fmt.Errorf("unable to load config file %s : %s", file, err)
	}

	err = config.Initialize()
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Initialize config internal parameters
func (config *Configuration) Initialize() (err error) {

	// For backward compatibility
	if config.LogLevel == "DEBUG" {
		config.Debug = true
		config.DebugRequests = true
	}

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

	if config.GoogleAPIClientID != "" && config.GoogleAPISecret != "" {
		config.GoogleAuthentication = true
	} else {
		config.GoogleAuthentication = false
	}

	if config.OvhAPIKey != "" && config.OvhAPISecret != "" {
		config.OvhAuthentication = true
	} else {
		config.OvhAuthentication = false
	}

	if !config.Authentication {
		config.NoAnonymousUploads = false
		config.GoogleAuthentication = false
		config.OvhAuthentication = false
		config.EmailVerification = false
		config.Registration = RegistrationClosed
	}

	if config.Domain != "" {
		if config.domainURL, err = url.Parse(config.Domain); err != nil {
			return fmt.Errorf("invalid domain URL %s : %s", config.Domain, err)
		}
	}

	if config.DownloadDomain != "" {
		strings.Trim(config.DownloadDomain, "/ ")
		var err error
		if config.downloadDomainURL, err = url.Parse(config.DownloadDomain); err != nil {
			return fmt.Errorf("invalid download domain URL %s : %s", config.DownloadDomain, err)
		}
	}

	// Todo comment this
	if config.Path != "" {
		config.Path = strings.TrimSuffix(config.Path, "/")
		if config.domainURL != nil {
			config.domainURL.Path = config.Path
		}
		if config.downloadDomainURL != nil {
			config.downloadDomainURL.Path = config.Path

		}
	} else {
		if config.domainURL != nil {
			config.Path = config.domainURL.Path
		}
	}

	if config.Registration != RegistrationOpen && config.Registration != RegistrationInvite {
		config.Registration = RegistrationClosed
	}

	if config.DefaultTTL > config.MaxTTL {
		return fmt.Errorf("DefaultTTL should not be more than MaxTTL")
	}

	return nil
}

// NewLogger returns a new logger instance
func (config *Configuration) NewLogger() (log *logger.Logger) {
	level := "INFO"
	if config.Debug {
		level = "DEBUG"
	}
	return logger.NewLogger().SetMinLevelFromString(level).SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel)
}

// GetUploadWhitelist return the parsed IP upload whitelist
func (config *Configuration) GetUploadWhitelist() []*net.IPNet {
	return config.uploadWhitelist
}

// HasDownloadURL return true if a download domain url has been set
func (config *Configuration) HasDownloadURL() bool {
	return config.downloadDomainURL != nil
}

// GetDownloadURL return the parsed download domain URL
func (config *Configuration) GetDownloadURL() *url.URL {
	if config.downloadDomainURL != nil {
		return config.downloadDomainURL
	}
	return config.GetServerURL()
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
	if config.domainURL != nil {
		return config.domainURL
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

func (config *Configuration) String() string {
	str := ""

	if config.Domain != "" {
		str += fmt.Sprintf("Domain : %s\n", config.DownloadDomain)
	}
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

	if config.OneShot {
		str += fmt.Sprintf("One shot upload : enabled\n")
	} else {
		str += fmt.Sprintf("One shot upload : disabled\n")
	}

	if config.Removable {
		str += fmt.Sprintf("Removable upload : enabled\n")
	} else {
		str += fmt.Sprintf("Removable upload : disabled\n")
	}

	if config.Stream {
		str += fmt.Sprintf("Streaming upload : enabled\n")
	} else {
		str += fmt.Sprintf("Streaming upload : disabled\n")
	}

	if config.ProtectedByPassword {
		str += fmt.Sprintf("Upload password : enabled\n")
	} else {
		str += fmt.Sprintf("Upload password : disabled\n")
	}

	if config.Authentication {
		str += fmt.Sprintf("Authentication : enabled\n")
		str += fmt.Sprintf("User registration : %s\n", config.Registration)

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
	} else {
		str += fmt.Sprintf("Authentication : disabled\n")
	}

	return str
}
