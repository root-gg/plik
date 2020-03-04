package data

import (
	"io"

	"github.com/root-gg/plik/server/common"
)

// Backend interface describes methods that data backend
// must implements to be compatible with plik.
type Backend interface {
	AddFile(file *common.File, reader io.Reader) (backendDetails string, err error)
	GetFile(file *common.File) (reader io.ReadCloser, err error)
	RemoveFile(file *common.File) (err error)
}
