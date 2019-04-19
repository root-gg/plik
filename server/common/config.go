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

	DownloadDomain    string   `json:"downloadDomain"`
	DownloadDomainURL *url.URL `json:"-"`

	YubikeyEnabled   bool             `json:"yubikeyEnabled"`
	YubikeyAPIKey    string           `json:"-"`
	YubikeyAPISecret string           `json:"-"`
	YubiAuth         *yubigo.YubiAuth `json:"-"`

	SourceIPHeader  string   `json:"-"`
	UploadWhitelist []string `json:"-"`

	Authentication       bool     `json:"authentication"`
	NoAnonymousUploads   bool     `json:"-"`
	OneShot              bool     `json:"oneShot"`
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

	uploadWhitelist []*net.IPNet
}

// NewConfiguration creates a new configuration
// object with default values
func NewConfiguration() (config *Configuration) {
	config = new(Configuration)
	config.LogLevel = "INFO"
	config.ListenAddress = "0.0.0.0"
	config.ListenPort = 8080
	config.DataBackend = "file"
	config.MetadataBackend = "file"
	config.MaxFileSize = 10737418240 // 10GB
	config.MaxFilePerUpload = 1000
	config.DefaultTTL = 2592000 // 30 days
	config.MaxTTL = 0
	config.SslEnabled = false
	config.StreamMode = true
	config.OneShot = true
	config.ProtectedByPassword = true
	config.OvhAPIEndpoint = "https://eu.api.ovh.com/1.0"
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

	config.Path = strings.TrimSuffix(config.Path, "/")

	// Do user specified a ApiKey and ApiSecret for Yubikey
	if config.YubikeyEnabled {
		yubiAuth, err := yubigo.NewYubiAuth(config.YubikeyAPIKey, config.YubikeyAPISecret)
		if err != nil {
			return nil, fmt.Errorf("Failed to load yubikey backend : %s", err)
		}
		config.YubiAuth = yubiAuth
	}

	// UploadWhitelist is only parsed once at startup time
	for _, cidr := range config.UploadWhitelist {
		if !strings.Contains(cidr, "/") {
			cidr += "/32"
		}
		if _, cidr, err := net.ParseCIDR(cidr); err == nil {
			config.uploadWhitelist = append(config.uploadWhitelist, cidr)
		} else {
			return nil, fmt.Errorf("Failed to parse upload whitelist : %s", cidr)
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

	if config.MetadataBackend == "file" {
		config.Authentication = false
		config.NoAnonymousUploads = false
	}

	if config.DownloadDomain != "" {
		strings.Trim(config.DownloadDomain, "/ ")
		var err error
		if config.DownloadDomainURL, err = url.Parse(config.DownloadDomain); err != nil {
			return nil, fmt.Errorf("Invalid download domain URL %s : %s", config.DownloadDomain, err)
		}
	}

	return config, nil
}

// GetUploadWhitelist return the IP upload whitelist
func (config *Configuration) GetUploadWhitelist() []*net.IPNet {
	return config.uploadWhitelist
}

// IsAdmin check if the user is a Plik server administrator
func (config *Configuration) IsAdmin(user *User) bool {
	if user.Admin == true {
		return true
	}

	// Check if user is admin
	for _, id := range config.Admins {
		if user.ID == id {
			user.Admin = true
			return true
		}
	}

	return false
}
