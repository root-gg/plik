package data

import (
	"io"

	"github.com/root-gg/plik/server/common"
)

// Backend interface describes methods that data backend
// must implements to be compatible with Plik.
type Backend interface {
	AddFile(file *common.File, reader io.Reader) (err error)
	GetFile(file *common.File) (reader io.ReadCloser, err error)
	// RemoveFile should not fail if the file is not found
	RemoveFile(file *common.File) (err error)
}
