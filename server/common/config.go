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
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/BurntSushi/toml"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/GeertJohan/yubigo"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/logger"
)

var (
	plikVersion = "##VERSION##"
)

// Configuration object
type Configuration struct {
	LogLevel      string
	ListenAddress string
	ListenPort    int
	MaxFileSize   int

	DefaultTTL int
	MaxTTL     int

	SslEnabled bool
	SslCert    string
	SslKey     string

	YubikeyEnabled   bool
	YubikeyAPIKey    string
	YubikeyAPISecret string
	YubiAuth         *yubigo.YubiAuth

	MetadataBackend       string
	MetadataBackendConfig map[string]interface{}

	DataBackend       string
	DataBackendConfig map[string]interface{}

	StreamMode          bool
	StreamBackendConfig map[string]interface{}

	ShortenBackend       string
	ShortenBackendConfig map[string]interface{}
}

// Global var to store conf
var Config *Configuration

// NewConfiguration creates a new configuration
// object with default values
func NewConfiguration() (this *Configuration) {
	this = new(Configuration)
	this.LogLevel = "INFO"
	this.ListenAddress = "0.0.0.0"
	this.ListenPort = 8080
	this.DataBackend = "file"
	this.MetadataBackend = "file"
	this.MaxFileSize = 1048576 // 1MB
	this.DefaultTTL = 2592000  // 30 days
	this.MaxTTL = 0
	this.SslEnabled = false
	this.SslCert = ""
	this.SslKey = ""
	this.StreamMode = true
	return
}

// LoadConfiguration creates a new empty configuration
// and try to load specified file with toml library to
// override default params
func LoadConfiguration(file string) {
	Config = NewConfiguration()
	if _, err := toml.DecodeFile(file, Config); err != nil {
		Log().Warningf("Unable to load config file %s : %s", file, err)
	}
	Log().SetMinLevelFromString(Config.LogLevel)
	Log().Dump(logger.DEBUG, Config)

	if Config.LogLevel == "DEBUG" {
		Log().SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel | logger.FshortFile | logger.FshortFunction)
	} else {
		Log().SetFlags(logger.Fdate | logger.Flevel | logger.FfixedSizeLevel)
	}

	// Do user specified a ApiKey and ApiSecret for Yubikey
	if Config.YubikeyEnabled {
		yubiAuth, err := yubigo.NewYubiAuth(Config.YubikeyAPIKey, Config.YubikeyAPISecret)
		if err != nil {
			Log().Warningf("Failed to load yubikey backend : %s", err)
			Config.YubikeyEnabled = false
		} else {
			Config.YubiAuth = yubiAuth
		}
	}
}

// GetVersion return the hardcoded version
// before compilation
func GetVersion() string {
	return plikVersion
}
