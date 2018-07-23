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
	Config *BackendConfig
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
func (ob *Backend) Encrypt(reader io.Reader, writer io.Writer) (err error) {
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

	var args []string
	args = append(args, ob.Config.Cipher)
	args = append(args, "-pass", fmt.Sprintf("fd:3"))
	args = append(args, strings.Fields(ob.Config.Options)...)

	cmd := exec.Command(ob.Config.Openssl, args...)
	cmd.Stdin = reader                                  // fd:0
	cmd.Stdout = writer                                 // fd:1
	cmd.Stderr = os.Stderr                              // fd:2
	cmd.ExtraFiles = append(cmd.ExtraFiles, passReader) // fd:3
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Unable to run openssl cmd : %s\n", err)
		os.Exit(1)
		return
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Printf("Unable to run openssl cmd : %s\n", err)
		os.Exit(1)
		return
	}
	return
}

// Comments implementation for OpenSSL Crypto Backend
func (ob *Backend) Comments() string {
	return fmt.Sprintf("openssl %s -d -pass pass:%s %s", ob.Config.Cipher, ob.Config.Passphrase, ob.Config.Options)
}

// GetConfiguration implementation for OpenSSL Crypto Backend
func (ob *Backend) GetConfiguration() interface{} {
	return ob.Config
}
