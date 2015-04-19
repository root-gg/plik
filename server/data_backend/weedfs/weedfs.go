package weedfs

import (
	"encoding/json"
	"errors"
	"github.com/root-gg/plik/server/utils"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

var client = http.Client{}

type WeedFsBackendConfig struct {
	MasterUrl          string
	ReplicationPattern string
}

func NewWeedFsBackendConfig(config map[string]interface{}) (this *WeedFsBackendConfig) {
	this = new(WeedFsBackendConfig)
	this.MasterUrl = "http://127.0.0.1:9333"
	this.ReplicationPattern = "000"
	utils.Assign(this, config)
	return
}

type WeedFsBackend struct {
	Config *WeedFsBackendConfig
}

func NewWeedFsBackend(config map[string]interface{}) (this *WeedFsBackend) {
	this = new(WeedFsBackend)
	this.Config = NewWeedFsBackendConfig(config)
	return
}

func (this *WeedFsBackend) GetFile(upload *utils.Upload, id string) (io.ReadCloser, error) {

	// Get file on upload
	file := upload.Files[id]

	// Get weed fs volume and id
	if file.BackendDetails["WeedFsVolume"] == nil || file.BackendDetails["WeedFsFileId"] == nil {
		return nil, errors.New("Missing WeedFS volume or fileId in file metadatas")
	}

	weedFsVolume := file.BackendDetails["WeedFsVolume"].(string)
	weedFsFileId := file.BackendDetails["WeedFsFileId"].(string)

	// Get url of volume
	volumeUrl, err := this.getVolumeUrl(weedFsVolume)
	if err != nil {
		return nil, err
	}

	// Get file
	fileCompleteUrl := "http://" + volumeUrl + "/" + weedFsVolume + "," + weedFsFileId
	log.Printf(" - [FILE] WeedFS file url : %s", fileCompleteUrl)
	resp, err := http.Get(fileCompleteUrl)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (this *WeedFsBackend) AddFile(upload *utils.Upload, file *utils.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {

	// Init backend details map
	backendDetails = make(map[string]interface{})

	// Make request to master to get an id
	assignUrl := this.Config.MasterUrl + "/dir/assign?replication=" + this.Config.ReplicationPattern
	resp, err := client.Post(assignUrl, "", nil)
	if err != nil {
		return
	}

	// Misc
	log.Printf(" - [FILE] Calling %s to get upload id", assignUrl)
	defer resp.Body.Close()

	// Get body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Decode it
	responseMap := make(map[string]interface{})
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		return
	}

	// Got an id ?
	if responseMap["fid"] != nil && responseMap["fid"].(string) != "" {
		splitVolumeFromId := strings.Split(responseMap["fid"].(string), ",")
		if len(splitVolumeFromId) > 1 {
			backendDetails["WeedFsVolume"] = splitVolumeFromId[0]
			backendDetails["WeedFsFileId"] = splitVolumeFromId[1]
		}
	}

	// Begin upload
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	go func() {
		filePart, err := multipartWriter.CreateFormFile("file", file.Name)
		if err != nil {
			return
		}

		_, err = io.Copy(filePart, fileReader)
		if err != nil {
			pipeWriter.CloseWithError(err)
			return
		}

		err = multipartWriter.Close()
		pipeWriter.CloseWithError(err)
	}()

	// Construct Url
	var Url *url.URL
	Url, err = url.Parse("http://" + responseMap["publicUrl"].(string) + "/" + responseMap["fid"].(string))
	if err != nil {
		return
	}

	// Construct request
	log.Printf(" - [FILE] Gonna PUT on %s", Url.String())
	req, err := http.NewRequest("PUT", Url.String(), pipeReader)
	if err != nil {
		return
	}

	// Exec
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	resp, err = client.Do(req)
	if err != nil {
		return
	}

	resp.Body.Close()

	return backendDetails, nil
}

func (this *WeedFsBackend) RemoveFile(upload *utils.Upload, id string) error {

	// Get file on upload
	file := upload.Files[id]

	// Get weed fs volume and id
	if file.BackendDetails["WeedFsVolume"] == nil || file.BackendDetails["WeedFsFileId"] == nil {
		return errors.New("Missing WeedFS volume or fileId in file metadatas")
	}

	weedFsVolume := file.BackendDetails["WeedFsVolume"].(string)
	weedFsFileId := file.BackendDetails["WeedFsFileId"].(string)

	// Get url of volume
	volumeUrl, err := this.getVolumeUrl(weedFsVolume)
	if err != nil {
		return err
	}

	// Construct Url
	var Url *url.URL
	Url, err = url.Parse("http://" + volumeUrl + "/" + weedFsVolume + "," + weedFsFileId)
	if err != nil {
		return err
	}

	// Construct request
	log.Printf(" - [FILE] Gonna DELETE on %s", Url.String())
	req, err := http.NewRequest("DELETE", Url.String(), nil)
	if err != nil {
		return err
	}

	// Exec
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return nil
}

func (this *WeedFsBackend) RemoveUpload(upload *utils.Upload) (err error) {

	for fileId, _ := range upload.Files {
		err = this.RemoveFile(upload, fileId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *WeedFsBackend) getVolumeUrl(volumeId string) (url string, err error) {

	// Get url of volume
	log.Printf(" - [FILE] Trying to get WeedFs url for volume id %s", volumeId)
	resp, err := client.Post(this.Config.MasterUrl+"/dir/lookup?volumeId="+volumeId, "", nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// Get body
	bodyStr, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Decode it
	responseMap := make(map[string]interface{})
	err = json.Unmarshal(bodyStr, &responseMap)
	if err != nil {
		return "", err
	}

	// Try to get urls
	urlsFound := make([]string, 0)
	if responseMap["locations"] == nil {
		return "", errors.New("Failed to get location of WeedFs volume.")
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
		err = errors.New("No locations found on WeedFS for volume " + volumeId)
	}

	return urlsFound[rand.Intn(len(urlsFound))], nil
}
