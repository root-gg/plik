package s3

import (
	"fmt"
	"io"

	"github.com/minio/minio-go/v6"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure Swift Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for Swift data backend
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Location        string
	Prefix          string
	PartSize        uint64
	UseSSL          bool
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	config.Bucket = "plik"
	config.Location = "us-east-1"
	config.PartSize = 16 * 1000 * 1000 // 16MB
	utils.Assign(config, params)
	return
}

// Validate check config parameters
func (config *Config) Validate() error {
	if config.Endpoint == "" {
		return fmt.Errorf("missing endpoint")
	}
	if config.AccessKeyID == "" {
		return fmt.Errorf("missing access key ID")
	}
	if config.SecretAccessKey == "" {
		return fmt.Errorf("missing secret access key")
	}
	if config.Bucket == "" {
		return fmt.Errorf("missing bucket name")
	}
	if config.Location == "" {
		return fmt.Errorf("missing location")
	}
	if config.PartSize < 5*1000*1000 {
		return fmt.Errorf("invalid part size")
	}
	return nil
}

// Backend object
type Backend struct {
	config *Config
	client *minio.Client
}

// NewBackend instantiate a new OpenSwift Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend, err error) {
	b = new(Backend)
	b.config = config

	err = b.config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid s3 data backend config : %s", err)
	}

	b.client, err = minio.New(config.Endpoint, config.AccessKeyID, config.SecretAccessKey, config.UseSSL)
	if err != nil {
		return nil, err
	}

	// Check if bucket exists
	exists, err := b.client.BucketExists(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("unable to check if bucket %s exists : %s", config.Bucket, err)
	}

	if !exists {
		// Create bucket
		err = b.client.MakeBucket(config.Bucket, config.Location)
		if err != nil {
			return nil, fmt.Errorf("unable to create bucket %s : %s", config.Bucket, err)
		}
	}

	return b, nil
}

// GetFile implementation for S3 Data Backend
func (b *Backend) GetFile(file *common.File) (reader io.ReadCloser, err error) {
	return b.client.GetObject(b.config.Bucket, b.getObjectName(file.ID), minio.GetObjectOptions{})
}

// AddFile implementation for S3 Data Backend
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (backendDetails string, err error) {
	if file.Size > 0 {
		_, err = b.client.PutObject(b.config.Bucket, b.getObjectName(file.ID), fileReader, file.Size, minio.PutObjectOptions{ContentType: file.Type})
	} else {
		// https://github.com/minio/minio-go/issues/989
		// Minio defaults to 128MB chunks and has to actually allocate a buffer of this size before uploading the chunk
		// This can lead to very high memory usage when uploading a lot of small files in parallel
		// We default to 16MB which allow to store files up to 160GB ( 10000 chunks of 16MB ), feel free to adjust this parameter to your needs.
		_, err = b.client.PutObject(b.config.Bucket, b.getObjectName(file.ID), fileReader, -1, minio.PutObjectOptions{ContentType: file.Type, PartSize: b.config.PartSize})
	}
	return "", err
}

// RemoveFile implementation for S3 Data Backend
func (b *Backend) RemoveFile(file *common.File) (err error) {
	return b.client.RemoveObject(b.config.Bucket, b.getObjectName(file.ID))
}

func (b *Backend) getObjectName(name string) string {
	if b.config.Prefix != "" {
		return fmt.Sprintf("%s/%s", b.config.Prefix, name)
	}
	return name
}
