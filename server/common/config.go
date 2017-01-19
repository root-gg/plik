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
	"net"
	"net/url"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/GeertJohan/yubigo"
	"github.com/root-gg/logger"
)

// Configuration object
type Configuration struct {
	LogLevel         string `json:"-"`
	ListenAddress    string `json:"-"`
	ListenPort       int    `json:"-"`
	MaxFileSize      int64  `json:"maxFileSize"`
	MaxFilePerUpload int    `json:"maxFilePerUpload"`

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

	Authentication       bool   `json:"authentication"`
	GoogleAuthentication bool   `json:"googleAuthentication"`
	GoogleAPISecret      string `json:"-"`
	GoogleAPIClientID    string `json:"-"`
	OvhAuthentication    bool   `json:"ovhAuthentication"`
	OvhAPIKey            string `json:"-"`
	OvhAPISecret         string `json:"-"`

	MetadataBackend       string                 `json:"-"`
	MetadataBackendConfig map[string]interface{} `json:"-"`

	DataBackend       string                 `json:"-"`
	DataBackendConfig map[string]interface{} `json:"-"`

	StreamMode          bool                   `json:"streamMode"`
	StreamBackendConfig map[string]interface{} `json:"-"`
}

// Config static variable
var Config *Configuration

// UploadWhitelist is only parsed once at startup time
var UploadWhitelist []*net.IPNet

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
	config.SslCert = ""
	config.SslKey = ""
	config.StreamMode = true
	return
}

// LoadConfiguration creates a new empty configuration
// and try to load specified file with toml library to
// override default params
func LoadConfiguration(file string) {
	Config = NewConfiguration()
	if _, err := toml.DecodeFile(file, Config); err != nil {
		Logger().Fatalf("Unable to load config file %s : %s", file, err)
	}
	Logger().SetMinLevelFromString(Config.LogLevel)

	if Config.LogLevel == "DEBUG" {
		Logger().SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel | logger.FshortFile | logger.FshortFunction)
	} else {
		Logger().SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel)
	}

	// Do user specified a ApiKey and ApiSecret for Yubikey
	if Config.YubikeyEnabled {
		yubiAuth, err := yubigo.NewYubiAuth(Config.YubikeyAPIKey, Config.YubikeyAPISecret)
		if err != nil {
			Logger().Warningf("Failed to load yubikey backend : %s", err)
			Config.YubikeyEnabled = false
		} else {
			Config.YubiAuth = yubiAuth
		}
	}

	// Parse upload whitelist
	UploadWhitelist = make([]*net.IPNet, 0)
	if Config.UploadWhitelist != nil {
		for _, cidr := range Config.UploadWhitelist {
			if !strings.Contains(cidr, "/") {
				cidr += "/32"
			}
			if _, net, err := net.ParseCIDR(cidr); err == nil {
				UploadWhitelist = append(UploadWhitelist, net)
			} else {
				Logger().Fatalf("Failed to parse upload whitelist : %s", cidr)
			}
		}
	}

	if Config.GoogleAPIClientID != "" && Config.GoogleAPISecret != "" {
		Config.GoogleAuthentication = true
	} else {
		Config.GoogleAuthentication = false
	}

	if Config.OvhAPIKey != "" && Config.OvhAPISecret != "" {
		Config.OvhAuthentication = true
	} else {
		Config.OvhAuthentication = false
	}

	if !Config.GoogleAuthentication && !Config.OvhAuthentication {
		Config.Authentication = false
	}

	if Config.MetadataBackend == "file" {
		Config.Authentication = false
	}

	if Config.DownloadDomain != "" {
		strings.Trim(Config.DownloadDomain, "/ ")
		var err error
		if Config.DownloadDomainURL, err = url.Parse(Config.DownloadDomain); err != nil {
			Logger().Fatalf("Invalid download domain URL %s : %s", Config.DownloadDomain, err)
		}
	}

	Logger().Dump(logger.DEBUG, Config)
}
