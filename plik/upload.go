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

	Yubikey string // Yubikey OTP to protect the upload
}

// Upload store the necessary data to upload files to a Plik server
type Upload struct {
	*UploadParams
	client     *Client        // Client that makes the actual HTTP calls
	files      []*File        // Files to upload
	uploadInfo *common.Upload // Upload metadata ( once created )
}

// newUpload create and initialize a new Upload object
func newUpload(client *Client) (upload *Upload) {
	upload = new(Upload)
	upload.client = client

	// Get the default upload params from the client
	upload.UploadParams = client.UploadParams

	return upload
}

// AddFiles add one or several files to be uploaded
func (upload *Upload) AddFiles(files ...*File) {
	for _, file := range files {
		// We use the reference to avoid using filenames to match files with their server side generated ID
		// Correct use of an array would have been a way better option
		i := strconv.Itoa(len(upload.files))
		file.Reference = i
		upload.files = append(upload.files, file)
	}
}

// Files Return the upload files
func (upload *Upload) Files() (files []*File) {
	return upload.files
}

// HasBeenCreated return true if the upload has been created server side ( has an ID )
func (upload *Upload) HasBeenCreated() bool {
	return upload.uploadInfo != nil
}

// Info returns the upload info if the upload has been created server side
func (upload *Upload) Info() *common.Upload {
	return upload.uploadInfo
}

// ID returns the upload ID if the upload has been created server side
func (upload *Upload) ID() string {
	if upload.uploadInfo == nil {
		return ""
	}
	return upload.uploadInfo.ID
}

// GetUploadURL returns the URL page of the upload
func (upload *Upload) GetUploadURL() (u *url.URL, err error) {
	if !upload.HasBeenCreated() {
		return nil, fmt.Errorf("Upload has not been created yet")
	}

	fileURL := fmt.Sprintf("%s/?id=%s", upload.client.URL, upload.ID())

	// Parse to get a nice escaped url
	return url.Parse(fileURL)
}

// GetFileURL returns the URL to download the file
func (upload *Upload) GetFileURL(file *File) (URL *url.URL, err error) {
	if !upload.HasBeenCreated() {
		return nil, fmt.Errorf("Upload has not been created yet")
	}

	if file.ID == "" {
		return nil, fmt.Errorf("Upload has not been uploaded yet")
	}

	mode := "file"
	if upload.Stream {
		mode = "stream"
	}

	var domain string
	if upload.uploadInfo.DownloadDomain != "" {
		domain = upload.uploadInfo.DownloadDomain
	} else {
		domain = upload.client.URL
	}

	fileURL := fmt.Sprintf("%s/%s/%s/%s/%s", domain, mode, upload.ID(), file.ID, file.Name)

	// Parse to get a nice escaped url
	return url.Parse(fileURL)
}

// getUploadParams returns a common.Upload from the uploadParams and the files to be uploaded
func (upload *Upload) getUploadParams() (params *common.Upload) {
	params = common.NewUpload()
	params.Stream = upload.Stream
	params.OneShot = upload.OneShot
	params.Removable = upload.Removable
	params.TTL = upload.TTL
	params.Comments = upload.Comments
	params.Token = upload.Token
	if upload.Login != "" && upload.Password != "" {
		params.ProtectedByPassword = true
		params.Login = upload.Login
		params.Password = upload.Password
	}
	if params.Yubikey != "" {
		params.ProtectedByYubikey = true
		params.Yubikey = upload.Yubikey
	}

	for _, file := range upload.files {
		params.Files[file.Reference] = file.File
	}

	return params
}

// Create a new empty upload on a Plik Server
func (upload *Upload) Create() (err error) {
	uploadParams := upload.getUploadParams()

	// Crate the upload on the Plik server
	uploadInfo, err := upload.client.Create(uploadParams)
	if err != nil {
		return err
	}

	// Keep all the uploadInfo but we are mostly interested in the upload ID
	upload.uploadInfo = uploadInfo

	// Here also we keep all the file info but we are also mostly interested in the file ID
	// We use the reference system to avoid problems if uploading several files with the same filename
LOOP:
	for _, file := range upload.uploadInfo.Files {
		for _, f := range upload.files {
			if file.Reference == f.Reference {
				f.File = file // Update the file info
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
		wg.Add(1)
		go func(file *File) {
			defer wg.Done()
			err := upload.UploadFile(file)
			if err != nil {
				fail()
				return
			}
		}(file)
	}

	wg.Wait()

	if ok {
		return nil
	}

	return fmt.Errorf("Failed to upload at least one file. Check each file status for more details")
}

// UploadFile uploads a single file.
func (upload *Upload) UploadFile(file *File) (err error) {
	if !upload.HasBeenCreated() {
		return fmt.Errorf("Upload must be created first")
	}

	fileInfo, err := upload.client.UploadFile(upload.uploadInfo, file.File, file.Reader)
	if err == nil {
		file.File = fileInfo
	} else {
		file.Status = "error"
		file.Error = err
	}

	// Call the done callback before upload.Upload() returns
	if file.Done != nil {
		file.Done()
	}

	return err
}
