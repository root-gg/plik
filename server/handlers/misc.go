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

package handlers

import (
	"image/png"
	"net/http"
	"strconv"

	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/boombuler/barcode/qr"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/juliet"
	"github.com/root-gg/plik/server/Godeps/_workspace/src/github.com/root-gg/utils"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/dataBackend"
	"github.com/root-gg/plik/server/metadataBackend"
)

// GetVersionHandler return the build information.
func GetVersionHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Print version and build information in the json response.
	json, err := utils.ToJson(common.GetBuildInfo())
	if err != nil {
		log.Warningf("Unable to serialize json response : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize json response", 500)
		return
	}

	resp.Write(json)
}

// GetConfigurationHandler return the server configuration
func GetConfigurationHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Print configuration in the json response.
	json, err := utils.ToJson(common.Config)
	if err != nil {
		log.Warningf("Unable to serialize response body : %s", err)
		common.Fail(ctx, req, resp, "Unable to serialize response body", 500)
		return
	}
	resp.Write(json)
}

// GetQrCodeHandler return a QRCode for the requested URL
func GetQrCodeHandler(ctx *juliet.Context, resp http.ResponseWriter, req *http.Request) {
	log := common.GetLogger(ctx)

	// Check params
	urlParam := req.FormValue("url")
	sizeParam := req.FormValue("size")

	// Parse int on size
	sizeInt, err := strconv.Atoi(sizeParam)
	if err != nil {
		sizeInt = 250
	}
	if sizeInt > 1000 {
		log.Warning("QRCode size must be lower than 1000")
		common.Fail(ctx, req, resp, "QRCode size must be lower than 1000", 400)
		return
	}

	// Generate QRCode png from url
	qrcode, err := qr.Encode(urlParam, qr.H, qr.Auto)
	if err != nil {
		log.Warningf("Unable to generate QRCode : %s", err)
		common.Fail(ctx, req, resp, "Unable to generate QRCode", 500)
		return
	}

	// Scale QRCode png size
	qrcode, err = barcode.Scale(qrcode, sizeInt, sizeInt)
	if err != nil {
		log.Warningf("Unable to scale QRCode : %s", err)
		common.Fail(ctx, req, resp, "Unable to generate QRCode", 500)
		return
	}

	resp.Header().Add("Content-Type", "image/png")
	err = png.Encode(resp, qrcode)
	if err != nil {
		log.Warningf("Unable to encode png : %s", err)
	}
}

// RemoveUploadIfNoFileAvailable iterates on upload files and remove upload files
// and metadata if all the files have been downloaded (usefull for OneShot uploads)
func RemoveUploadIfNoFileAvailable(ctx *juliet.Context, upload *common.Upload) {
	log := common.GetLogger(ctx)

	// Test if there are remaining files
	filesInUpload := len(upload.Files)
	for _, f := range upload.Files {
		if f.Status != "uploaded" {
			filesInUpload--
		}
	}

	if filesInUpload == 0 {
		log.Debugf("No more files in upload. Removing.")

		if !upload.Stream {
			err := dataBackend.GetDataBackend().RemoveUpload(ctx, upload)
			if err != nil {
				log.Warningf("Unable to remove upload : %s", err)
				return
			}
		}
		err := metadataBackend.GetMetaDataBackend().Remove(ctx, upload)
		if err != nil {
			log.Warningf("Unable to remove upload : %s", err)
			return
		}
	}

	return
}
