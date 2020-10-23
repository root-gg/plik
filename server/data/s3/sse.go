package s3

import (
	"encoding/json"
	"fmt"

	"github.com/minio/minio-go/v7/pkg/encrypt"

	"github.com/root-gg/plik/server/common"
)

// Build Server Side Encryption configuration
func (b *Backend) getServerSideEncryption(file *common.File) (sse encrypt.ServerSide, err error) {
	switch encrypt.Type(b.config.SSE) {
	case "":
		return nil, nil
	case encrypt.S3:
		return encrypt.NewSSE(), nil
	case encrypt.SSEC:
		key, err := getServerSideEncryptionKey(file)
		if err != nil {
			return nil, fmt.Errorf("unable to get Server Side Encryption Key : %s", err)
		}
		return encrypt.NewSSEC([]byte(key))
	case encrypt.KMS:
		return nil, fmt.Errorf("KMS server side encryption is not yet implemented")
	default:
		return nil, fmt.Errorf("invalid SSE type %s", b.config.SSE)
	}
}

// Generate a 32Bytes / 256bits encryption key
func genServerSideEncryptionKey() string {
	return common.GenerateRandomID(32)
}

// Get the SSE Key from the file backend details or generate one and store it in the file backend details
func getServerSideEncryptionKey(file *common.File) (key string, err error) {
	// Retrieve the SSE Key from the backend details
	if file.BackendDetails != "" {
		backendDetails := &BackendDetails{}
		err = json.Unmarshal([]byte(file.BackendDetails), backendDetails)
		if err != nil {
			return "", fmt.Errorf("unable to deserialize backend details : %s", err)
		}

		if backendDetails.SSEKey != "" {
			return backendDetails.SSEKey, nil
		}
	}

	key = genServerSideEncryptionKey()

	// Store the SSE Key in the backend details
	err = setServerSideEncryptionKey(file, key)
	if err != nil {
		return "", err
	}

	return key, nil
}

// Add the SSE Key to the file backend details
func setServerSideEncryptionKey(file *common.File, key string) (err error) {
	backendDetails := &BackendDetails{}

	if file.BackendDetails != "" {
		err = json.Unmarshal([]byte(file.BackendDetails), backendDetails)
		if err != nil {
			return fmt.Errorf("unable to deserialize backend details : %s", err)
		}
	}

	backendDetails.SSEKey = key

	backendDetailsJSON, err := json.Marshal(backendDetails)
	if err != nil {
		return fmt.Errorf("unable to serialize backend details : %s", err)
	}

	file.BackendDetails = string(backendDetailsJSON)
	return nil
}
