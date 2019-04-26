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
	"fmt"
	"io"
	"net/url"
	"strconv"
	"sync"

	"github.com/root-gg/plik/server/common"
)

// UploadParams store the different options available when uploading file to a Plik server
// One should add files to the upload before calling Create or Upload
type UploadParams struct {
	Stream    bool // Don't store the file on the server
	OneShot   bool // Force deletion of the file from the server after the first download
	Removable bool // Allow upload and upload files to be removed from the server at any time

	TTL      int    // Time in second before automatic deletion of the file from the server
	Comments string // Arbitrary comment to attach to the upload ( the web interface support markdown language )

	Token string // Authentication token to link an upload to a Plik user

	Login    string // HttpBasic protection for the upload
	Password string // Login and Password

	Yubikey string // Yubikey OTP
}

// Upload store the necessary data to upload files to a Plik server
type Upload struct {
	UploadParams
	client  *Client        // Client that makes the actual HTTP calls
	files   []*File        // Files to upload
	details *common.Upload // Upload metadata ( once created )
}

// newUpload create and initialize a new Upload object
func newUpload(client *Client) (upload *Upload) {
	upload = new(Upload)
	upload.client = client

	// Copy the default upload params from the client
	upload.UploadParams = *client.UploadParams

	return upload
}

func newUploadFromParams(client *Client, params *common.Upload) (upload *Upload) {
	upload = newUpload(client)
	upload.Stream = params.Stream
	upload.OneShot = params.OneShot
	upload.Removable = params.Removable
	upload.TTL = params.TTL
	upload.Comments = params.Comments
	upload.details = params

	for _, file := range params.Files {
		upload.add(newFileFromParams(upload, file))
	}

	return upload
}

// AddFiles add one or several files to be uploaded
func (upload *Upload) add(file *File) {
	upload.files = append(upload.files, file)
}

// AddFileFromPath add a new file from a filesystem path
func (upload *Upload) AddFileFromPath(name string) (file *File, err error) {
	file, err = newFileFromPath(upload, name)
	if err != nil {
		return nil, err
	}
	upload.add(file)
	return file, nil
}

// AddFileFromReader add a new file from a filename and io.Reader
func (upload *Upload) AddFileFromReader(name string, reader io.Reader) (file *File) {
	file = newFileFromReader(upload, name, reader)
	upload.add(file)
	return file
}

// AddFileFromReadCloser add a new file from a filename and io.ReadCloser
func (upload *Upload) AddFileFromReadCloser(name string, reader io.ReadCloser) (file *File) {
	file = newFileFromReadCloser(upload, name, reader)
	upload.add(file)
	return file
}

// Details return the upload details returned by the server
func (upload *Upload) Details() (details *common.Upload) {
	return upload.details
}

// getParams returns a common.Upload to be passed to internal methods
func (upload *Upload) getParams() (params *common.Upload) {
	params = common.NewUpload()
	params.Stream = upload.Stream
	params.OneShot = upload.OneShot
	params.Removable = upload.Removable
	params.TTL = upload.TTL
	params.Comments = upload.Comments
	params.Token = upload.Token
	params.Login = upload.Login
	params.Password = upload.Password

	if upload.HasBeenCreated() {
		params.ID = upload.details.ID
		params.UploadToken = upload.details.UploadToken
	}

	for i, file := range upload.files {
		fileParams := file.getParams()
		if fileParams.ID == "" {
			reference := strconv.Itoa(i)
			fileParams.Reference = reference
			params.Files[reference] = fileParams
		} else {
			params.Files[fileParams.ID] = fileParams
		}
	}
	return params
}

// Files Return the upload files
func (upload *Upload) Files() (files []*File) {
	return upload.files
}

// HasBeenCreated return true if the upload has been created server side ( has an ID )
func (upload *Upload) HasBeenCreated() bool {
	return upload.details != nil
}

// ID returns the upload ID if the upload has been created server side
func (upload *Upload) ID() string {
	if upload.details == nil {
		return ""
	}
	return upload.details.ID
}

// GetURL returns the URL page of the upload
func (upload *Upload) GetURL() (u *url.URL, err error) {
	if !upload.HasBeenCreated() {
		return nil, fmt.Errorf("Upload has not been created yet")
	}

	fileURL := fmt.Sprintf("%s/?id=%s", upload.client.URL, upload.ID())

	// Parse to get a nice escaped url
	return url.Parse(fileURL)
}

// Create a new empty upload on a Plik Server
func (upload *Upload) Create() (err error) {
	uploadParams := upload.getParams()

	// Crate the upload on the Plik server
	uploadInfo, err := upload.client.create(uploadParams)
	if err != nil {
		return err
	}

	// Keep all the uploadInfo but we are mostly interested in the upload ID
	upload.details = uploadInfo

	// Here also we keep all the file info but we are also mostly interested in the file ID
	// We use the reference system to avoid problems if uploading several files with the same filename
LOOP:
	for _, file := range upload.details.Files {
		for i, f := range upload.files {
			reference := strconv.Itoa(i)

			if file.Reference == reference {
				f.details = file // Update the file info
				continue LOOP
			}
		}
		return fmt.Errorf("No file match for file reference %s", file.Reference)
	}

	return nil
}

// Upload uploads all files of the upload in parallel
func (upload *Upload) Upload() (err error) {
	if !upload.HasBeenCreated() {
		err = upload.Create()
		if err != nil {
			return err
		}
	}

	ok := true
	var mu sync.Mutex
	fail := func() {
		mu.Lock()
		defer mu.Unlock()
		ok = false
	}

	var wg sync.WaitGroup
	for _, file := range upload.files {
		if file.HasBeenUploaded() {
			continue
		}
		wg.Add(1)
		go func(file *File) {
			defer wg.Done()
			err := file.Upload()
			if err != nil {
				fail()
				return
			}
		}(file)
	}

	wg.Wait()

	if !ok {
		return fmt.Errorf("Failed to upload at least one file. Check each file status for more details")
	}

	return nil
}

// DownloadZipArchive downloads all the upload files in a zip archive
func (upload *Upload) DownloadZipArchive() (reader io.ReadCloser, err error) {
	return upload.client.downloadArchive(upload.getParams())
}

// Delete remove the upload and all the associated files from the remote server
func (upload *Upload) Delete() (err error) {
	return upload.client.removeUpload(upload.getParams())
}
