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
}

// Upload store the necessary data to upload files to a Plik server
type Upload struct {
	UploadParams
	client *Client // Client that makes the actual HTTP calls
	files  []*File // Files to upload

	lock     sync.Mutex     // The following fields need to be protected
	metadata *common.Upload // Upload metadata ( once created )

	done chan struct{} // Used to synchronize Create() calls
	err  error         // If an error occurs during a Create() call this will be set
}

// newUpload create and initialize a new Upload object
func newUpload(client *Client) (upload *Upload) {
	upload = new(Upload)
	upload.client = client

	// Copy the default upload params from the client
	upload.UploadParams = *client.UploadParams

	return upload
}

// newUploadFromMetadata create and initialize a new Upload object from server metadata
func newUploadFromMetadata(client *Client, uploadMetadata *common.Upload) (upload *Upload) {
	upload = newUpload(client)
	upload.Stream = uploadMetadata.Stream
	upload.OneShot = uploadMetadata.OneShot
	upload.Removable = uploadMetadata.Removable
	upload.TTL = uploadMetadata.TTL
	upload.Comments = uploadMetadata.Comments
	upload.metadata = uploadMetadata

	// Generate files
	for _, file := range uploadMetadata.Files {
		upload.add(newFileFromParams(upload, file))
	}

	// Remove files from metadata as this could be misleading
	uploadMetadata.Files = nil

	return upload
}

// AddFiles add one or several files to be uploaded
func (upload *Upload) add(file *File) {
	upload.lock.Lock()
	defer upload.lock.Unlock()

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

// Metadata return the upload metadata returned by the server
func (upload *Upload) Metadata() (details *common.Upload) {
	upload.lock.Lock()
	defer upload.lock.Unlock()

	return upload.metadata
}

// getParams returns a common.Upload to be passed to internal methods
func (upload *Upload) getParams() (params *common.Upload) {
	upload.lock.Lock()
	defer upload.lock.Unlock()

	params = &common.Upload{}
	params.Stream = upload.Stream
	params.OneShot = upload.OneShot
	params.Removable = upload.Removable
	params.TTL = upload.TTL
	params.Comments = upload.Comments
	params.Token = upload.Token
	params.Login = upload.Login
	params.Password = upload.Password

	if upload.metadata != nil {
		params.ID = upload.metadata.ID
		params.UploadToken = upload.metadata.UploadToken
	}

	for i, file := range upload.files {
		fileParams := file.getParams()
		if fileParams.ID == "" {
			reference := strconv.Itoa(i)
			fileParams.Reference = reference
		}
		params.Files = append(params.Files, fileParams)
	}

	return params
}

// Files Return the upload files
func (upload *Upload) Files() (files []*File) {
	upload.lock.Lock()
	defer upload.lock.Unlock()

	return upload.files
}

// ID returns the upload ID if the upload has been created server side
func (upload *Upload) ID() string {
	metadata := upload.Metadata()
	if metadata == nil {
		return ""
	}
	return metadata.ID
}

// ready ensure that only one upload occurs ( like sync.Once )
// the first call to Create() will proceed ( abort false ) and must close the done channel once done
// subsequent calls to Create() will abort ( abort true ) and must :
//  - wait on the done channel for the former Create() call to complete ( if not nil )
//  - return the error in upload.err
func (upload *Upload) ready() (done chan struct{}, abort bool) {
	upload.lock.Lock()
	defer upload.lock.Unlock()

	// Create is in progress or finished
	if upload.done != nil {
		return upload.done, true
	}

	// Upload has already been created
	if upload.metadata != nil {
		return nil, true
	}

	// Grab the lock by setting this channel
	upload.done = make(chan struct{})
	return upload.done, false
}

// Create a new empty upload on a Plik Server
func (upload *Upload) Create() (err error) {

	// synchronize
	done, abort := upload.ready()
	if abort {
		if done != nil {
			<-done
		}
		return upload.err
	}

	// Get upload parameters to send to the server
	uploadParams := upload.getParams()

	// Crate the upload on the server
	uploadMetadata, err := upload.client.create(uploadParams)

	// update upload with API call result
	upload.lock.Lock()
	if err == nil {
		err = upload.updateUpload(uploadMetadata)
	}
	if err != nil {
		upload.err = err
	}
	upload.lock.Unlock()

	// notify that we are done
	close(done)

	return err
}

// update the upload and files metadata with the result from the Create() API call
func (upload *Upload) updateUpload(uploadMetadata *common.Upload) (err error) {
	upload.metadata = uploadMetadata

	// Plik uses a reference system to avoid problems if uploading several files with the same filename
LOOP:
	for _, file := range uploadMetadata.Files {
		for i, f := range upload.files {
			reference := strconv.Itoa(i)

			if file.Reference == reference {
				f.lock.Lock()
				f.metadata = file // Update the file metadata
				f.lock.Unlock()
				continue LOOP
			}
		}
		return fmt.Errorf("no file match for file reference %s", file.Reference)
	}

	// Remove files from metadata as this could be misleading
	uploadMetadata.Files = nil

	return nil
}

// Upload uploads all files of the upload in parallel
func (upload *Upload) Upload() (err error) {

	// initialize the upload if not already done
	err = upload.Create()
	if err != nil {
		return err
	}

	files := upload.Files()
	errors := make(chan error, len(files))

	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file *File) {
			defer wg.Done()
			errors <- file.Upload()
		}(file)
	}

	// Wait for all files to be uploaded
	wg.Wait()

	// Check for errors
	close(errors)
	for err := range errors {
		if err != nil {
			// Print all errors in Debug mode
			if upload.client.Debug {
				for _, file := range files {
					if file.Error() != nil {
						fmt.Println(file.Error().Error())
					}
				}
			}
			return fmt.Errorf("failed to upload at least one file. Check each file status for more details")
		}
	}

	return nil
}

// GetURL returns the URL page of the upload
func (upload *Upload) GetURL() (u *url.URL, err error) {

	// Get upload metadata
	uploadMetadata := upload.Metadata()
	if uploadMetadata == nil || uploadMetadata.ID == "" {
		return nil, fmt.Errorf("upload has not been created yet")
	}

	fileURL := fmt.Sprintf("%s/#/?id=%s", upload.client.URL, uploadMetadata.ID)

	// Parse to get a nice escaped url
	return url.Parse(fileURL)
}

// GetAdminURL return the URL page of the upload with upload admin rights
func (upload *Upload) GetAdminURL() (u *url.URL, err error) {
	// Get upload metadata
	uploadMetadata := upload.Metadata()
	if uploadMetadata == nil || uploadMetadata.ID == "" {
		return nil, fmt.Errorf("upload has not been created yet")
	}

	fileURL := fmt.Sprintf("%s/#/?id=%s&uploadToken=%s", upload.client.URL, uploadMetadata.ID, uploadMetadata.UploadToken)

	// Parse to get a nice escaped url
	return url.Parse(fileURL)
}

// DownloadZipArchive downloads all the upload files in a zip archive
func (upload *Upload) DownloadZipArchive() (reader io.ReadCloser, err error) {
	return upload.client.downloadArchive(upload.getParams())
}

// Delete remove the upload and all the associated files from the remote server
func (upload *Upload) Delete() (err error) {
	return upload.client.removeUpload(upload.getParams())
}
