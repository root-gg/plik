package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure S3 Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for S3 data backend
type Config struct {
	Endpoint              string
	AccessKeyID           string
	SecretAccessKey       string
	Bucket                string
	Location              string
	Prefix                string
	PartSize              uint64
	PartUploadConcurrency uint
	UseSSL                bool
	SendContentMd5        bool
	SSE                   string
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]any) (config *Config) {
	config = new(Config)
	config.Bucket = "plik"
	config.Location = "us-east-1"
	config.PartSize = 16 * 1024 * 1024 // 16MiB
	config.PartUploadConcurrency = 4
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
	if config.PartSize < 5*1024*1024 {
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

// NewBackend instantiate a new S3 Data Backend
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
func (b *Backend) GetFile(file *common.File) (reader io.ReadSeekCloser, err error) {
	getOpts := minio.GetObjectOptions{}

	// Configure server side encryption
	getOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return nil, err
	}

	// Try new object name format first ({uploadID}.{fileID})
	obj, err := b.client.GetObject(context.TODO(), b.config.Bucket, b.getObjectName(file.UploadID, file.ID), getOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to get s3 object : %s", err)
	}

	// Peek to check if the object exists (GetObject does only basic checking)
	_, statErr := obj.Stat()
	if statErr == nil {
		return obj, nil
	}
	_ = obj.Close()

	// Check if it's a "not found" error before falling back
	errResponse := minio.ToErrorResponse(statErr)
	if errResponse.Code != "NoSuchKey" {
		return nil, fmt.Errorf("unable to get s3 object : %s", statErr)
	}

	// Fall back to legacy object name format ({fileID}) for backward compatibility
	return b.client.GetObject(context.TODO(), b.config.Bucket, b.getObjectNameLegacy(file.ID), getOpts)
}

// AddFile implementation for S3 Data Backend
//
// Uses a buffer-then-decide strategy:
//   - Read up to PartSize bytes from the stream
//   - If EOF is reached: single PUT with the exact size (optimal for small files)
//   - If more data remains: multipart upload with optional parallelism
//
// This avoids the overhead of multipart initiation for small files while correctly
// handling unknown-size streams (e.g. E2EE where the encrypted size differs from
// the declared size).
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	putOpts := b.newPutObjectOptions(file.Type)

	// Configure server side encryption
	putOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return err
	}

	objectName := b.getObjectName(file.UploadID, file.ID)
	partSize := b.config.PartSize

	// Buffer up to partSize+1 bytes to determine upload strategy.
	// Using io.CopyN + bytes.Buffer instead of a fixed make([]byte, partSize)
	// so small files only allocate memory proportional to their actual size.
	var buf bytes.Buffer
	n, readErr := io.CopyN(&buf, fileReader, int64(partSize)+1)

	switch {
	case readErr == io.EOF:
		// File fits in a single part → single PUT with exact size (1 HTTP request)
		_, err = b.client.PutObject(context.TODO(), b.config.Bucket, objectName,
			bytes.NewReader(buf.Bytes()), n, putOpts)

	case readErr == nil:
		// Read partSize+1 bytes, more data to come → multipart upload
		// https://github.com/minio/minio-go/issues/989
		// We default to 16MiB parts which allow files up to 156GiB (10000 × 16MiB).
		putOpts.PartSize = partSize

		// Enable parallel part uploads if configured
		concurrency := max(b.config.PartUploadConcurrency, 1)
		if concurrency > 1 {
			putOpts.ConcurrentStreamParts = true
			putOpts.NumThreads = concurrency
		}

		// Chain the buffered data with the remaining stream
		combined := io.MultiReader(&buf, fileReader)
		_, err = b.client.PutObject(context.TODO(), b.config.Bucket, objectName,
			combined, -1, putOpts)

	default:
		err = readErr
	}

	return err
}

func (b *Backend) newPutObjectOptions(contentType string) minio.PutObjectOptions {
	return minio.PutObjectOptions{
		ContentType:    contentType,
		SendContentMd5: b.config.SendContentMd5,
	}
}

// RemoveFile implementation for S3 Data Backend
func (b *Backend) RemoveFile(file *common.File) (err error) {
	// Build stat options with server side encryption if configured
	statOpts := minio.StatObjectOptions{}
	statOpts.ServerSideEncryption, err = b.getServerSideEncryption(file)
	if err != nil {
		return err
	}

	// Try new object name format first ({uploadID}.{fileID})
	objectName := b.getObjectName(file.UploadID, file.ID)
	removed, err := b.removeObject(objectName, statOpts)
	if err != nil {
		return fmt.Errorf("unable to remove s3 object %s : %s", objectName, err)
	}
	if removed {
		return nil
	}

	// Fall back to legacy object name format ({fileID}) for backward compatibility
	legacyName := b.getObjectNameLegacy(file.ID)
	_, err = b.removeObject(legacyName, statOpts)
	if err != nil {
		return fmt.Errorf("unable to remove s3 object %s : %s", legacyName, err)
	}

	return nil
}

// removeObject checks if an object exists via StatObject, then removes it.
// S3's DeleteObject API returns success even for non-existent objects,
// so we must check existence first to know whether the object was actually found.
// The VersionID from StatObject is passed to RemoveObject to ensure permanent
// deletion on versioned buckets (otherwise only a delete marker is created).
// Returns (true, nil) if removed, (false, nil) if not found, (false, err) on error.
func (b *Backend) removeObject(objectName string, statOpts minio.StatObjectOptions) (removed bool, err error) {
	info, err := b.client.StatObject(context.TODO(), b.config.Bucket, objectName, statOpts)
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}

	removeOpts := minio.RemoveObjectOptions{VersionID: info.VersionID}
	err = b.client.RemoveObject(context.TODO(), b.config.Bucket, objectName, removeOpts)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Backend) getObjectName(uploadID, fileID string) string {
	name := fmt.Sprintf("%s.%s", uploadID, fileID)
	if b.config.Prefix != "" {
		return fmt.Sprintf("%s/%s", b.config.Prefix, name)
	}
	return name
}

// getObjectNameLegacy returns the legacy object name format for backward compatibility
// with objects stored before the {uploadID}.{fileID} naming convention.
func (b *Backend) getObjectNameLegacy(fileID string) string {
	if b.config.Prefix != "" {
		return fmt.Sprintf("%s/%s", b.config.Prefix, fileID)
	}
	return fileID
}
