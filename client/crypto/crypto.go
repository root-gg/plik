/**

    Plik upload client

The MIT License (MIT)

Copyright (c) <2015>
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
	Encrypt(reader io.Reader, writer io.Writer) (err error)
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
