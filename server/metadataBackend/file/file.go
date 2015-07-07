/**

    Plik upload server

The MIT License (MIT)

Copyright (c) <2015>
	- Mathieu Bodjikian <mathieu@bodjikian.fr>
	- Charles-Antoine Mathieu <skatkatt@root.gg>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
**/

package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/root-gg/plik/server/common"
)

var (
	locks map[string]*sync.RWMutex
)

// MetadataBackend object
type MetadataBackend struct {
	Config *MetadataBackendConfig
}

// NewFileMetadataBackend instantiate a new File Metadata Backend
// from configuration passed as argument
func NewFileMetadataBackend(config map[string]interface{}) (fmb *MetadataBackend) {
	fmb = new(MetadataBackend)
	fmb.Config = NewFileMetadataBackendConfig(config)
	locks = make(map[string]*sync.RWMutex)
	return
}

// Create implementation for File Metadata Backend
func (fmb *MetadataBackend) Create(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	// Get upload directory
	directory, err := fmb.getDirectoryFromUploadID(upload.ID)
	if err != nil {
		ctx.Warningf("Unable to get upload directory : %s", err)
		return
	}

	// Get metadata file path
	metadataFile := directory + "/.config"

	// Serialize metadata to json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		err = ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Create upload directory if needed
	if _, err = os.Stat(directory); err != nil {
		if err = os.MkdirAll(directory, 0777); err != nil {
			err = ctx.EWarningf("Unable to create upload directory %s : %s", directory, err)
			return
		}
		ctx.Infof("Upload directory %s successfully created", directory)
	}

	// Create metadata file
	f, err := os.OpenFile(metadataFile, os.O_RDWR|os.O_CREATE, os.FileMode(0666))
	if err != nil {
		err = ctx.EWarningf("Unable to create metadata file %s : %s", metadataFile, err)
		return
	}
	defer f.Close()

	// Print content
	_, err = f.Write(b)
	if err != nil {
		err = ctx.EWarningf("Unable to write metadata file %s : %s", metadataFile, err)
		return
	}

	// Sync on disk
	err = f.Sync()
	if err != nil {
		err = ctx.EWarningf("Unable to sync metadata file %s : %s", metadataFile, err)
		return
	}

	ctx.Infof("Metadata file successfully saved %s", metadataFile)
	return
}

// Get implementation for File Metadata Backend
func (fmb *MetadataBackend) Get(ctx *common.PlikContext, id string) (upload *common.Upload, err error) {
	defer ctx.Finalize(err)

	// Get upload directory
	directory, err := fmb.getDirectoryFromUploadID(id)
	if err != nil {
		ctx.Warningf("Unable to get upload directory : %s", err)
		return
	}

	// Get metadata file path
	metadataFile := directory + "/.config"

	// Open and read metadata
	var buffer []byte
	buffer, err = ioutil.ReadFile(metadataFile)
	if err != nil {
		err = ctx.EWarningf("Unable read metadata file %s : %s", metadataFile, err)
		return
	}

	// Unserialize metadata from json
	upload = new(common.Upload)
	if err = json.Unmarshal(buffer, upload); err != nil {
		err = ctx.EWarningf("Unable to unserialize metadata from json \"%s\" : %s", string(buffer), err)
		return
	}

	return
}

// AddOrUpdateFile implementation for File Metadata Backend
func (fmb *MetadataBackend) AddOrUpdateFile(ctx *common.PlikContext, upload *common.Upload, file *common.File) (err error) {
	defer ctx.Finalize(err)

	// avoid race condition
	lock(upload.ID)
	defer unlock(upload.ID)

	// The first thing to do is to reload the file from disk
	upload, err = fmb.Get(ctx.Fork("reload metadata"), upload.ID)

	// Add file metadata to upload metadata
	upload.Files[file.ID] = file

	// Serialize metadata to json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		err = ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Get upload directory
	directory, err := fmb.getDirectoryFromUploadID(upload.ID)
	if err != nil {
		ctx.Warningf("Unable to get upload directory : %s", err)
		return
	}

	// Get metadata file path
	metadataFile := directory + "/.config"

	// Create directory if needed
	if _, err = os.Stat(directory); err != nil {
		if err = os.MkdirAll(directory, 0777); err != nil {
			err = ctx.EWarningf("Unable to create upload directory %s : %s", directory, err)
			return
		}
		ctx.Infof("Upload directory %s successfully created", directory)
	}

	// Override metadata file
	f, err := os.OpenFile(metadataFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0666))
	if err != nil {
		err = ctx.EWarningf("Unable to create metadata file %s : %s", metadataFile, err)
		return
	}

	// Print content
	_, err = f.Write(b)
	if err != nil {
		err = ctx.EWarningf("Unable to write metadata file %s : %s", metadataFile, err)
		return
	}

	// Sync on disk
	err = f.Sync()
	if err != nil {
		err = ctx.EWarningf("Unable to sync metadata file %s : %s", metadataFile, err)
		return
	}

	ctx.Infof("Metadata file successfully updated %s", metadataFile)
	return
}

