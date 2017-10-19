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

package pgp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

// Backend object
type Backend struct {
	Config *BackendConfig
}

// NewPgpBackend instantiate a new PGP Crypto Backend
// and configure it from config map
func NewPgpBackend(config map[string]interface{}) (pb *Backend) {
	pb = new(Backend)
	pb.Config = NewPgpBackendConfig(config)
	return
}

// Configure implementation for PGP Crypto Backend
func (pb *Backend) Configure(arguments map[string]interface{}) (err error) {

	// Parse options
	if arguments["--recipient"] != nil && arguments["--recipient"].(string) != "" {
		pb.Config.Recipient = arguments["--recipient"].(string)
	}

	if pb.Config.Recipient == "" {
		return errors.New("No PGP recipient specified (--recipient Bob or Recipient param in section [SecureOptions] of .plikrc)")
	}

	// Keyring is here ?
	_, err = os.Stat(pb.Config.Keyring)
	if err != nil {
		return fmt.Errorf("GnuPG Keyring %s not found on your system", pb.Config.Keyring)
	}

	// Open it
	pubringFile, err := os.Open(pb.Config.Keyring)
	if err != nil {
		return fmt.Errorf("Fail to open your GnuPG keyring : %s", err)
	}

	// Read it
	pubring, err := openpgp.ReadKeyRing(pubringFile)
	if err != nil {
		return fmt.Errorf("Fail to read your GnuPG keyring : %s", err)
	}

	// Search for key
	var emailsFound []string

	entitiesFound := make(map[uint64]*openpgp.Entity)
	intToEntity := make(map[int]uint64)
	countEntitiesFound := 0

	for _, entity := range pubring {
		for _, ident := range entity.Identities {
			if strings.Contains(strings.ToLower(ident.UserId.Email), strings.ToLower(pb.Config.Recipient)) || strings.Contains(strings.ToLower(ident.UserId.Name), strings.ToLower(pb.Config.Recipient)) {
				if _, ok := entitiesFound[entity.PrimaryKey.KeyId]; !ok {
					entitiesFound[entity.PrimaryKey.KeyId] = entity
					intToEntity[countEntitiesFound] = entity.PrimaryKey.KeyId
					emailsFound = append(emailsFound, ident.Name)
					countEntitiesFound++
				}
			}
		}
	}

	// How many entities we have found ?
	if countEntitiesFound == 0 {
		return fmt.Errorf("No key found for input '%s' in your keyring", pb.Config.Recipient)
	} else if countEntitiesFound == 1 {
		pb.Config.Entity = entitiesFound[intToEntity[0]]
		pb.Config.Email = emailsFound[0]
	} else {
		errorMessage := fmt.Sprintf("There are %d keys that match your search :\n", countEntitiesFound)
		for _, email := range emailsFound {
			errorMessage += fmt.Sprintf("\t-%s\n", email)
		}

		return errors.New(errorMessage)
	}

	return nil
}

// Encrypt implementation for PGP Crypto Backend
func (pb *Backend) Encrypt(reader io.Reader, writer io.Writer) (err error) {
	w, err := armor.Encode(writer, "PGP MESSAGE", nil)
	if err != nil {
		return (err)
	}

	plaintext, err := openpgp.Encrypt(w, []*openpgp.Entity{pb.Config.Entity}, nil, &openpgp.FileHints{IsBinary: true}, nil)
	if err != nil {
		return (err)
	}

	_, err = io.Copy(plaintext, reader)
	if err != nil {
		return (err)
	}

	plaintext.Close()
	w.Close()

	return
}

// Comments implementation for PGP Crypto Backend
func (pb *Backend) Comments() string {
	return "gpg -d"
}

// GetConfiguration implementation for PGP Crypto Backend
func (pb *Backend) GetConfiguration() interface{} {
	return pb.Config
}
