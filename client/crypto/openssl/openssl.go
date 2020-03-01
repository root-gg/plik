package openssl

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	Config *Config
}

// NewOpenSSLBackend instantiate a new PGP Crypto Backend
// and configure it from config map
func NewOpenSSLBackend(config map[string]interface{}) (ob *Backend) {
	ob = new(Backend)
	ob.Config = NewOpenSSLBackendConfig(config)
	return
}

// Configure implementation for OpenSSL Crypto Backend
func (ob *Backend) Configure(arguments map[string]interface{}) (err error) {
	if arguments["--openssl"] != nil && arguments["--openssl"].(string) != "" {
		ob.Config.Openssl = arguments["--openssl"].(string)
	}
	if arguments["--cipher"] != nil && arguments["--cipher"].(string) != "" {
		ob.Config.Cipher = arguments["--cipher"].(string)
	}
	if arguments["--passphrase"] != nil && arguments["--passphrase"].(string) != "" {
		ob.Config.Passphrase = arguments["--passphrase"].(string)
		if ob.Config.Passphrase == "-" {
			fmt.Printf("Please enter a passphrase : ")
			_, err = fmt.Scanln(&ob.Config.Passphrase)
			if err != nil {
				return err
			}
		}
	} else {
		ob.Config.Passphrase = common.GenerateRandomID(25)
		fmt.Println("Passphrase : " + ob.Config.Passphrase)
	}
	if arguments["--secure-options"] != nil && arguments["--secure-options"].(string) != "" {
		ob.Config.Options = arguments["--secure-options"].(string)
	}

	return
}

// Encrypt implementation for OpenSSL Crypto Backend
func (ob *Backend) Encrypt(in io.Reader) (out io.Reader, err error) {
	passReader, passWriter, err := os.Pipe()
	if err != nil {
		fmt.Printf("Unable to make pipe : %s\n", err)
		os.Exit(1)
		return
	}
	_, err = passWriter.Write([]byte(ob.Config.Passphrase))
	if err != nil {
		fmt.Printf("Unable to write to pipe : %s\n", err)
		os.Exit(1)
		return
	}
	err = passWriter.Close()
	if err != nil {
		fmt.Printf("Unable to close to pipe : %s\n", err)
		os.Exit(1)
		return
	}

	out, writer := io.Pipe()

	var args []string
	args = append(args, ob.Config.Cipher)
	args = append(args, "-pass", fmt.Sprintf("fd:3"))
	args = append(args, strings.Fields(ob.Config.Options)...)

	go func() {
		cmd := exec.Command(ob.Config.Openssl, args...)
		cmd.Stdin = in                                      // fd:0
		cmd.Stdout = writer                                 // fd:1
		cmd.Stderr = os.Stderr                              // fd:2
		cmd.ExtraFiles = append(cmd.ExtraFiles, passReader) // fd:3
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to run openssl cmd : %s\n", err)
			writer.CloseWithError(err)
			return
		}
		err = cmd.Wait()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to run openssl cmd : %s\n", err)
			writer.CloseWithError(err)
			return
		}

		writer.Close()
	}()

	return out, nil
}

// Comments implementation for OpenSSL Crypto Backend
func (ob *Backend) Comments() string {
	return fmt.Sprintf("openssl %s -d -pass pass:%s %s", ob.Config.Cipher, ob.Config.Passphrase, ob.Config.Options)
}

// GetConfiguration implementation for OpenSSL Crypto Backend
func (ob *Backend) GetConfiguration() interface{} {
	return ob.Config
}
