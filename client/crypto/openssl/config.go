package openssl

import (
	"github.com/root-gg/utils"
)

// Config object
type Config struct {
	Openssl    string
	Cipher     string
	Passphrase string
	Options    string
}

// NewOpenSSLBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewOpenSSLBackendConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	config.Openssl = "/usr/bin/openssl"
	config.Cipher = "aes-256-cbc"
	config.Options = "-md sha512 -pbkdf2 -iter 120000"
	utils.Assign(config, params)
	return
}
