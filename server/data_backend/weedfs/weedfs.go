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
	"github.com/root-gg/utils"
)

var client = http.Client{}

type WeedFsBackendConfig struct {
	MasterUrl          string
	ReplicationPattern string
}

func NewWeedFsBackendConfig(config map[string]interface{}) (weedFs *WeedFsBackendConfig) {
	weedFs = new(WeedFsBackendConfig)
	weedFs.MasterUrl = "http://127.0.0.1:9333"
	weedFs.ReplicationPattern = "000"
	utils.Assign(weedFs, config)
	return
}

type WeedFsBackend struct {
	Config *WeedFsBackendConfig
}

func NewWeedFsBackend(config map[string]interface{}) (weedFs *WeedFsBackend) {
	weedFs = new(WeedFsBackend)
	weedFs.Config = NewWeedFsBackendConfig(config)
	return
}

func (weedFs *WeedFsBackend) GetFile(ctx *common.PlikContext, upload *common.Upload, id string) (reader io.ReadCloser, err error) {
	defer ctx.Finalize(err)

	file := upload.Files[id]

	// Get WeedFS volume from upload metadata
	if file.BackendDetails["WeedFsVolume"] == nil {
		err = ctx.EWarningf("Missing WeedFS volume from backend details")
		return
	}
	weedFsVolume := file.BackendDetails["WeedFsVolume"].(string)

	// Get WeedFS file id from upload metadata
	if file.BackendDetails["WeedFsFileId"] == nil {
		err = ctx.EWarningf("Missing WeedFS file id from backend details")
		return
	}
	weedFsFileId := file.BackendDetails["WeedFsFileId"].(string)

	// Get WeedFS volume url
	volumeUrl, err := weedFs.getVolumeUrl(ctx, weedFsVolume)
	if err != nil {
		err = ctx.EWarningf("Unable to get WeedFS volume url %s : %s", weedFsVolume)
		return
	}

	// Get file from WeedFS volume, the response will be
	// piped directly to the client response body
	fileCompleteUrl := "http://" + volumeUrl + "/" + weedFsVolume + "," + weedFsFileId
	ctx.Infof("Getting WeedFS file from : %s", fileCompleteUrl)
	resp, err := http.Get(fileCompleteUrl)
	if err != nil {
		err = ctx.EWarningf("Error while downloading file from WeedFS at %s : %s", fileCompleteUrl, err)
		return
	}

	return resp.Body, nil
}

func (weedFs *WeedFsBackend) AddFile(ctx *common.PlikContext, upload *common.Upload, file *common.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	defer func() {
		if err != nil {
			ctx.Finalize(err)
		}
	}() // Finalize the context only if error, else let it be finalized by the upload goroutine

	backendDetails = make(map[string]interface{})

	// Request a volume and a new file id from a WeedFS master
	assignUrl := weedFs.Config.MasterUrl + "/dir/assign?replication=" + weedFs.Config.ReplicationPattern
	ctx.Debugf("Getting volume and file id from WeedFS master at %s", assignUrl)

	resp, err := client.Post(assignUrl, "", nil)
	if err != nil {
		err = ctx.EWarningf("Error while getting id from WeedFS master at %s : %s", assignUrl, err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response body from WeedFS master at %s : %s", assignUrl, err)
		return
	}

	// Unserialize response body
	responseMap := make(map[string]interface{})
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		err = ctx.EWarningf("Unable to unserialize json response \"%s\" from WeedFS master at %s : %s", bodyStr, assignUrl, err)
		return
	}

	if responseMap["fid"] != nil && responseMap["fid"].(string) != "" {
		splitVolumeFromId := strings.Split(responseMap["fid"].(string), ",")
		if len(splitVolumeFromId) > 1 {
			backendDetails["WeedFsVolume"] = splitVolumeFromId[0]
			backendDetails["WeedFsFileId"] = splitVolumeFromId[1]
		} else {
			err = ctx.EWarningf("Invalid fid from WeedFS master response \"%s\" at %s", bodyStr, assignUrl)
			return
		}
	} else {
		err = ctx.EWarningf("Missing fid from WeedFS master response \"%s\" at %", bodyStr, assignUrl)
		return
	}

	// Construct upload url
	if responseMap["publicUrl"] == nil || responseMap["publicUrl"].(string) == "" {
		err = ctx.EWarningf("Missing publicUrl from WeedFS master response \"%s\" at %s", bodyStr, assignUrl)
		return
	}
	fileUrl := "http://" + responseMap["publicUrl"].(string) + "/" + responseMap["fid"].(string)
	var Url *url.URL
	Url, err = url.Parse(fileUrl)
	if err != nil {
		err = ctx.EWarningf("Unable to construct WeedFS upload url \"%s\"", fileUrl)
		return
	}

	ctx.Infof("Uploading file %s to volume %s to WeedFS at %s", backendDetails["WeedFsFileId"], backendDetails["WeedFsVolume"], fileUrl)

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
	req, err := http.NewRequest("PUT", Url.String(), pipeReader)
	if err != nil {
		err = ctx.EWarningf("Unable to create PUT request to %s : %s", Url.String(), err)
		return
	}
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	resp, err = client.Do(req)
	if err != nil {
		err = ctx.EWarningf("Unable to upload file to WeedFS at %s : %s", Url.String(), err)
		return
	}
	defer resp.Body.Close()

	return
}

