/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015> Copyright holders list can be found in AUTHORS file
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
)

// Create creates a new empty upload on the Plik Server and return the upload metadata
func (c *Client) create(uploadParams *common.Upload) (uploadInfo *common.Upload, err error) {
	if uploadParams == nil {
		return nil, errors.New("missing upload params")
	}

	var j []byte
	j, err = json.Marshal(uploadParams)
	if err != nil {
		return nil, err
	}

	req, err := c.UploadRequest(uploadParams, "POST", c.URL+"/upload", bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

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
func (c *Client) uploadFile(upload *common.Upload, fileParams *common.File, reader io.Reader) (fileInfo *common.File, err error) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	if upload == nil || fileParams == nil || reader == nil {
		return nil, errors.New("missing file upload parameter")
	}

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

	req, err := c.UploadRequest(upload, "POST", URL.String(), pipeReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

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

// UploadRequest creates a new HTTP request with the header generated from the given upload params
func (c *Client) UploadRequest(upload *common.Upload, method, URL string, body io.Reader) (req *http.Request, err error) {

	req, err = http.NewRequest(method, URL, body)
	if err != nil {
		return nil, err
	}

	// TODO Authorization

	if upload.Token != "" {
		req.Header.Set("X-PlikToken", upload.Token)
	}

	if upload.UploadToken != "" {
		req.Header.Set("X-UploadToken", upload.UploadToken)
	}

	return req, nil
}

// getUploadWithParams return the remote upload info for the given upload params
func (c *Client) getUploadWithParams(uploadParams *common.Upload) (upload *Upload, err error) {
	URL := c.URL + "/upload/" + uploadParams.ID

	req, err := c.UploadRequest(uploadParams, "GET", URL, nil)
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
	params := &common.Upload{}
	err = json.Unmarshal(body, params)
	if err != nil {
		return nil, err
	}

	upload = newUploadFromParams(c, params)

	return upload, nil
}

// downloadFile download the remote file from the server
func (c *Client) downloadFile(uploadParams *common.Upload, fileParams *common.File) (reader io.ReadCloser, err error) {
	URL := c.URL + "/file/" + uploadParams.ID + "/" + fileParams.ID + "/" + fileParams.Name

	req, err := c.UploadRequest(uploadParams, "GET", URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// downloadArchive download the remote upload files as a zip archive from the server
func (c *Client) downloadArchive(uploadParams *common.Upload) (reader io.ReadCloser, err error) {
	URL := c.URL + "/archive/" + uploadParams.ID + "/archive.zip"

	req, err := c.UploadRequest(uploadParams, "GET", URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// removeFile remove the remote file from the server
func (c *Client) removeFile(uploadParams *common.Upload, fileParams *common.File) (err error) {
	URL := c.URL + "/file/" + uploadParams.ID + "/" + fileParams.ID + "/" + fileParams.Name

	req, err := c.UploadRequest(uploadParams, "DELETE", URL, nil)
	if err != nil {
		return err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return err
	}

	_ = resp.Body.Close()

	return nil
}

// removeUpload remove the remote upload and all the associated files from the server
func (c *Client) removeUpload(uploadParams *common.Upload) (err error) {
	URL := c.URL + "/upload/" + uploadParams.ID

	req, err := c.UploadRequest(uploadParams, "DELETE", URL, nil)
	if err != nil {
		return err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return err
	}

	_ = resp.Body.Close()

	return nil
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
		if req.Method == "POST" && (strings.Contains(req.URL.String(), "/file") || strings.Contains(req.URL.String(), "/stream")) {
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

	if resp.StatusCode != 200 {
		return nil, parseErrorResponse(resp)
	}

	// Log response
	if c.Debug {
		dumpBody := true
		if req.Method == "GET" && (strings.Contains(req.URL.String(), "/file") || strings.Contains(req.URL.String(), "/archive")) {
			dumpBody = false
		}
		dump, err := httputil.DumpResponse(resp, dumpBody)
		if err == nil {
			fmt.Println(string(dump))
		} else {
			return nil, fmt.Errorf("Unable to dump HTTP response : %s", err)
		}
	}

	return resp, nil
}

func parseErrorResponse(resp *http.Response) (err error) {
	defer resp.Body.Close()

	// Reade response body

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse json error

	result := new(common.Result)
	err = json.Unmarshal(body, result)
	if err == nil && result.Message != "" {
		return fmt.Errorf("%s : %s", resp.Status, result.Message)
	} else if len(body) > 0 {
		return fmt.Errorf("%s : %s", resp.Status, string(body))
	} else {
		return fmt.Errorf("%s", resp.Status)
	}
}
