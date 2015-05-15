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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
)

type FileMetadataBackendConfig struct {
	Directory string
}

func NewFileMetadataBackendConfig(config map[string]interface{}) (fmb *FileMetadataBackendConfig) {
	fmb = new(FileMetadataBackendConfig)
	// Default upload directory is ./files
	// this is the same as the default file
	// data backend so by default files and
	// metadata are colocated
	fmb.Directory = "files"
	utils.Assign(fmb, config)
	return
}

type FileMetadataBackend struct {
	Config *FileMetadataBackendConfig
}

var locks map[string]*sync.RWMutex

func NewFileMetadataBackend(config map[string]interface{}) (fmb *FileMetadataBackend) {
	fmb = new(FileMetadataBackend)
	fmb.Config = NewFileMetadataBackendConfig(config)
	locks = make(map[string]*sync.RWMutex)
	return
}

func (fmb *FileMetadataBackend) Create(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	// Get metadata file path
	directory := fmb.Config.Directory + "/" + upload.Id[:2] + "/" + upload.Id
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

func (fmb *FileMetadataBackend) Get(ctx *common.PlikContext, id string) (upload *common.Upload, err error) {
	defer ctx.Finalize(err)

	// Get metadata file path
	metadataFile := fmb.Config.Directory + "/" + id[:2] + "/" + id + "/.config"

	// Open and read metadata
	buffer := make([]byte, 0)
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

func (fmb *FileMetadataBackend) AddOrUpdateFile(ctx *common.PlikContext, upload *common.Upload, file *common.File) (err error) {
	defer ctx.Finalize(err)

	// avoid race condition
	Lock(upload.Id)
	defer Unlock(upload.Id)

	// The first thing to do is to reload the file from disk
	upload, err = fmb.Get(ctx.Fork("reload metadata"), upload.Id)

	// Add file metadata to upload metadata
	upload.Files[file.Id] = file

	// Serialize metadata to json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		err = ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Get metadata file path
	directory := fmb.Config.Directory + "/" + upload.Id[:2] + "/" + upload.Id
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

func (fmb *FileMetadataBackend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, file *common.File) (err error) {
	defer ctx.Finalize(err)

	// avoid race condition
	Lock(upload.Id)
	defer Unlock(upload.Id)

	// The first thing to do is to reload the file from disk
	upload, err = fmb.Get(ctx.Fork("reload metadata"), upload.Id)

	// Remove file metadata from upload metadata
	delete(upload.Files, file.Name)

	// Serialize metadata to json
	b, err := json.MarshalIndent(upload, "", "    ")
	if err != nil {
		err = ctx.EWarningf("Unable to serialize metadata to json : %s", err)
		return
	}

	// Get metadata file path
	directory := fmb.Config.Directory + "/" + upload.Id[:2] + "/" + upload.Id
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

func (fmb *FileMetadataBackend) Remove(ctx *common.PlikContext, upload *common.Upload) (err error) {

	// Get metadata file path
	directory := fmb.Config.Directory + "/" + upload.Id[:2] + "/" + upload.Id
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

func (fmb *FileMetadataBackend) GetUploadsToRemove(ctx *common.PlikContext) (ids []string, err error) {
	defer ctx.Finalize(err)

	// Look for uploads older than MaxTTL to schedule them for removal ( defaults to 30 days )
	// This is suboptimal as some of them might have an inferior TTL but it's
	// a lot cheaper than opening and deserializing each metadata file.
	if common.Config.MaxTtl > 0 {
		ids = make([]string, 0)

		// Let's call our friend find
		args := make([]string, 0)
		args = append(args, fmb.Config.Directory)
		args = append(args, "-mindepth", "2") // Remember that the upload directory
		args = append(args, "-maxdepth", "2") // structure is splitted in two
		args = append(args, "-type", "d")
		args = append(args, "-cmin", "+"+strconv.Itoa(common.Config.MaxTtl))
		ctx.Debugf("Executing command : %s", strings.Join(args, " "))

		// Exec find command
		cmd := exec.Command("find", args...)
		var o []byte
		o, err = cmd.Output()
		if err != nil {
			err = ctx.EWarningf("Unable to get find output : %s", err)
			return
		}

		pathsToRemove := strings.Split(string(o), "\n")
		for _, pathToRemove := range pathsToRemove {
			if pathToRemove != "" {
				// Extract upload id from path
				uploadId := filepath.Base(pathToRemove)
				ids = append(ids, uploadId)
			}
		}
	}

	return ids, nil
}

// /!\ There is a race condition to avoid /!\
// If a client add/remove many files of the same upload
// in parallel the associated metadata file
// might be read by many goroutine at the same time,
// then every of them will override the file with
// their own possibly incomplete/invalid version.
func Lock(uploadId string) {
	if locks[uploadId] == nil {
		locks[uploadId] = new(sync.RWMutex)
	}
	locks[uploadId].Lock()
}

func Unlock(uploadId string) {
	locks[uploadId].Unlock()
	go func() {
		time.Sleep(time.Minute)
		delete(locks, uploadId)
	}()
}
