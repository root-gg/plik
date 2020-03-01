package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"

	"github.com/root-gg/plik/server/common"
)

var (
	userAgent = "Plik Golang SDK v1.0.0"
)

// Client is the main plik client
// It will allow to intercact with plik API to :
//  - Get/Create/Delete uploads
//  - Add/Remove files to the upload
//  - Get plik server configuration
type Client struct {
	HTTPClient *http.Client
	BaseURL    *url.URL
	serverConf *common.Configuration
	once       *sync.Once
}

// NewClient instantiate a new Plik client from an URL
func NewClient(baseURL string) (c *Client, err error) {

	c = new(Client)
	c.HTTPClient = new(http.Client)
	c.BaseURL, err = url.Parse(baseURL)
	c.once = new(sync.Once)
	if err != nil {
		return
	}

	return
}

// NewUpload creates a new empty upload on Plik
func (c *Client) NewUpload() (upload *Upload) {
	return c.NewUploadWithOptions(&UploadOptions{})
}

// NewUploadWithOptions creates a new empty upload on Plik
// with custom options such as TTL, OneShot, Removable,...
func (c *Client) NewUploadWithOptions(opts *UploadOptions) (upload *Upload) {

	upload = new(Upload)
	upload.PlikUpload = common.NewUpload()
	upload.Files = make(map[string]*File)
	upload.client = c
	upload.TTL = opts.TTL
	upload.Comments = opts.Comments
	upload.Stream = opts.Stream
	upload.OneShot = opts.OneShot
	upload.Removable = opts.Removable
	upload.ProtectedByPassword = opts.ProtectedByPassword
	upload.User = opts.User
	upload.Token = opts.Token
	upload.Login = opts.Login
	upload.Password = opts.Password
	upload.ProtectedByYubikey = opts.ProtectedByYubikey
	upload.Yubikey = opts.Yubikey
	upload.IsAdmin = true

	return
}

// GetUpload will get upload from Plik
func (c *Client) GetUpload(uploadID string) (upload *common.Upload, err error) {

	path := fmt.Sprintf("/upload/%s", uploadID)
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	_, err = c.do(req, &upload)
	if err != nil {
		return nil, err
	}

	return upload, err
}

// ServerConfiguration will get remote plik server configuration
// Configuration contains :
//   - Max file size
//   - Max files per upload
//   - Default TTL
//   - Max TTL
//   - Download domain (if different than main domain)
//
func (c *Client) ServerConfiguration() (conf *common.Configuration, err error) {

	if c.serverConf == nil {
		req, err := c.newRequest("GET", "/config", nil)
		if err != nil {
			return nil, err
		}

		_, err = c.do(req, &c.serverConf)
		if err != nil {
			return nil, err
		}
	}

	return c.serverConf, nil
}

//
// Private subs
//

func (c *Client) newRequest(method, path string, body interface{}) (req *http.Request, err error) {

	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)

	// Encode body if present
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)

		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}

	}

	// Make request
	req, err = http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	return
}
func (c *Client) newFileRequest(method, path string, fileName string, file io.Reader) (req *http.Request, err error) {

	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)

	// Check filename
	if fileName == "" {
		fileName = "file"
	}

	// Create multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	// Make request
	req, err = http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return
}
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var r common.Result
		err = json.NewDecoder(resp.Body).Decode(&r)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("%s : %s", resp.Status, r.Message)

	}

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return nil, err
	}

	return resp, err
}
