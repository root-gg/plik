package file

import (
	"encoding/json"
	"github.com/root-gg/plik/server/utils"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileMetadataBackendConfig struct {
	Directory string
}

func NewFileMetadataBackendConfig(config map[string]interface{}) (this *FileMetadataBackendConfig) {
	this = new(FileMetadataBackendConfig)
	this.Directory = "files"
	utils.Assign(this, config)
	return
}

type FileMetadataBackend struct {
	Config *FileMetadataBackendConfig
}

var locks map[string]*sync.RWMutex

func NewFileMetadataBackend(config map[string]interface{}) (this *FileMetadataBackend) {
	this = new(FileMetadataBackend)
	this.Config = NewFileMetadataBackendConfig(config)
	return
}

func (this *FileMetadataBackend) Create(upload *utils.Upload) (err error) {

	// Get Splice
	splice := upload.Id
	if len(upload.Id) > 2 {
		splice = upload.Id[:2]
	}

	directory := this.Config.Directory + "/" + splice + "/" + upload.Id
	metadatasFile := directory + "/.config"

	// Get json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		return err
	}

	// Create upload dir if not exists
	if _, err := os.Stat(directory); err != nil {
		if err := os.MkdirAll(directory, 0777); err != nil {
			return err
		}

		log.Printf(" - [META] Folder %s successfully created", directory)
	}

	// Open metadatas files
	f, err := os.OpenFile(metadatasFile, os.O_RDWR|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		log.Printf(" - [META] Failed to open metadatas : %s", err)
		return err
	}

	// Print content
	_, err = f.Write(b)
	if err != nil {
		log.Printf(" - [META] Failed to write metadatas : %s", err)
		return err
	}

	// Sync on disk
	err = f.Sync()
	if err != nil {
		log.Printf(" - [META] Failed to sync metadatas : %s", err)
		return err
	}

	log.Printf(" - [META] Metadatas file %s for upload %s successfully writed on disk", metadatasFile, upload.Id)
	return nil
}

func (this *FileMetadataBackend) Get(id string) (upload *utils.Upload, err error) {

	// Get Splice
	splice := id
	if len(id) > 2 {
		splice = id[:2]
	}
	metadatasFile := this.Config.Directory + "/" + splice + "/" + id + "/.config"

	// Open & Read metadatas
	by := make([]byte, 0)
	by, err = ioutil.ReadFile(metadatasFile)
	if err != nil {
		return nil, err
	}

	// Unmarshal
	upload = new(utils.Upload)
	if err := json.Unmarshal(by, upload); err != nil {
		return nil, err
	}

	return upload, nil
}

func (this *FileMetadataBackend) AddOrUpdateFile(upload *utils.Upload, file *utils.File) (err error) {

	// Lock
	Lock(upload.Id)
	defer Unlock(upload.Id)

	// Reload
	upload, err = this.Get(upload.Id)

	// Add file to metadata
	upload.Files[file.Id] = file

	// Get json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		return err
	}

	// Get splice
	splice := upload.Id
	if len(upload.Id) > 2 {
		splice = upload.Id[:2]
	}

	// Create directory if not exist
	directory := this.Config.Directory + "/" + splice + "/" + upload.Id
	metadatas := directory + "/.config"

	if _, err := os.Stat(directory); err != nil {
		if err := os.MkdirAll(directory, 0777); err != nil {
			return err
		}

		log.Printf(" - [META] Folder %s successfully created", directory)
	}

	// Truncate
	err = os.Truncate(metadatas, 0)
	if err != nil {
		log.Printf(" - [META] Failed to truncate metadatas : %s", err)
		return err
	}

	// Open metadatas files
	f, err := os.OpenFile(metadatas, os.O_RDWR|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		log.Printf(" - [META] Failed to open metadatas : %s", err)
		return err
	}

	// Print content
	_, err = f.Write(b)
	if err != nil {
		log.Printf(" - [META] Failed to write metadatas : %s", err)
		return err
	}

	// Sync on disk
	err = f.Sync()
	if err != nil {
		log.Printf(" - [META] Failed to sync metadatas : %s", err)
		return err
	}

	log.Printf(" - [META] Metadatas file %s for upload %s successfully writed on disk", metadatas, upload.Id)

	return nil
}

func (this *FileMetadataBackend) RemoveFile(upload *utils.Upload, file *utils.File) (err error) {

	// Lock
	Lock(upload.Id)
	defer Unlock(upload.Id)

	// Reload
	upload, err = this.Get(upload.Id)

	// Remove file frome metadata
	delete(upload.Files, file.Name)

	// Get json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		return err
	}

	// Get splice
	splice := upload.Id
	if len(upload.Id) > 2 {
		splice = upload.Id[:2]
	}

	// Truncate first
	directory := this.Config.Directory + "/" + splice + "/" + upload.Id
	metadatas := directory + "/.config"

	// Truncate
	err = os.Truncate(metadatas, 0)
	if err != nil {
		log.Printf(" - [META] Failed to truncate metadatas : %s", err)
		return err
	}

	// Open metadatas files
	f, err := os.OpenFile(metadatas, os.O_RDWR, os.FileMode(0666))
	if err != nil {
		log.Printf(" - [META] Failed to open metadatas : %s", err)
		return err
	}

	// Print content
	_, err = f.Write(b)
	if err != nil {
		log.Printf(" - [META] Failed to write metadatas : %s", err)
		return err
	}

	// Sync on disk
	err = f.Sync()
	if err != nil {
		log.Printf(" - [META] Failed to sync metadatas : %s", err)
		return err
	}

	return nil
}

func (this *FileMetadataBackend) Remove(upload *utils.Upload) (err error) {

	// Splice
	splice := upload.Id
	if len(upload.Id) > 2 {
		splice = upload.Id[:2]
	}

	directory := this.Config.Directory + "/" + splice + "/" + upload.Id
	fullPath := directory + "/.config"

	// Remove
	err = os.Remove(fullPath)
	if err != nil {
		return err
	}

	return nil
}

func (this *FileMetadataBackend) GetUploadsToRemove() (ids []string, err error) {

	if utils.Config.MaxTtl > 0 {
		ids = make([]string, 0)

		// Let's call our friend find
		args := make([]string, 0)
		args = append(args, this.Config.Directory)
		args = append(args, "-mindepth", "2")
		args = append(args, "-maxdepth", "2")
		args = append(args, "-type", "d")
		args = append(args, "-cmin", "+"+strconv.Itoa(utils.Config.MaxTtl))

		// Exec
		cmd := exec.Command("find", args...)
		o, err := cmd.Output()
		if err != nil {
			return ids, err
		}

		// Split
		pathsToRemove := strings.Split(string(o), "\n")

		for _, pathToRemove := range pathsToRemove {
			if pathToRemove != "" {
				uploadId := filepath.Base(pathToRemove)
				ids = append(ids, uploadId)
			}
		}
	}

	return ids, nil
}

//
//// Lock for file this (to avoid problems on concurrent access)
//

func Lock(uploadId string) {
	if locks == nil {
		locks = make(map[string]*sync.RWMutex)
	}
	if locks[uploadId] == nil {
		locks[uploadId] = new(sync.RWMutex)

		go func() {
			time.Sleep(time.Hour)
			delete(locks, uploadId)
		}()
	}

	locks[uploadId].Lock()
}

func Unlock(uploadId string) {
	locks[uploadId].Unlock()
}
