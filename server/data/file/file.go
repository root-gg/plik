package file

import (
	"fmt"
	"io"
	"os"

	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
)

// Ensure File Data Backend implements data.Backend interface
var _ data.Backend = (*Backend)(nil)

// Config describes configuration for File Databackend
type Config struct {
	Directory string
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	config.Directory = "files" // Default upload directory is ./files
	utils.Assign(config, params)
	return
}

// Backend object
type Backend struct {
	Config *Config
}

// NewBackend instantiate a new File Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend) {
	b = new(Backend)
	b.Config = config
	return
}

// GetFile implementation for file data backend will search
// on filesystem the asked file and return its reading filehandle
func (b *Backend) GetFile(file *common.File) (reader io.ReadCloser, err error) {
	_, path, err := b.getPathCompat(file)
	if err != nil {
		return nil, err
	}

	// The file content will be piped directly
	// to the client response body
	reader, err = os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s : %s", path, err)
	}

	return reader, nil
}

// AddFile implementation for file data backend will creates a new file for the given upload
// and save it on filesystem with the given file reader
func (b *Backend) AddFile(file *common.File, fileReader io.Reader) (err error) {
	dir, path, err := b.getPath(file)
	if err != nil {
		return err
	}

	// Create directory
	err = os.MkdirAll(dir, 0777)
	if err != nil {
		return fmt.Errorf("unable to create upload directory")
	}

	// Create file
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file %s : %s", path, err)
	}

	// Copy file data from the client request body
	// to the file system
	_, err = io.Copy(out, fileReader)
	if err != nil {
		return fmt.Errorf("unable to save file %s : %s", path, err)
	}

	return nil
}

// RemoveFile implementation for file data backend will delete the given
// file from filesystem
func (b *Backend) RemoveFile(file *common.File) (err error) {
	_, path, err := b.getPathCompat(file)
	if err != nil {
		if err == errNoSuchFileOrDirectory {
			return nil
		}
		return err
	}

	// Remove file
	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("unable to remove %s : %s", path, err)
	}

	return nil
}

func (b *Backend) getPath(file *common.File) (dir string, path string, err error) {
	// To avoid too many files in the same directory
	// data directory is split in two levels the
	// first level is the 2 first chars from the file id
	// it gives 3844 possibilities reaching 65535 files per
	// directory at ~250.000.000 files uploaded.

	if file == nil || file.ID == "" || len(file.ID) < 3 || len(file.UploadID) < 3 {
		return "", "", fmt.Errorf("file not initialized")
	}

	dir = fmt.Sprintf("%s/%s", b.Config.Directory, file.ID[:2])
	path = fmt.Sprintf("%s/%s", dir, file.ID)

	return dir, path, nil
}

var errNoSuchFileOrDirectory = fmt.Errorf("no such file or directory")

func (b *Backend) getPathCompat(file *common.File) (dir string, path string, err error) {
	dir, path, err = b.getPath(file)
	if err != nil {
		return "", "", err
	}

	// Check file

	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		return dir, path, nil
	}
	if !os.IsNotExist(err) {
		return "", "", err
	}

	// For compatibility with <1.3 implementations

	dir = fmt.Sprintf("%s/%s/%s", b.Config.Directory, file.UploadID[:2], file.UploadID)
	path = fmt.Sprintf("%s/%s", dir, file.ID)

	info, err = os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return "", "", fmt.Errorf("file is a directory")
		}
		return dir, path, nil
	}
	if !os.IsNotExist(err) {
		return "", "", err
	}

	return "", "", errNoSuchFileOrDirectory
}
