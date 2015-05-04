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
	"github.com/root-gg/utils"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"io"
	"os"
	"strings"
)

type PgpBackendConfig struct {
	Gpg       string
	Keyring   string
	Recipient string
	Email     string
	Entity    *openpgp.Entity
}

func NewPgpBackendConfig(config map[string]interface{}) (this *PgpBackendConfig) {
	this = new(PgpBackendConfig)
	this.Gpg = "/usr/bin/gpg"
	this.Keyring = os.Getenv("HOME") + "/.gnupg/pubring.gpg"
	utils.Assign(this, config)
	return
}

type PgpBackend struct {
	Config *PgpBackendConfig
}

func NewPgpBackend(config map[string]interface{}) (this *PgpBackend) {
	this = new(PgpBackend)
	this.Config = NewPgpBackendConfig(config)
	return
}

func (this *PgpBackend) Configure(arguments map[string]interface{}) (err error) {

	// Parse options
	if arguments["--recipient"] != nil && arguments["--recipient"].(string) != "" {
		this.Config.Recipient = arguments["--recipient"].(string)
	}

	if this.Config.Recipient == "" {
		return errors.New("No PGP recipient specified (--recipient Bob or Recipient param in section [SecureOptions] of .plikrc)")
	}

	// Keyring is here ?
	_, err = os.Stat(this.Config.Keyring)
	if err != nil {
		return errors.New(fmt.Sprintf("GnuPG Keyring %s not found on your system", this.Config.Keyring))
	}

	// Open it
	pubringFile, err := os.Open(this.Config.Keyring)
	if err != nil {
		return errors.New(fmt.Sprintf("Fail to open your GnuPG keyring : %s", err))
	}

	// Read it
	pubring, err := openpgp.ReadKeyRing(pubringFile)
	if err != nil {
		return errors.New(fmt.Sprintf("Fail to read your GnuPG keyring : %s", err))
	}

	// Search for key
	entitiesFound := make(map[uint64]*openpgp.Entity)
	emailsFound := make([]string, 0)
	intToEntity := make(map[int]uint64)
	countEntitiesFound := 0

	for _, entity := range pubring {
		for _, ident := range entity.Identities {
			if strings.Contains(ident.UserId.Email, this.Config.Recipient) || strings.Contains(ident.UserId.Name, this.Config.Recipient) {
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
		return errors.New(fmt.Sprintf("No key found for input '%s' in your keyring !", this.Config.Recipient))
	} else if countEntitiesFound == 1 {
		this.Config.Entity = entitiesFound[intToEntity[0]]
		this.Config.Email = emailsFound[0]
	} else {
		errorMessage := fmt.Sprintf("There are %d keys that match your search :\n", countEntitiesFound)
		for _, email := range emailsFound {
			errorMessage += fmt.Sprintf("\t-%s\n", email)
		}

		return errors.New(errorMessage)
	}

	return nil
}

func (this *PgpBackend) Encrypt(reader io.Reader, writer io.Writer) (err error) {
	w, _ := armor.Encode(writer, "PGP MESSAGE", nil)
	plaintext, _ := openpgp.Encrypt(w, []*openpgp.Entity{this.Config.Entity}, nil, &openpgp.FileHints{IsBinary: true}, nil)

	_, err = io.Copy(plaintext, reader)
	if err != nil {
		return (err)
	}

	plaintext.Close()
	w.Close()

	return
}

func (this *PgpBackend) Comments() string {
	return "gpg -d"
}

func (this *PgpBackend) GetConfiguration() interface{} {
	return this.Config
}
