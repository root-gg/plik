package gcs

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
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
	return config
}

// Backend object
type Backend struct {
	Config *Config
	client *storage.Client
}

// NewBackend instantiate a new GCS Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend, err error) {
	b = new(Backend)
	b.Config = config

	// Initialize GCS client
	b.client, err = storage.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to create GCS client : %s", err)
	}

	return b, nil
}

// GetFile implementation for Google Cloud Storage Data Backend
func (b *Backend) GetFile(file *common.File) (reader io.ReadCloser, err error) {
	// Get object name
	objectName := b.getObjectName(file.UploadID, file.ID)

	// Get the object
	reader, err = b.client.Bucket(b.Config.Bucket).Object(objectName).NewReader(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to get GCS object %s : %s", objectName, err)
	}

	return reader, nil
}

// AddFile implementation for Google Cloud Storage Data Backend
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	// Get object name
	objectName := b.getObjectName(file.UploadID, file.ID)

	// Get a writer
	wc := b.client.Bucket(b.Config.Bucket).Object(objectName).NewWriter(context.Background())
	defer wc.Close()

	_, err = io.Copy(wc, fileReader)
	if err != nil {
		return fmt.Errorf("Unable to write GCS object %s : %s", objectName, err)
	}

	return nil
}

// RemoveFile implementation for Google Cloud Storage Data Backend
func (b *Backend) RemoveFile(file *common.File) (err error) {
	// Get object name
	objectName := b.getObjectName(file.UploadID, file.ID)

	// Delete the object
	err = b.client.Bucket(b.Config.Bucket).Object(objectName).Delete(context.Background())
	if err != nil {
		// Ignore "file not found" errors
		if err == storage.ErrObjectNotExist {
			return nil
		}

		return fmt.Errorf("Unable to remove gcs object %s : %s", objectName, err)
	}

	return nil
}

func (b *Backend) getObjectName(uploadID string, fileID string) string {
	if b.Config.Folder != "" {
		return fmt.Sprintf("%s/%s.%s", b.Config.Folder, uploadID, fileID)
	}
	return fmt.Sprintf("%s.%s", uploadID, fileID)
}
