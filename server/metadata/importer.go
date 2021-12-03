package metadata

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"

	"github.com/root-gg/utils"

	"github.com/root-gg/plik/server/common"

	"github.com/golang/snappy"
)

// ImportOptions for metadata imports
type ImportOptions struct {
	IgnoreErrors bool
}

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
func (b *Backend) Import(path string, options *ImportOptions) (err error) {
	i, err := newImporter(path)
	if err != nil {
		return err
	}

	defer func() { _ = i.close() }()

	var uploads, files, users, tokens, settings int
	var uploadErrors, fileErrors, userErrors, tokenErrors, settingErrors int

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
				utils.Dump(obj)
				if options.IgnoreErrors {
					fmt.Printf("Unable to load upload : %s\n", err)
				} else {
					return err
				}
				uploadErrors++
			} else {
				uploads++
			}
		case metadataTypeFile:
			err = b.CreateFile(obj.Object.(*common.File))
			if err != nil {
				utils.Dump(obj)
				fmt.Printf("Unable to load file : %s\n", err)
				if !options.IgnoreErrors {
					return err
				}
				fileErrors++
			} else {
				files++
			}
		case metadataTypeUser:
			err = b.CreateUser(obj.Object.(*common.User))
			if err != nil {
				utils.Dump(obj)
				fmt.Printf("Unable to load user : %s\n", err)
				if !options.IgnoreErrors {
					return err
				}
				userErrors++
			} else {
				users++
			}
		case metadataTypeToken:
			err = b.CreateToken(obj.Object.(*common.Token))
			if err != nil {
				utils.Dump(obj)
				fmt.Printf("Unable to load token : %s\n", err)
				if !options.IgnoreErrors {
					return err
				}
				tokenErrors++
			} else {
				tokens++
			}
		case metadataTypeSetting:
			err = b.CreateSetting(obj.Object.(*common.Setting))
			if err != nil {
				utils.Dump(obj)
				fmt.Printf("Unable to load setting : %s\n", err)
				if !options.IgnoreErrors {
					return err
				}
				settingErrors++
			} else {
				settings++
			}
		default:
			return fmt.Errorf("invalid object type")
		}
	}

	fmt.Printf("imported %d out of %d uploads\n", uploads, uploads+uploadErrors)
	fmt.Printf("imported %d out of %d files\n", files, files+fileErrors)
	fmt.Printf("imported %d out of %d users\n", users, users+userErrors)
	fmt.Printf("imported %d out of %d tokens\n", tokens, tokens+tokenErrors)
	fmt.Printf("imported %d out of %d settings\n", settings, settings+settingErrors)

	return nil
}
