package pgp

import (
	"os"

	"github.com/root-gg/utils"
	"golang.org/x/crypto/openpgp"
)

// Config object
type Config struct {
	Gpg       string
	Keyring   string
	Recipient string
	Email     string
	Entity    *openpgp.Entity
}

// NewPgpBackendConfig instantiate a new Backend Configuration
// from config map passed as argument
func NewPgpBackendConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	config.Gpg = "/usr/bin/gpg"
	config.Keyring = os.Getenv("HOME") + "/.gnupg/pubring.gpg"
	utils.Assign(config, params)
	return
}
