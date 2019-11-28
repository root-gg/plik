package sdk

import (
	"fmt"
	"io"

	"github.com/root-gg/plik/server/common"
)

// File override from a Plik common.File
// to add some handy getters in SDK
type File struct {
	*common.File
	reader io.Reader
	client *Client
	upload *Upload
}

// URL to download the file
// If remote plik instance has a different download domain set,
// it will take taht one to compute file URL
func (f *File) URL() string {

	if f.client == nil || f.upload == nil {
		return ""
	}

	downloadDomain := f.client.BaseURL.String()
	if f.upload.DownloadDomain != "" {
		downloadDomain = f.upload.DownloadDomain
	}

	return fmt.Sprintf("%s/file/%s/%s/%s", downloadDomain, f.upload.ID, f.ID, f.Name)
}
