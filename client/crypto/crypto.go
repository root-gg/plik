package crypto

import (
	"errors"
	"io"

	"github.com/root-gg/plik/client/crypto/openssl"
	"github.com/root-gg/plik/client/crypto/pgp"
)

// Backend interface describe methods that the different
// types of crypto backend must implement to work.
type Backend interface {
	Configure(arguments map[string]interface{}) (err error)
	Encrypt(in io.Reader) (out io.Reader, err error)
	Comments() string
	GetConfiguration() interface{}
}

// NewCryptoBackend instantiate the wanted archive backend with the name provided in configuration file
// We are passing its configuration found in .plikrc file or arguments
func NewCryptoBackend(name string, config map[string]interface{}) (backend Backend, err error) {
	switch name {
	case "openssl":
		backend = openssl.NewOpenSSLBackend(config)
	case "pgp":
		backend = pgp.NewPgpBackend(config)
	default:
		err = errors.New("Invalid crypto backend")
	}
	return
}
