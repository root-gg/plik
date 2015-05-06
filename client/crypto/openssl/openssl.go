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
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"io"
	"os"
	"os/exec"
)

type OpenSSLBackendConfig struct {
	Openssl    string
	Cipher     string
	Passphrase string
	Options    string
}

func NewOpenSSLBackendConfig(config map[string]interface{}) (this *OpenSSLBackendConfig) {
	this = new(OpenSSLBackendConfig)
	this.Openssl = "/usr/bin/openssl"
	this.Cipher = "aes256"
	utils.Assign(this, config)
	return
}

type OpenSSLBackend struct {
	Config *OpenSSLBackendConfig
}

func NewOpenSSLBackend(config map[string]interface{}) (this *OpenSSLBackend) {
	this = new(OpenSSLBackend)
	this.Config = NewOpenSSLBackendConfig(config)
	return
}

func (this *OpenSSLBackend) Configure(arguments map[string]interface{}) (err error) {
	if arguments["--openssl"] != nil && arguments["--openssl"].(string) != "" {
		this.Config.Openssl = arguments["--openssl"].(string)
	}
	if arguments["--cipher"] != nil && arguments["--cipher"].(string) != "" {
		this.Config.Cipher = arguments["--cipher"].(string)
	}
	if arguments["--passphrase"] != nil && arguments["--passphrase"].(string) != "" {
		this.Config.Passphrase = arguments["--passphrase"].(string)
		if this.Config.Passphrase == "-" {
			fmt.Printf("Please enter a passphrase : ")
			_, err = fmt.Scanln(&this.Config.Passphrase)
			if err != nil {
				return err
			}
		}
	} else {
		this.Config.Passphrase = common.GenerateRandomId(25)
		fmt.Println("Passphrase : " + this.Config.Passphrase)
	}
	if arguments["--secure-options"] != nil && arguments["--secure-options"].(string) != "" {
		this.Config.Options = arguments["--secure-options"].(string)
	}

	return
}

func (this *OpenSSLBackend) Encrypt(reader io.Reader, writer io.Writer) (err error) {
	passReader, passWriter, err := os.Pipe()
	if err != nil {
		fmt.Printf("Unable to make pipe : %s\n", err)
		os.Exit(1)
		return
	}
	_, err = passWriter.Write([]byte(this.Config.Passphrase))
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
	cmd := exec.Command(this.Config.Openssl, this.Config.Cipher, "-pass", fmt.Sprintf("fd:3"))
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

func (this *OpenSSLBackend) Comments() string {
	return fmt.Sprintf("openssl %s -d -pass pass:%s", this.Config.Cipher, this.Config.Passphrase)
}

func (this *OpenSSLBackend) GetConfiguration() interface{} {
	return this.Config
}
