/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package common

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/GeertJohan/yubigo"
)

// Configuration object
type Configuration struct {
	LogLevel string `json:"-"`

	ListenAddress string `json:"-"`
	ListenPort    int    `json:"-"`
	Path          string `json:"-"`

	MaxFileSize      int64 `json:"maxFileSize"`
	MaxFilePerUpload int   `json:"maxFilePerUpload"`

	DefaultTTL int `json:"defaultTTL"`
	MaxTTL     int `json:"maxTTL"`

	SslEnabled bool   `json:"-"`
	SslCert    string `json:"-"`
	SslKey     string `json:"-"`

	DownloadDomain string `json:"downloadDomain"`

	YubikeyEnabled   bool   `json:"yubikeyEnabled"`
	YubikeyAPIKey    string `json:"-"`
	YubikeyAPISecret string `json:"-"`

	SourceIPHeader  string   `json:"-"`
	UploadWhitelist []string `json:"-"`

	Authentication       bool     `json:"authentication"`
	NoAnonymousUploads   bool     `json:"-"`
	OneShot              bool     `json:"oneShot"`
	Removable            bool     `json:"removable"`
	ProtectedByPassword  bool     `json:"protectedByPassword"`
	GoogleAuthentication bool     `json:"googleAuthentication"`
	GoogleAPISecret      string   `json:"-"`
	GoogleAPIClientID    string   `json:"-"`
	GoogleValidDomains   []string `json:"-"`
	OvhAuthentication    bool     `json:"ovhAuthentication"`
	OvhAPIEndpoint       string   `json:"ovhApiEndpoint"`
	OvhAPIKey            string   `json:"-"`
	OvhAPISecret         string   `json:"-"`
	Admins               []string `json:"-"`

	MetadataBackend       string                 `json:"-"`
	MetadataBackendConfig map[string]interface{} `json:"-"`

	DataBackend       string                 `json:"-"`
	DataBackendConfig map[string]interface{} `json:"-"`

	StreamMode          bool                   `json:"streamMode"`
	StreamBackendConfig map[string]interface{} `json:"-"`

	clean             bool
	yubiAuth          *yubigo.YubiAuth
	downloadDomainURL *url.URL
	uploadWhitelist   []*net.IPNet
}

// NewConfiguration creates a new configuration
// object with default values
func NewConfiguration() (config *Configuration) {
	config = new(Configuration)
	config.LogLevel = "INFO"
	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8080
	config.DataBackend = "file"
	config.MetadataBackend = "bolt"
	config.MaxFileSize = 10737418240 // 10GB
	config.MaxFilePerUpload = 1000
	config.DefaultTTL = 2592000 // 30 days
	config.MaxTTL = 0
	config.SslEnabled = false
	config.StreamMode = true
	config.OneShot = true
	config.Removable = true
	config.ProtectedByPassword = true
	config.OvhAPIEndpoint = "https://eu.api.ovh.com/1.0"
	config.clean = true
	return
}

// LoadConfiguration creates a new empty configuration
// and try to load specified file with toml library to
// override default params
func LoadConfiguration(file string) (config *Configuration, err error) {
	config = NewConfiguration()

	if _, err := toml.DecodeFile(file, config); err != nil {
		return nil, fmt.Errorf("Unable to load config file %s : %s", file, err)
	}

	err = config.Initialize()
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Initialize config internal parameters
func (config *Configuration) Initialize() (err error) {
	config.Path = strings.TrimSuffix(config.Path, "/")

	// Do user specified a ApiKey and ApiSecret for Yubikey
	if config.YubikeyEnabled {
		yubiAuth, err := yubigo.NewYubiAuth(config.YubikeyAPIKey, config.YubikeyAPISecret)
		if err != nil {
			return fmt.Errorf("Failed to load yubikey backend : %s", err)
		}
		config.yubiAuth = yubiAuth
	}

	// UploadWhitelist is only parsed once at startup time
	for _, cidr := range config.UploadWhitelist {
		if !strings.Contains(cidr, "/") {
			cidr += "/32"
		}
		if _, cidr, err := net.ParseCIDR(cidr); err == nil {
			config.uploadWhitelist = append(config.uploadWhitelist, cidr)
		} else {
			return fmt.Errorf("Failed to parse upload whitelist : %s", cidr)
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

	if !config.GoogleAuthentication && !config.OvhAuthentication {
		config.Authentication = false
		config.NoAnonymousUploads = false
	}

	if config.DownloadDomain != "" {
		strings.Trim(config.DownloadDomain, "/ ")
		var err error
		if config.downloadDomainURL, err = url.Parse(config.DownloadDomain); err != nil {
			return fmt.Errorf("Invalid download domain URL %s : %s", config.DownloadDomain, err)
		}
	}

	return nil
}

// GetUploadWhitelist return the parsed IP upload whitelist
func (config *Configuration) GetUploadWhitelist() []*net.IPNet {
	return config.uploadWhitelist
}

// GetDownloadDomain return the parsed download domain URL
func (config *Configuration) GetDownloadDomain() *url.URL {
	return config.downloadDomainURL
}

// GetYubiAuth return the Yubikey authenticator
func (config *Configuration) GetYubiAuth() *yubigo.YubiAuth {
	return config.yubiAuth
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

// IsUserAdmin check if the user is a Plik server administrator
func (config *Configuration) IsUserAdmin(user *User) bool {
	for _, id := range config.Admins {
		if user.ID == id {
			return true
		}
	}

	return false
}

// GetServerURL is a helper to get the server HTP URL
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
