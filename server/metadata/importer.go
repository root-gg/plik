package metadata

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/root-gg/plik/server/common"

	"github.com/golang/snappy"
)

type importer struct {
	reader       io.ReadCloser
	decompressor *snappy.Reader
	decoder      *gob.Decoder
}

func newImporter(path string) (i *importer, err error) {
	i = &importer{}

	// Open file for reading
	i.reader, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	// Snappy decompressor
	i.decompressor = snappy.NewReader(i.reader)

	// Gog decoder
	gob.Register(&common.Upload{})
	gob.Register(&common.File{})
	gob.Register(&common.User{})
	gob.Register(&common.Token{})
	gob.Register(&common.Setting{})
	i.decoder = gob.NewDecoder(i.decompressor)

	return i, nil
}

func (i *importer) close() (err error) {
	return i.reader.Close()
}

// Import imports metadata from a compressed binary file
func (b *Backend) Import(path string) (err error) {
	i, err := newImporter(path)
	if err != nil {
		return err
	}

	defer func() { _ = i.close() }()

	var uploads, files, users, tokens, settings int
	for {
		obj := &object{}
		err = i.decoder.Decode(obj)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		switch obj.Type {
		case metadataTypeUpload:
			err = b.CreateUpload(obj.Object.(*common.Upload))
			if err != nil {
				return err
			}
			uploads++
		case metadataTypeFile:
			err = b.CreateFile(obj.Object.(*common.File))
			if err != nil {
				return err
			}
			files++
		case metadataTypeUser:
			err = b.CreateUser(obj.Object.(*common.User))
			if err != nil {
				return err
			}
			users++
		case metadataTypeToken:
			err = b.CreateToken(obj.Object.(*common.Token))
			if err != nil {
				return err
			}
			tokens++
		case metadataTypeSetting:
			err = b.CreateSetting(obj.Object.(*common.Setting))
			if err != nil {
				return err
			}
			settings++
		default:
			return fmt.Errorf("invalid object type")
		}
	}

	fmt.Printf("imported %d uploads\n", uploads)
	fmt.Printf("imported %d files\n", files)
	fmt.Printf("imported %d users\n", users)
	fmt.Printf("imported %d tokens\n", tokens)
	fmt.Printf("imported %d settings\n", settings)

	return nil
}
