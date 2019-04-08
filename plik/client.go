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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
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

	// This breaks go get so ignore until we find a better way to do
	//bi := common.GetBuildInfo()
	//if bi != nil {
	c.ClientVersion = runtime.GOOS + "-" + runtime.GOARCH //+ "-" + bi.Version
	//}

	// Create a new default HTTP client. Override it if may you have more specific requirements
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	c.HTTPClient = &http.Client{Transport: transport}

	return c
}

// Create creates a new empty upload on the Plik Server and return the upload metadata
func (c *Client) Create(uploadParams *common.Upload) (uploadInfo *common.Upload, err error) {
	var URL *url.URL
	URL, err = url.Parse(c.URL + "/upload")
	if err != nil {
		return nil, err
	}

	var j []byte
	j, err = json.Marshal(uploadParams)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if uploadParams.Token != "" {
		req.Header.Set("X-PlikToken", uploadParams.Token)
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
	uploadInfo = &common.Upload{}
	err = json.Unmarshal(body, uploadInfo)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		fmt.Printf("Upload created : %s\n", utils.Sdump(uploadInfo))
	}

	return uploadInfo, nil
}

// UploadFile uploads a data stream to the Plik Server and return the file metadata
func (c *Client) UploadFile(upload *common.Upload, fileParams *common.File, reader io.Reader) (fileInfo *common.File, err error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	errCh := make(chan error)
	go func(errCh chan error) {
		writer, err := multipartWriter.CreateFormFile("file", fileParams.Name)
		if err != nil {
			err = fmt.Errorf("Unable to create multipartWriter : %s", err)
			pipeWriter.CloseWithError(err)
			errCh <- err
			return
		}

		_, err = io.Copy(writer, reader)
		if err != nil {
			pipeWriter.CloseWithError(err)
			errCh <- err
			return
		}

		err = multipartWriter.Close()
		if err != nil {
			err = fmt.Errorf("Unable to close multipartWriter : %s", err)
			errCh <- err
			return
		}

		pipeWriter.CloseWithError(err)
		errCh <- err
	}(errCh)

	mode := "file"
	if upload.Stream {
		mode = "stream"
	}

	var URL *url.URL
	if fileParams.ID != "" {
		URL, err = url.Parse(c.URL + "/" + mode + "/" + upload.ID + "/" + fileParams.ID + "/" + fileParams.Name)
	} else {
		// Old method without file id that can also be used to add files to an existing upload
		if upload.Stream {
			return nil, fmt.Errorf("Files must be added to upload before creation for stream mode to work")
		}
		URL, err = url.Parse(c.URL + "/" + mode + "/" + upload.ID)
	}

	if err != nil {
		return nil, err
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), pipeReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("X-UploadToken", upload.UploadToken)

	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	err = <-errCh
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse json response
	fileInfo = &common.File{}
	err = json.Unmarshal(body, fileInfo)
	if err != nil {
		return nil, err
	}

	if c.Debug {
		fmt.Printf("File uploaded : %s\n", utils.Sdump(fileInfo))
	}

	return fileInfo, nil
}

// MakeRequest perform an HTTP request to a Plik Server HTTP API.
//  - Manage request header X-ClientApp and X-ClientVersion
//  - Log the request and response if the client is in Debug mode
//  - Parsing response error to Go error
func (c *Client) MakeRequest(req *http.Request) (resp *http.Response, err error) {

	// Set client version headers
	if c.ClientName != "" {
		req.Header.Set("X-ClientApp", c.ClientName)
	}
	if c.ClientVersion != "" {
		req.Header.Set("X-ClientVersion", c.ClientVersion)
	}

	// Log request
	if c.Debug {
		dumpBody := true
		if strings.Contains(req.URL.String(), "/file/") || strings.Contains(req.URL.String(), "/stream") {
			dumpBody = false
		}
		dump, err := httputil.DumpRequest(req, dumpBody)
		if err == nil {
			fmt.Println(string(dump))
		} else {
			return nil, fmt.Errorf("Unable to dump HTTP request : %s", err)
		}
	}

	// Make request
	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return
	}

	// Log response
	if c.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			fmt.Println(string(dump))
		} else {
			return nil, fmt.Errorf("Unable to dump HTTP response : %s", err)
		}
	}

	// Parse json error
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}

		result := new(common.Result)
		err = json.Unmarshal(body, result)
		if err == nil && result.Message != "" {
			err = fmt.Errorf("%s : %s", resp.Status, result.Message)
		} else if len(body) > 0 {
			err = fmt.Errorf("%s : %s", resp.Status, string(body))
		} else {
			err = fmt.Errorf("%s", resp.Status)
		}
		return resp, err
	}

	return resp, nil
}

// NewUpload create a new Upload object with the client default upload params
func (c *Client) NewUpload() *Upload {
	return newUpload(c)
}

// UploadFiles is a handy wrapper to upload one several filesystem files
func (c *Client) UploadFiles(paths ...string) (upload *Upload, err error) {
	upload = c.NewUpload()

	// Create files
	for _, path := range paths {
		file, err := NewFileFromPath(path)
		if err != nil {
			return nil, err
		}
		upload.AddFiles(file)
	}

	// Create upload and upload the files
	err = upload.Upload()
	if err != nil {
		return nil, err
	}

	return upload, nil
}

// UploadReader is a handy wrapper to upload a single arbitrary data stream
func (c *Client) UploadReader(name string, reader io.Reader) (upload *Upload, err error) {
	upload = c.NewUpload()

	// Create a new file from the io.Reader
	upload.AddFiles(NewFileFromReader(name, reader))

	// Create upload and upload the file
	err = upload.Upload()
	if err != nil {
		return nil, err
	}

	return upload, nil
}
