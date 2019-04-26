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

package plik

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/version"
)

// Client manage the process of communicating with a Plik server via the HTTP API
type Client struct {
	*UploadParams // Default upload params for the Client. Those can be overridden per upload

	Debug bool // Display HTTP request and response and other helpful debug data

	URL           string // URL of the Plik server
	ClientName    string // X-ClientApp HTTP Header setting
	ClientVersion string // X-ClientVersion HTTP Header setting

	HTTPClient *http.Client // HTTP Client ot use to make the requests
}

// NewClient creates a new Plik Client
func NewClient(url string) (c *Client) {
	c = &Client{}

	// Default upload params
	c.UploadParams = &UploadParams{}
	c.URL = url

	// Default values for X-ClientApp and X-ClientVersion HTTP Headers
	c.ClientName = "plik_client"
	c.ClientVersion = runtime.GOOS + "-" + runtime.GOARCH + "-" + version.Get()

	// Create a new default HTTP client. Override it if may you have more specific requirements
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	c.HTTPClient = &http.Client{Transport: transport}

	return c
}

// NewUpload create a new Upload object with the client default upload params
func (c *Client) NewUpload() *Upload {
	return newUpload(c)
}

// UploadFile is a handy wrapper to upload a file from the filesystem
func (c *Client) UploadFile(path string) (upload *Upload, file *File, err error) {
	upload = c.NewUpload()

	file, err = upload.AddFileFromPath(path)
	if err != nil {
		return nil, nil, err
	}

	// Create upload and upload the file
	err = upload.Upload()
	if err != nil {
		// Return the upload and file to get a chance to get the file error
		return upload, file, err
	}

	return upload, file, nil
}

// UploadReader is a handy wrapper to upload a single arbitrary data stream
func (c *Client) UploadReader(name string, reader io.Reader) (upload *Upload, file *File, err error) {
	upload = c.NewUpload()

	file = upload.AddFileFromReader(name, reader)

	// Create upload and upload the file
	err = upload.Upload()
	if err != nil {
		// Return the upload and file to get a chance to get the file error
		return upload, file, err
	}

	return upload, file, nil
}

// GetServerVersion return the remote server version
func (c *Client) GetServerVersion() (bi *common.BuildInfo, err error) {
	var req *http.Request
	req, err = http.NewRequest("GET", c.URL+"/version", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse json response
	bi = &common.BuildInfo{}
	err = json.Unmarshal(body, bi)
	if err != nil {
		return nil, err
	}

	return bi, nil
}

// GetUpload return the remote upload info for the given upload id
func (c *Client) GetUpload(id string) (upload *Upload, err error) {
	uploadParams := &common.Upload{}
	uploadParams.ID = id
	uploadParams.Token = c.Token
	uploadParams.Login = c.Login
	uploadParams.Password = c.Password
	return c.getUploadWithParams(uploadParams)
}
