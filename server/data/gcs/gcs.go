package gcs

import (
	"context"
	"fmt"
	"io"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	"github.com/root-gg/utils"
)

// Ensure File Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for Google Cloud Storage data backend
type Config struct {
	Bucket string
	Folder string
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	utils.Assign(config, params)
	return
}

// Backend object
type Backend struct {
	Config *Config
	client *storage.Client
	lock   *sync.RWMutex
}

// NewBackend instantiate a new File Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend) {
	b = new(Backend)
	b.Config = config
	b.lock = new(sync.RWMutex)
	return
}

// GetFile implementation for Google Cloud Storage Data Backend
func (sb *Backend) GetFile(file *common.File) (reader io.ReadCloser, err error) {

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = fmt.Errorf("unable to get gcs client: %s", err)
		return
	}

	// Get the object
	return sb.client.Bucket(sb.Config.Bucket).Object(sb.getFileID(file.UploadID, file.ID)).NewReader(context.Background())
}

// AddFile implementation for Google Cloud Storage Data Backend
func (sb *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = fmt.Errorf("unable to get gcs client: %s", err)
		return
	}

	// Get a writer
	wc := sb.client.Bucket(sb.Config.Bucket).Object(sb.getFileID(file.UploadID, file.ID)).NewWriter(context.Background())
	defer wc.Close()

	_, err = io.Copy(wc, fileReader)
	if err != nil {
		return
	}

	return
}

// RemoveFile implementation for Google Cloud Storage Data Backend
func (sb *Backend) RemoveFile(file *common.File) (err error) {

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = fmt.Errorf("unable to get gcs client: %s", err)
		return
	}

	// Delete the object
	err = sb.client.Bucket(sb.Config.Bucket).Object(sb.getFileID(file.UploadID, file.ID)).Delete(context.Background())
	if err != nil {
		if err == storage.ErrObjectNotExist {
			err = nil
		}
	}

	return
}

// RemoveUpload implementation for Google Cloud Storage Data Backend
// Iterates on each upload file and call RemoveFile
func (sb *Backend) RemoveUpload(ctx *juliet.Context, upload *common.Upload) (err error) {

	// Get the GCS client
	err = sb.initClient()
	if err != nil {
		err = fmt.Errorf("unable to get gcs client: %s", err)
		return
	}

	// Remove all files
	for _, file := range upload.Files {
		uuid := sb.getFileID(file.UploadID, file.ID)
		err = sb.client.Bucket(sb.Config.Bucket).Object(uuid).Delete(context.Background())
		if err != nil {
			if err == storage.ErrObjectNotExist {
				err = nil
			} else {
				return fmt.Errorf("unable to remove object %s : %s", uuid, err)
			}
		}
	}

	return
}

func (sb *Backend) getFileID(uploadID string, fileID string) string {
	if sb.Config.Folder != "" {
		return fmt.Sprintf("%s/%s.%s", sb.Config.Folder, uploadID, fileID)
	}

	return fmt.Sprintf("%s.%s", uploadID, fileID)
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
