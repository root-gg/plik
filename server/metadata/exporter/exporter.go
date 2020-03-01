package exporter

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
)

type object struct {
	Type   metadataType
	Object interface{}
}

// Exporter to export metadata to import them in Plik 1.3 format
type Exporter struct {
	writer                        io.WriteCloser
	compressor                    *snappy.Writer
	encoder                       *gob.Encoder
	uploads, files, users, tokens int
}

// NewExporter creates a new Exporter
func NewExporter(path string) (e *Exporter, err error) {
	e = &Exporter{}

	// Open file for writing
	e.writer, err = os.Create(path)
	if err != nil {
		return nil, err
	}

	// Snappy compressor
	e.compressor = snappy.NewBufferedWriter(e.writer)

	// Gob encoder
	gob.RegisterName("*common.Upload", &Upload{})
	gob.RegisterName("*common.File", &File{})
	gob.RegisterName("*common.User", &User{})
	gob.RegisterName("*common.Token", &Token{})

	e.encoder = gob.NewEncoder(e.compressor)

	return e, nil
}

// AddUpload add an upload and it's files to the export
func (e *Exporter) AddUpload(upload *common.Upload) (err error) {
	u, err := AdaptUpload(upload)
	if err != nil {
		return err
	}

	files := u.Files
	u.Files = nil

	obj := &object{Type: metadataTypeUpload, Object: u}
	err = e.encoder.Encode(obj)
	if err != nil {
		return err
	}
	e.uploads++

	for _, file := range files {
		obj := &object{Type: metadataTypeFile, Object: file}
		err = e.encoder.Encode(obj)
		if err != nil {
			return err
		}
		e.files++
	}

	return nil
}

// AddUser add a user and it's token to the export
func (e *Exporter) AddUser(user *common.User) (err error) {
	u, err := AdaptUser(user)
	if err != nil {
		return err
	}

	tokens := u.Tokens
	u.Tokens = nil

	obj := &object{Type: metadataTypeUser, Object: u}
	err = e.encoder.Encode(obj)
	if err != nil {
		return err
	}
	e.users++

	for _, token := range tokens {
		obj := &object{Type: metadataTypeToken, Object: token}
		err = e.encoder.Encode(obj)
		if err != nil {
			return err
		}
		e.tokens++
	}

	return nil
}

// Close finalize the export
func (e *Exporter) Close() (err error) {
	err = e.compressor.Close()
	if err != nil {
		return err
	}
	err = e.writer.Close()
	if err != nil {
		return err
	}

	fmt.Printf("%d uploads exported\n", e.uploads)
	fmt.Printf("%d files exported\n", e.files)
	fmt.Printf("%d users exported\n", e.users)
	fmt.Printf("%d tokens exported\n", e.tokens)

	return nil
}
