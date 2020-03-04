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
	Config *Config
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
func (pb *Backend) Encrypt(in io.Reader) (out io.Reader, err error) {
	out, writer := io.Pipe()

	go func() {
		w, err := armor.Encode(writer, "PGP MESSAGE", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to armor encode pgp : %s\n", err)
			writer.CloseWithError(err)
			return
		}

		plaintext, err := openpgp.Encrypt(w, []*openpgp.Entity{pb.Config.Entity}, nil, &openpgp.FileHints{IsBinary: true}, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to encrypt pgp : %s\n", err)
			writer.CloseWithError(err)
			return
		}

		_, err = io.Copy(plaintext, in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to pipe pgp : %s\n", err)
			writer.CloseWithError(err)
			return
		}

		plaintext.Close()
		w.Close()
		writer.Close()
	}()

	return out, nil
}

// Comments implementation for PGP Crypto Backend
func (pb *Backend) Comments() string {
	return "gpg -d"
}

// GetConfiguration implementation for PGP Crypto Backend
func (pb *Backend) GetConfiguration() interface{} {
	return pb.Config
}
