package pgp

import (
	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"
	"errors"
	"fmt"
	"github.com/root-gg/plik/client/config"
	"github.com/root-gg/plik/server/utils"
	"io"
	"os"
	"strings"
)

type PgpBackendConfig struct {
	Gpg       string
	Keyring   string
	Recipient string
	Entity    *openpgp.Entity
}

func NewPgpBackendConfig(configuration map[string]interface{}) (this *PgpBackendConfig) {
	this = new(PgpBackendConfig)
	this.Gpg = "/usr/bin/gpg"
	this.Keyring = config.Config.HomeDir + "/.gnupg/pubring.gpg"
	utils.Assign(this, configuration)
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

	if arguments["--recipient"] != nil && arguments["--recipient"].(string) != "" {
		this.Config.Recipient = arguments["--recipient"].(string)
	} else {
		return errors.New("No PGP recipient specified (--recipient Bob)")
	}

	config.Debug("PGP configuration : " + config.Sdump(this.Config))

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
		if !config.Config.Quiet {
			fmt.Printf("Encrypted for pgp recipent : %s\n", emailsFound[0])
		}
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
