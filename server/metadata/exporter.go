package metadata

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/golang/snappy"

	"github.com/root-gg/plik/server/common"
)

type metadataType int

const (
	metadataTypeUpload metadataType = iota
	metadataTypeFile
	metadataTypeUser
	metadataTypeToken
	metadataTypeSetting
)

type object struct {
	Type   metadataType
	Object interface{}
}

type exporter struct {
	writer     io.WriteCloser
	compressor *snappy.Writer
	encoder    *gob.Encoder
}

func newExporter(path string) (e *exporter, err error) {
	e = &exporter{}

	// Open file for writing
	e.writer, err = os.Create(path)
	if err != nil {
		return nil, err
	}

	// Snappy compressor
	e.compressor = snappy.NewBufferedWriter(e.writer)

	// Gob encoder
	gob.Register(&common.Upload{})
	gob.Register(&common.File{})
	gob.Register(&common.User{})
	gob.Register(&common.Token{})
	gob.Register(&common.Setting{})
	e.encoder = gob.NewEncoder(e.compressor)

	return e, nil
}

func (e *exporter) addUpload(upload *common.Upload) (err error) {
	obj := &object{Type: metadataTypeUpload, Object: upload}
	return e.encoder.Encode(obj)
}

func (e *exporter) addFile(file *common.File) (err error) {
	obj := &object{Type: metadataTypeFile, Object: file}
	return e.encoder.Encode(obj)
}

func (e *exporter) addUser(user *common.User) (err error) {
	obj := &object{Type: metadataTypeUser, Object: user}
	return e.encoder.Encode(obj)
}

func (e *exporter) addToken(token *common.Token) (err error) {
	obj := &object{Type: metadataTypeToken, Object: token}
	return e.encoder.Encode(obj)
}

func (e *exporter) addSetting(setting *common.Setting) (err error) {
	obj := &object{Type: metadataTypeSetting, Object: setting}
	return e.encoder.Encode(obj)
}

func (e *exporter) close() (err error) {
	err = e.compressor.Close()
	if err != nil {
		return err
	}
	err = e.writer.Close()
	if err != nil {
		return err
	}
	return nil
}

// Export exports all metadata from the backend to a compressed binary file
func (b *Backend) Export(path string) (err error) {
	e, err := newExporter(path)
	if err != nil {
		return err
	}

	defer func() { _ = e.close() }()

	count := 0
	err = b.ForEachUsers(func(user *common.User) error {
		count++
		return e.addUser(user)
	})
	if err != nil {
		return err
	}
	fmt.Printf("exported %d users\n", count)

	count = 0
	err = b.ForEachToken(func(token *common.Token) error {
		count++
		return e.addToken(token)
	})
	if err != nil {
		return err
	}
	fmt.Printf("exported %d tokens\n", count)

	count = 0
	// Need to export "soft deleted" uploads too else some removed/deleted files will have broken foreign keys
	err = b.ForEachUploadUnscoped(func(upload *common.Upload) error {
		count++
		return e.addUpload(upload)
	})
	if err != nil {
		return err
	}
	fmt.Printf("exported %d uploads\n", count)

	count = 0
	err = b.ForEachFile(func(file *common.File) error {
		count++
		return e.addFile(file)
	})
	if err != nil {
		return err
	}
	fmt.Printf("exported %d files\n", count)

	count = 0
	err = b.ForEachSetting(func(setting *common.Setting) error {
		count++
		return e.addSetting(setting)
	})
	if err != nil {
		return err
	}
	fmt.Printf("exported %d settings\n", count)

	return nil
}