// RemoveFile implementation for File Metadata Backend
func (fmb *MetadataBackend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, file *common.File) (err error) {
	defer ctx.Finalize(err)

	// avoid race condition
	lock(upload.ID)
	defer unlock(upload.ID)

	// The first thing to do is to reload the file from disk
	upload, err = fmb.Get(ctx.Fork("reload metadata"), upload.ID)

	// Remove file metadata from upload metadata
	delete(upload.Files, file.Name)

	// Serialize metadata to json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		err = ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Get upload directory
	directory, err := fmb.getDirectoryFromUploadID(upload.ID)
	if err != nil {
		ctx.Warningf("Unable to get upload directory : %s", err)
		return
	}

	// Get metadata file path
	metadataFile := directory + "/.config"

	// Override metadata file
	f, err := os.OpenFile(metadataFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0666))
	if err != nil {
		err = ctx.EWarningf("Unable to create metadata file %s : %s", metadataFile, err)
		return
	}

	// Print content
	_, err = f.Write(b)
	if err != nil {
		err = ctx.EWarningf("Unable to write metadata file %s : %s", metadataFile, err)
		return
	}

	// Sync on disk
	err = f.Sync()
	if err != nil {
		err = ctx.EWarningf("Unable to sync metadata file %s : %s", metadataFile, err)
		return
	}

	ctx.Infof("Metadata file successfully updated %s", metadataFile)
	return nil
}

// Remove implementation for File Metadata Backend
func (fmb *MetadataBackend) Remove(ctx *common.PlikContext, upload *common.Upload) (err error) {

	// Get upload directory
	directory, err := fmb.getDirectoryFromUploadID(upload.ID)
	if err != nil {
		ctx.Warningf("Unable to get upload directory : %s", err)
		return
	}

	// Get metadata file path
	metadataFile := directory + "/.config"

	// Test if file exist
	_, err = os.Stat(metadataFile)
	if err != nil {
		ctx.Infof("Metadata file is already deleted")
		return nil
	}

	// Remove all metadata at once
	err = os.Remove(metadataFile)
	if err != nil {
		err = ctx.EWarningf("Unable to remove upload directory %s : %s", metadataFile, err)
		return
	}

	return
}

// GetUploadsToRemove implementation for File Metadata Backend
func (fmb *MetadataBackend) GetUploadsToRemove(ctx *common.PlikContext) (ids []string, err error) {

	// Init ids list
	ids = make([]string, 0)

	// List upload subdirectories in main directory
	subdirectories, err := ioutil.ReadDir(fmb.Config.Directory)
	if err != nil {
		return ids, err
	}

	for _, subDirectory := range subdirectories {

		uploadDirectories, err := ioutil.ReadDir(fmb.Config.Directory + "/" + subDirectory.Name())
		if err != nil {
			return ids, err
		}

		for _, uploadDirectory := range uploadDirectories {

			// Get upload metadata
			upload, err := fmb.Get(ctx, uploadDirectory.Name())
			if err != nil {
				return ids, err
			}

			// If a TTL is set, test if expired or not
			if upload.TTL != 0 && time.Now().Unix() > (upload.Creation+int64(upload.TTL)) {
				ids = append(ids, upload.ID)
			}
		}
	}

	return ids, nil
}

func (fmb *MetadataBackend) getDirectoryFromUploadID(uploadID string) (string, error) {
	// To avoid too many files in the same directory
	// data directory is splitted in two levels the
	// first level is the 2 first chars from the upload id
	// it gives 3844 possibilities reaching 65535 files per
	// directory at ~250.000.000 files uploaded.

	if len(uploadID) < 3 {
		return "", fmt.Errorf("Invalid uploadid %s", uploadID)
	}
	return fmb.Config.Directory + "/" + uploadID[:2] + "/" + uploadID, nil
}

// /!\ There is a race condition to avoid /!\
// If a client add/remove many files of the same upload
// in parallel the associated metadata file
// might be read by many goroutine at the same time,
// then every of them will override the file with
// their own possibly incomplete/invalid version.

func lock(uploadID string) {
	if locks[uploadID] == nil {
		locks[uploadID] = new(sync.RWMutex)
	}
	locks[uploadID].Lock()
}

func unlock(uploadID string) {
	locks[uploadID].Unlock()
	go func() {
		time.Sleep(time.Minute)
		delete(locks, uploadID)
	}()
}
