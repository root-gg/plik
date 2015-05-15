/**

    Plik upload client

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

package weedfs

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/root-gg/plik/server/common"
)

var (
	client = http.Client{}
)

// Backend object
type Backend struct {
	Config *BackendConfig
}

// NewWeedFsBackend instantiate a new WeedFS Data Backend
// from configuration passed as argument
func NewWeedFsBackend(config map[string]interface{}) (weedFs *Backend) {
	weedFs = new(Backend)
	weedFs.Config = NewWeedFsBackendConfig(config)
	return
}

// GetFile implementation for WeedFS Data Backend
func (weedFs *Backend) GetFile(ctx *common.PlikContext, upload *common.Upload, id string) (reader io.ReadCloser, err error) {
	defer ctx.Finalize(err)

	file := upload.Files[id]

	// Get WeedFS volume from upload metadata
	if file.BackendDetails["WeedFsVolume"] == nil {
		err = ctx.EWarningf("Missing WeedFS volume from backend details")
		return
	}
	weedFsVolume := file.BackendDetails["WeedFsVolume"].(string)

	// Get WeedFS file id from upload metadata
	if file.BackendDetails["WeedFsFileID"] == nil {
		err = ctx.EWarningf("Missing WeedFS file id from backend details")
		return
	}
	WeedFsFileID := file.BackendDetails["WeedFsFileID"].(string)

	// Get WeedFS volume url
	volumeURL, err := weedFs.getvolumeURL(ctx, weedFsVolume)
	if err != nil {
		err = ctx.EWarningf("Unable to get WeedFS volume url %s : %s", weedFsVolume)
		return
	}

	// Get file from WeedFS volume, the response will be
	// piped directly to the client response body
	fileCompleteURL := "http://" + volumeURL + "/" + weedFsVolume + "," + WeedFsFileID
	ctx.Infof("Getting WeedFS file from : %s", fileCompleteURL)
	resp, err := http.Get(fileCompleteURL)
	if err != nil {
		err = ctx.EWarningf("Error while downloading file from WeedFS at %s : %s", fileCompleteURL, err)
		return
	}

	return resp.Body, nil
}

// AddFile implementation for WeedFS Data Backend
func (weedFs *Backend) AddFile(ctx *common.PlikContext, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	defer func() {
		if err != nil {
			ctx.Finalize(err)
		}
	}() // Finalize the context only if error, else let it be finalized by the upload goroutine

	backendDetails = make(map[string]interface{})

	// Request a volume and a new file id from a WeedFS master
	assignURL := weedFs.Config.MasterURL + "/dir/assign?replication=" + weedFs.Config.ReplicationPattern
	ctx.Debugf("Getting volume and file id from WeedFS master at %s", assignURL)

	resp, err := client.Post(assignURL, "", nil)
	if err != nil {
		err = ctx.EWarningf("Error while getting id from WeedFS master at %s : %s", assignURL, err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response body from WeedFS master at %s : %s", assignURL, err)
		return
	}

	// Unserialize response body
	responseMap := make(map[string]interface{})
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		err = ctx.EWarningf("Unable to unserialize json response \"%s\" from WeedFS master at %s : %s", bodyStr, assignURL, err)
		return
	}

	if responseMap["fid"] != nil && responseMap["fid"].(string) != "" {
		splitVolumeFromID := strings.Split(responseMap["fid"].(string), ",")
		if len(splitVolumeFromID) > 1 {
			backendDetails["WeedFsVolume"] = splitVolumeFromID[0]
			backendDetails["WeedFsFileID"] = splitVolumeFromID[1]
		} else {
			err = ctx.EWarningf("Invalid fid from WeedFS master response \"%s\" at %s", bodyStr, assignURL)
			return
		}
	} else {
		err = ctx.EWarningf("Missing fid from WeedFS master response \"%s\" at %", bodyStr, assignURL)
		return
	}

	// Construct upload url
	if responseMap["publicUrl"] == nil || responseMap["publicUrl"].(string) == "" {
		err = ctx.EWarningf("Missing publicUrl from WeedFS master response \"%s\" at %s", bodyStr, assignURL)
		return
	}
	fileURL := "http://" + responseMap["publicUrl"].(string) + "/" + responseMap["fid"].(string)
	var URL *url.URL
	URL, err = url.Parse(fileURL)
	if err != nil {
		err = ctx.EWarningf("Unable to construct WeedFS upload url \"%s\"", fileURL)
		return
	}

	ctx.Infof("Uploading file %s to volume %s to WeedFS at %s", backendDetails["WeedFsFileID"], backendDetails["WeedFsVolume"], fileURL)

	// Pipe the uploaded file from the client request body
	// to the WeedFS request body without buffering
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	go func() {
		defer ctx.Finalize(err)
		filePart, err := multipartWriter.CreateFormFile("file", file.Name)
		if err != nil {
			ctx.Warningf("Unable to create multipart form : %s", err)
			return
		}

		_, err = io.Copy(filePart, fileReader)
		if err != nil {
			ctx.Warningf("Unable to copy file to WeedFS request body : %s", err)
			pipeWriter.CloseWithError(err)
			return
		}

		err = multipartWriter.Close()
		if err != nil {
			ctx.Warningf("Unable to close multipartWriter : %s", err)
		}
		pipeWriter.CloseWithError(err)
	}()

	// Upload file to WeedFS volume
	req, err := http.NewRequest("PUT", URL.String(), pipeReader)
	if err != nil {
		err = ctx.EWarningf("Unable to create PUT request to %s : %s", URL.String(), err)
		return
	}
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	resp, err = client.Do(req)
	if err != nil {
		err = ctx.EWarningf("Unable to upload file to WeedFS at %s : %s", URL.String(), err)
		return
	}
	defer resp.Body.Close()

	return
}

// RemoveFile implementation for WeedFS Data Backend
func (weedFs *Backend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, id string) (err error) {
	defer ctx.Finalize(err)

	// Get file metadata
	file := upload.Files[id]

	// Get WeedFS volume and file id from upload metadata
	if file.BackendDetails["WeedFsVolume"] == nil {
		err = ctx.EWarningf("Missing WeedFS volume from backend details")
		return
	}
	weedFsVolume := file.BackendDetails["WeedFsVolume"].(string)

	if file.BackendDetails["WeedFsFileID"] == nil {
		err = ctx.EWarningf("Missing WeedFS file id from backend details")
		return
	}
	WeedFsFileID := file.BackendDetails["WeedFsFileID"].(string)

	// Get the WeedFS volume url
	volumeURL, err := weedFs.getvolumeURL(ctx, weedFsVolume)
	if err != nil {
		return
	}

	// Construct Url
	fileURL := "http://" + volumeURL + "/" + weedFsVolume + "," + WeedFsFileID
	var URL *url.URL
	URL, err = url.Parse(fileURL)
	if err != nil {
		err = ctx.EWarningf("Unable to construct WeedFS url \"%s\"", fileURL)
		return
	}

	ctx.Infof("Removing file %s from WeedFS volume %s at %s", WeedFsFileID, weedFsVolume, fileURL)

	// Remove file from WeedFS volume
	req, err := http.NewRequest("DELETE", URL.String(), nil)
	if err != nil {
		err = ctx.EWarningf("Unable to create DELETE request to %s : %s", URL.String(), err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		err = ctx.EWarningf("Unable to delete file from WeedFS volume at %s : %s", URL.String(), err)
		return
	}
	resp.Body.Close()

	return
}

// RemoveUpload implementation for WeedFS Data Backend
// Iterates on every file and call RemoveFile
func (weedFs *Backend) RemoveUpload(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	for fileID := range upload.Files {
		err = weedFs.RemoveFile(ctx.Fork("remove file"), upload, fileID)
		if err != nil {
			return
		}
	}

	return nil
}

func (weedFs *Backend) getvolumeURL(ctx *common.PlikContext, volumeID string) (URL string, err error) {
	timer := ctx.Time("get volume url")
	defer timer.Stop()

	// Ask a WeedFS master the volume urls
	URL = weedFs.Config.MasterURL + "/dir/lookup?volumeId=" + volumeID
	resp, err := client.Post(URL, "", nil)
	if err != nil {
		err = ctx.EWarningf("Unable to get volume %s url from WeedFS master at %s : %s", volumeID, URL, err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response from WeedFS master at %s : %s", volumeID, URL, err)
		return
	}

	// Unserialize response body
	responseMap := make(map[string]interface{})
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		err = ctx.EWarningf("Unable to unserialize json response \"%s\"from WeedFS master at %s : %s", bodyStr, URL, err)
		return
	}

	// As volumes can be replicated there may be more than one
	// available url for a given volume
	var urlsFound []string
	if responseMap["locations"] == nil {
		err = ctx.EWarningf("Missing url from WeedFS master response \"%s\" at %s", bodyStr, URL)
		return
	}
	if locationsArray, ok := responseMap["locations"].([]interface{}); ok {
		for _, location := range locationsArray {
			if locationInfos, ok := location.(map[string]interface{}); ok {
				if locationInfos["publicUrl"] != nil {
					if foundURL, ok := locationInfos["publicUrl"].(string); ok {
						urlsFound = append(urlsFound, foundURL)
					}
				}
			}
		}
	}
	if len(urlsFound) == 0 {
		err = ctx.EWarningf("No url found for WeedFS volume %s", volumeID)
		return
	}

	// Take a random url from the list
	URL = urlsFound[rand.Intn(len(urlsFound))]
	return
}