func (weedFs *WeedFsBackend) RemoveFile(ctx *common.PlikContext, upload *common.Upload, id string) (err error) {
	defer ctx.Finalize(err)

	// Get file metadata
	file := upload.Files[id]

	// Get WeedFS volume and file id from upload metadata
	if file.BackendDetails["WeedFsVolume"] == nil {
		err = ctx.EWarningf("Missing WeedFS volume from backend details")
		return
	}
	weedFsVolume := file.BackendDetails["WeedFsVolume"].(string)

	if file.BackendDetails["WeedFsFileId"] == nil {
		err = ctx.EWarningf("Missing WeedFS file id from backend details")
		return
	}
	weedFsFileId := file.BackendDetails["WeedFsFileId"].(string)

	// Get the WeedFS volume url
	volumeUrl, err := weedFs.getVolumeUrl(ctx, weedFsVolume)
	if err != nil {
		return
	}

	// Construct Url
	fileUrl := "http://" + volumeUrl + "/" + weedFsVolume + "," + weedFsFileId
	var Url *url.URL
	Url, err = url.Parse(fileUrl)
	if err != nil {
		err = ctx.EWarningf("Unable to construct WeedFS url \"%s\"", fileUrl)
		return
	}

	ctx.Infof("Removing file %s from WeedFS volume %s at %s", weedFsFileId, weedFsVolume, fileUrl)

	// Remove file from WeedFS volume
	req, err := http.NewRequest("DELETE", Url.String(), nil)
	if err != nil {
		err = ctx.EWarningf("Unable to create DELETE request to %s : %s", Url.String(), err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		err = ctx.EWarningf("Unable to delete file from WeedFS volume at %s : %s", Url.String(), err)
		return
	}
	resp.Body.Close()

	return
}

func (weedFs *WeedFsBackend) RemoveUpload(ctx *common.PlikContext, upload *common.Upload) (err error) {
	defer ctx.Finalize(err)

	for fileId := range upload.Files {
		err = weedFs.RemoveFile(ctx.Fork("remove file"), upload, fileId)
		if err != nil {
			return
		}
	}

	return nil
}

func (weedFs *WeedFsBackend) getVolumeUrl(ctx *common.PlikContext, volumeId string) (url string, err error) {
	timer := ctx.Time("get volume url")
	defer timer.Stop()

	// Ask a WeedFS master the volume urls
	url = weedFs.Config.MasterUrl + "/dir/lookup?volumeId=" + volumeId
	resp, err := client.Post(url, "", nil)
	if err != nil {
		err = ctx.EWarningf("Unable to get volume %s url from WeedFS master at %s : %s", volumeId, url, err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = ctx.EWarningf("Unable to read response from WeedFS master at %s : %s", volumeId, url, err)
		return
	}

	// Unserialize response body
	responseMap := make(map[string]interface{})
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		err = ctx.EWarningf("Unable to unserialize json response \"%s\"from WeedFS master at %s : %s", bodyStr, url, err)
		return
	}

	// As volumes can be replicated there may be more than one
	// available url for a given volume
	urlsFound := make([]string, 0)
	if responseMap["locations"] == nil {
		err = ctx.EWarningf("Missing url from WeedFS master response \"%s\" at %s", bodyStr, url)
		return
	}
	if locationsArray, ok := responseMap["locations"].([]interface{}); ok {
		for _, location := range locationsArray {
			if locationInfos, ok := location.(map[string]interface{}); ok {
				if locationInfos["publicUrl"] != nil {
					if foundUrl, ok := locationInfos["publicUrl"].(string); ok {
						urlsFound = append(urlsFound, foundUrl)
					}
				}
			}
		}
	}
	if len(urlsFound) == 0 {
		err = ctx.EWarningf("No url found for WeedFS volume %s", volumeId)
		return
	}

	// Take a random url from the list
	url = urlsFound[rand.Intn(len(urlsFound))]
	return
}
