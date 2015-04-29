package common

import (
	"github.com/BurntSushi/toml"
	"github.com/root-gg/logger"
)

var PlikVersion = "##VERSION##"

type Configuration struct {
	LogLevel      string
	ListenAddress string
	ListenPort    int
	MaxFileSize   int

	DefaultTtl int
	MaxTtl     int

	SslCert string
	SslKey  string

	UploadIpRestriction bool
	UploadIpSubnets     []string
	UploadIpGetMethod   string

	UploadIdLength int
	FileIdLength   int

	MetadataBackend       string
	MetadataBackendConfig map[string]interface{}

	DataBackend       string
	DataBackendConfig map[string]interface{}

	ShortenBackend       string
	ShortenBackendConfig map[string]interface{}
}

// Global var to store conf
var Config *Configuration

func NewConfiguration() (this *Configuration) {
	this = new(Configuration)
	this.LogLevel = "INFO"
	this.ListenAddress = "0.0.0.0"
	this.ListenPort = 8080
	this.UploadIpRestriction = false
	this.UploadIpGetMethod = "request"
	this.MetadataBackend = "file"
	this.MaxFileSize = 1048576 // 1MB
	this.DefaultTtl = 2592000  // 30 days
	this.MaxTtl = 0
	this.SslCert = ""
	this.SslKey = ""
	return
}

func LoadConfiguration(file string) {
	Config = NewConfiguration()
	if _, err := toml.DecodeFile(file, Config); err != nil {
		Log().Warningf("Unable to load config file %s : %s", file, err)
	}
	Log().SetMinLevelFromString(Config.LogLevel)
	Log().Dump(logger.DEBUG, Config)
}
