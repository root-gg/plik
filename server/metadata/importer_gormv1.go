package metadata

import (
	"encoding/gob"
	"fmt"
	"github.com/golang/snappy"
	"github.com/root-gg/plik/server/common"
	"io"
	"os"
)

type importerGormV1 struct {
	reader       io.ReadCloser
	decompressor *snappy.Reader
	decoder      *gob.Decoder
}

func newImporterGormV1(path string) (i *importerGormV1, err error) {
	i = &importerGormV1{}

	// Open file for reading
	i.reader, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	// Snappy decompressor
	i.decompressor = snappy.NewReader(i.reader)

	// Gog decoder
	gob.Register(&common.UploadGormV1{})
	gob.Register(&common.FileGormV1{})
	gob.Register(&common.User{})
	gob.Register(&common.Token{})
	gob.Register(&common.Setting{})
	i.decoder = gob.NewDecoder(i.decompressor)

	return i, nil
}

func (i *importerGormV1) close() (err error) {
	return i.reader.Close()
}

func (i *importerGormV1) export(path string) (err error) {
	e, err := newExporter(path)
	if err != nil {
		return err
	}
	defer func() { _ = e.close() }()

	var uploads, files, users, tokens, settings int
	var uploadErrors, fileErrors, userErrors, tokenErrors, settingErrors int

	for {
		obj := &object{}
		err = i.decoder.Decode(obj)
		if err == io.EOF {
			break
		} else if err != nil {
			if err.Error() == "gob: wrong type (gorm.DeletedAt) for received field Upload.DeletedAt" {

			}
			return err
		}

		switch obj.Type {
		case metadataTypeUpload:
			err := e.addUpload(obj.Object.(*common.UploadGormV1).ToUpload())
			if err != nil {
				return err
			}
			
			if err != nil {
				return err
			} else {
				uploads++
			}
		case metadataTypeFile:
			err := e.addFile(obj.Object.(*common.FileGormV1).ToFile())
			if err != nil {
				return err
			}

			if err != nil {
				return err
			} else {
				files++
			}
		case metadataTypeUser:
			err := e.addUser(obj.Object.(*common.User))
			if err != nil {
				return err
			}

			if err != nil {
				return err
			} else {
				users++
			}
		case metadataTypeToken:
			err := e.addToken(obj.Object.(*common.Token))
			if err != nil {
				return err
			}

			if err != nil {
				return err
			} else {
				uploads++
			}
		case metadataTypeSetting:
			err := e.addSetting(obj.Object.(*common.Setting))
			if err != nil {
				return err
			}

			if err != nil {
				return err
			} else {
				uploads++
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

	fmt.Printf("Updated export format at %s\n", path)
	return nil
}

func fixGormV1ExportFormat(path string) (newPath string, err error) {
	i, err := newImporter(path)
	if err != nil {
		return "", err
	}

	defer func() { _ = i.close() }()

	needFix := false
	for {
		obj := &object{}
		err = i.decoder.Decode(obj)
		if err == io.EOF {
			break
		} else if err != nil {
			if err.Error() == "gob: wrong type (gorm.DeletedAt) for received field Upload.DeletedAt" {
				needFix = true
				break
			} else {
				return "", err
			}
		}
	}

	if !needFix {
		return path, nil
	}

	fmt.Println("Fixing GormV1 export")

	iGormV1, err := newImporterGormV1(path)
	if err != nil {
		return "", err
	}

	newPath = path + ".fixed"
	err = iGormV1.export(newPath)
	if err != nil {
		return "", err
	}

	return newPath, nil
}