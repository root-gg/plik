package crypto

import (
	"errors"
	"github.com/root-gg/plik/client/crypto/openssl"
	"io"
)

type CryptoBackend interface {
	Configure(arguments map[string]interface{}) (err error)
	Encrypt(reader io.Reader, writer io.Writer) (err error)
	Comments() string
}

func NewCryptoBackend(name string, config map[string]interface{}) (backend CryptoBackend, err error) {
	switch name {
	case "openssl":
		backend = openssl.NewOpenSSLBackend(config)
	default:
		err = errors.New("Invalid crypto backend")
	}
	return
}
