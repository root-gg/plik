package plik

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"

	"github.com/root-gg/plik/server/common"
)

// Client manage the process of communicating with a Plik server via the HTTP API
type Client struct {
	*UploadParams // Default upload params for the Client. Those can be overridden per upload

	Debug bool // Display HTTP request and response and other helpful debug data

	URL           string // URL of the Plik server
	ClientName    string // X-ClientApp HTTP Header setting
	ClientVersion string // X-ClientVersion HTTP Header setting

	HTTPClient *http.Client // HTTP Client ot use to make the requests
}

// NewClient creates a new Plik Client
func NewClient(url string) (c *Client) {
	c = &Client{}

	// Default upload params
	c.UploadParams = &UploadParams{}
	c.URL = url

	// Default values for X-ClientApp and X-ClientVersion HTTP Headers
	c.ClientName = "plik_client"
	c.ClientVersion = runtime.GOOS + "-" + runtime.GOARCH + "-" + common.GetBuildInfo().Version

	c.HTTPClient = NewHTTPClient(false)

	return c
}

// Insecure mode does not verify the server's certificate chain and host name
func (c *Client) Insecure() {
	c.HTTPClient = NewHTTPClient(true)
}

// NewUpload create a new Upload object with the client default upload params
func (c *Client) NewUpload() *Upload {
	return newUpload(c)
}

// UploadFile is a handy wrapper to upload a file from the filesystem
func (c *Client) UploadFile(path string) (upload *Upload, file *File, err error) {
	upload = c.NewUpload()

	file, err = upload.AddFileFromPath(path)
	if err != nil {
		return nil, nil, err
	}

	// Create upload and upload the file
	err = upload.Upload()
	if err != nil {
		// Return the upload and file to get a chance to get the file error
		return upload, file, err
	}

	return upload, file, nil
}

// UploadReader is a handy wrapper to upload a single arbitrary data stream
func (c *Client) UploadReader(name string, reader io.Reader) (upload *Upload, file *File, err error) {
	upload = c.NewUpload()

	file = upload.AddFileFromReader(name, reader)

	// Create upload and upload the file
	err = upload.Upload()
	if err != nil {
		// Return the upload and file to get a chance to get the file error
		return upload, file, err
	}

	return upload, file, nil
}

// GetServerVersion return the remote server version
func (c *Client) GetServerVersion() (bi *common.BuildInfo, err error) {
	var req *http.Request
	req, err = http.NewRequest("GET", c.URL+"/version", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse json response
	bi = &common.BuildInfo{}
	err = json.Unmarshal(body, bi)
	if err != nil {
		return nil, err
	}

	return bi, nil
}

// GetServerConfig return the remote server configuration
func (c *Client) GetServerConfig() (config *common.Configuration, err error) {
	var req *http.Request
	req, err = http.NewRequest("GET", c.URL+"/config", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse json response
	config = &common.Configuration{}
	err = json.Unmarshal(body, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetUpload fetch upload metadata from the server
func (c *Client) GetUpload(id string) (upload *Upload, err error) {
	return c.GetUploadProtectedByPassword(id, c.Login, c.Password)
}

// GetUploadProtectedByPassword fetch upload metadata from the server with login and password
func (c *Client) GetUploadProtectedByPassword(id string, login string, password string) (upload *Upload, err error) {
	uploadParams := c.NewUpload().getParams()
	uploadParams.ID = id
	uploadParams.Login = login
	uploadParams.Password = password

	upload, err = c.getUploadWithParams(uploadParams)
	if err != nil {
		return nil, err
	}

	upload.Login = login
	upload.Password = password

	return upload, nil
}

// NewHTTPClient Create a new HTTP client with ProxyFromEnvironment and InsecureSkipVerify setup
func NewHTTPClient(insecure bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		},
	}
}
