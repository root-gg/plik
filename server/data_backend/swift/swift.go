package swift

import (
	"github.com/ncw/swift"
	"github.com/root-gg/plik/server/utils"
	"io"
	"log"
)

type SwiftBackend struct {
	config     *configInfo
	connection swift.Connection
}

type configInfo struct {
	Username, Password, Host, ProjectName, Container string
}

func NewSwiftBackend(config map[string]interface{}) (this *SwiftBackend) {
	this = new(SwiftBackend)
	this.config = new(configInfo)
	this.config.Container = "PlickData"
	utils.Assign(this.config, config)
	return this
}

func (this *SwiftBackend) auth() (err error) {

	if this.connection.Authenticated() {
		return
	}

	connection := swift.Connection{
		UserName: this.config.Username,
		ApiKey:   this.config.Password,
		AuthUrl:  this.config.Host,
		Tenant:   this.config.ProjectName,
	}

	// Authenticate
	err = connection.Authenticate()
	if err != nil {
		log.Println(err)
		return err
	}
	this.connection = connection

	// Create container
	this.connection.ContainerCreate(this.config.Container, nil)

	return
}

func (this *SwiftBackend) GetFile(upload *utils.Upload, fileId string) (io.ReadCloser, error) {
	err := this.auth()
	if err != nil {
		return nil, err
	}

	log.Printf(" - [FILE] Try to get file %s on upload %s", fileId, upload.Id)
	pipeReader, pipeWriter := io.Pipe()
	uuid := this.getFileId(upload, fileId)
	go func() {
		_, err := this.connection.ObjectGet(this.config.Container, uuid, pipeWriter, true, nil)
		defer pipeWriter.Close()

		if err != nil {
			return
		}

		log.Printf(" - [DONE]")
	}()

	return pipeReader, nil
}

func (this *SwiftBackend) AddFile(upload *utils.Upload, file *utils.File, fileReader io.Reader) (backendDetails map[string]interface{}, err error) {
	err = this.auth()
	if err != nil {
		return backendDetails, err
	}

	uuid := this.getFileId(upload, file.Id)
	log.Println(" - [FILE] Begin upload of file on upload %s", uuid)
	object, err := this.connection.ObjectCreate(this.config.Container, uuid, true, "", "", nil)

	_, err = io.Copy(object, fileReader)
	if err != nil {
		return backendDetails, err
	}
	object.Close()
	log.Printf(" - [FILE] File %s successfully created on swift", uuid)

	return
}

func (this *SwiftBackend) RemoveFile(upload *utils.Upload, fileId string) (err error) {
	err = this.auth()
	if err != nil {
		return
	}

	log.Printf(" - [FILE] Try to delete file %s on upload %s", fileId, upload.Id)
	uuid := this.getFileId(upload, fileId)
	err = this.connection.ObjectDelete(this.config.Container, uuid)
	return
}

func (this *SwiftBackend) RemoveUpload(upload *utils.Upload) (err error) {
	err = this.auth()
	if err != nil {
		return
	}

	for fileId, _ := range upload.Files {
		log.Printf(" - [FILE] Try to delete file %s on upload %s", fileId, upload.Id)
		uuid := this.getFileId(upload, fileId)
		err = this.connection.ObjectDelete(this.config.Container, uuid)
	}

	return
}

func (bf *SwiftBackend) getFileId(upload *utils.Upload, fileId string) string {
	return upload.Id + "." + fileId
}
