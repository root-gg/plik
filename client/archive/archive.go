package archive

import (
	"errors"
	"github.com/root-gg/plik/client/archive/tar"
	"github.com/root-gg/plik/client/archive/zip"
	"io"
)

type ArchiveBackend interface {
	Configure(arguments map[string]interface{}) (err error)
	Archive(files []string, writer io.WriteCloser) (name string, err error)
	Comments() (comments string)
}

func NewArchiveBackend(name string, config map[string]interface{}) (backend ArchiveBackend, err error) {
	switch name {
	case "tar":
		backend, err = tar.NewTarBackend(config)
	case "zip":
		backend, err = zip.NewZipBackend(config)
	default:
		err = errors.New("Invalid archive backend")
	}
	return
}
