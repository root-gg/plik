/**

    Plik upload server

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

package gcs

import (
	"context"
	"fmt"
	"io"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
)

// Backend object
type Backend struct {
	config *BackendConfig
	lock   *sync.RWMutex
	client *storage.Client
}

// NewGoogleCloudStorageBackend instantiate a new Google Cloud Storaga Data Backend
// from configuration passed as argument
func NewGoogleCloudStorageBackend(config map[string]interface{}) (sb *Backend) {
	sb = new(Backend)
	sb.config = NewGoogleCloudStorageBackendConfig(config)
	sb.lock = new(sync.RWMutex)
	return sb
}

// GetFile implementation for Swift Data Backend
func (sb *Backend) GetFile(ctx *juliet.Context, upload *common.Upload, fileID string) (reader io.ReadCloser, err error) {
	log := common.GetLogger(ctx)

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = log.EWarningf("unable to get gcs client: %s", err)
		return
	}

	// Get the object
	return sb.client.Bucket(sb.config.Bucket).Object(sb.getFileID(upload, fileID)).NewReader(context.Background())
}

// AddFile implementation for Swift Data Backend
func (sb *Backend) AddFile(ctx *juliet.Context, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	log := common.GetLogger(ctx)

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = log.EWarningf("unable to get gcs client: %s", err)
		return
	}

	// Get a writer
	wc := sb.client.Bucket(sb.config.Bucket).Object(sb.getFileID(upload, file.ID)).NewWriter(context.Background())
	defer wc.Close()

	_, err = io.Copy(wc, fileReader)
	if err != nil {
		return
	}

	return
}

// RemoveFile implementation for Swift Data Backend
func (sb *Backend) RemoveFile(ctx *juliet.Context, upload *common.Upload, fileID string) (err error) {
	log := common.GetLogger(ctx)

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = log.EWarningf("unable to get gcs client: %s", err)
		return
	}

	// Delete the object
	return sb.client.Bucket(sb.config.Bucket).Object(sb.getFileID(upload, fileID)).Delete(context.Background())
}

// RemoveUpload implementation for Swift Data Backend
// Iterates on each upload file and call RemoveFile
func (sb *Backend) RemoveUpload(ctx *juliet.Context, upload *common.Upload) (err error) {
	log := common.GetLogger(ctx)

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = log.EWarningf("unable to get gcs client: %s", err)
		return
	}

	// Remove all files
	for fileID := range upload.Files {
		uuid := sb.getFileID(upload, fileID)
		err = sb.client.Bucket(sb.config.Bucket).Object(uuid).Delete(context.Background())
		if err != nil {
			err = log.EWarningf("unable to remove object %s : %s", uuid, err)
		}
	}

	return
}

func (sb *Backend) getFileID(upload *common.Upload, fileID string) string {
	if sb.config.Folder != "" {
		return fmt.Sprintf("%s/%s.%s", sb.config.Folder, upload.ID, fileID)
	}

	return fmt.Sprintf("%s.%s", upload.ID, fileID)
}

func (sb *Backend) initClient() (err error) {

	// Lock
	sb.lock.Lock()
	defer sb.lock.Unlock()

	// Get the GCS client
	if sb.client == nil {
		sb.client, err = storage.NewClient(context.Background())
		if err != nil {
			return
		}
	}

	return
}
