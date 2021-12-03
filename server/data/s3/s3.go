package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	SSE             string
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

// BackendDetails additional backend metadata
type BackendDetails struct {
	SSEKey string
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

	b.client, err = minio.New(config.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(config.AccessKeyID, config.SecretAccessKey, ""),
		//Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Secure: config.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// Check if bucket exists
	exists, err := b.client.BucketExists(context.TODO(), config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("unable to check if bucket %s exists : %s", config.Bucket, err)
	}

	if !exists {
		// Create bucket
		err = b.client.MakeBucket(context.TODO(), config.Bucket, minio.MakeBucketOptions{Region: config.Location})
		if err != nil {
			return nil, fmt.Errorf("unable to create bucket %s : %s", config.Bucket, err)
		}
	}

	return b, nil
}

// GetFile implementation for S3 Data Backend
func (b *Backend) GetFile(file *common.File) (reader io.ReadCloser, err error) {
	getOpts := minio.GetObjectOptions{}

	// Configure server side encryption
	getOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return nil, err
	}

	// This does only very basic checking and basically always return nil, error will happen when reading from the reader
	return b.client.GetObject(context.TODO(), b.config.Bucket, b.getObjectName(file.ID), getOpts)
}

// AddFile implementation for S3 Data Backend
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	putOpts := minio.PutObjectOptions{ContentType: file.Type}

	// Configure server side encryption
	putOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return err
	}

	if file.Size > 0 {
		_, err = b.client.PutObject(context.TODO(), b.config.Bucket, b.getObjectName(file.ID), fileReader, file.Size, putOpts)
	} else {
		// https://github.com/minio/minio-go/issues/989
		// Minio defaults to 128MB chunks and has to actually allocate a buffer of this size before uploading the chunk
		// This can lead to very high memory usage when uploading a lot of small files in parallel
		// We default to 16MB which allow to store files up to 160GB ( 10000 chunks of 16MB ), feel free to adjust this parameter to your needs.
		putOpts.PartSize = b.config.PartSize

		_, err = b.client.PutObject(context.TODO(), b.config.Bucket, b.getObjectName(file.ID), fileReader, -1, putOpts)
	}
	return err
}

// RemoveFile implementation for S3 Data Backend
func (b *Backend) RemoveFile(file *common.File) (err error) {
	objectName := b.getObjectName(file.ID)
	err = b.client.RemoveObject(context.TODO(), b.config.Bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		// Ignore "file not found" errors
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return nil
		}
		return fmt.Errorf("Unable to remove s3 object %s : %s", objectName, err)
	}

	return nil
}

func (b *Backend) getObjectName(name string) string {
	if b.config.Prefix != "" {
		return fmt.Sprintf("%s/%s", b.config.Prefix, name)
	}
	return name
}
