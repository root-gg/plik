/**

    Plik test

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

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/root-gg/plik/server/common"
)

var (
	plikURL         = "http://127.0.0.1:8080"
	basicAuth       = ""
	client          = &http.Client{}
	contentToUpload = "PLIK"
	readerForUpload = strings.NewReader(contentToUpload)
	err             error
)

func TestSimpleFileUploadAndGet(t *testing.T) {
	upload := createUpload(&common.Upload{}, t)
	file := uploadFile(upload, "test", readerForUpload, t)

	// We have upload && file ?
	test("getUpload", upload, nil, 200, t)
	test("getFile", upload, file, 200, t)
}

func TestMultipleFilesUploadAndGet(t *testing.T) {
	upload := createUpload(&common.Upload{}, t)

	file1 := uploadFile(upload, "test1", readerForUpload, t)
	file2 := uploadFile(upload, "test2", readerForUpload, t)
	file3 := uploadFile(upload, "test3", readerForUpload, t)
	file4 := uploadFile(upload, "test4", readerForUpload, t)

	// We have upload && files ?
	test("getUpload", upload, nil, 200, t)
	test("getFile", upload, file1, 200, t)
	test("getFile", upload, file2, 200, t)
	test("getFile", upload, file3, 200, t)
	test("getFile", upload, file4, 200, t)
}

func TestNonExistingUpload(t *testing.T) {
	fake := common.NewUpload()
	fake.ID = "f4s6f4sd4f56sd4f64sd6f4s64f6sd4f4s56df4s"
	test("getUpload", fake, nil, 404, t)
}

func TestNonExistingFile(t *testing.T) {
	upload := createUpload(&common.Upload{}, t)
	file := uploadFile(upload, "test", readerForUpload, t)

	// We have upload ?
	test("getUpload", upload, nil, 200, t)

	// Good file id, bad file name
	test("getFile", upload, &common.File{ID: file.ID, Name: "plop"}, 404, t)

	// Bad file id, bad file name
	test("getFile", upload, &common.File{ID: "dfsmlkdsflmks", Name: "plop"}, 404, t)

	// Bad file id, good file name
	test("getFile", upload, &common.File{ID: "dfsmlkdsflmks", Name: file.Name}, 404, t)
}

func TestOneShot(t *testing.T) {
	upload := createUpload(&common.Upload{OneShot: true}, t)
	file := uploadFile(upload, "test", readerForUpload, t)

	test("getFile", upload, file, 200, t)
	test("getFile", upload, file, 404, t)
}

func TestRemovable(t *testing.T) {
	upload := createUpload(&common.Upload{}, t)
	uploadRemovable := createUpload(&common.Upload{Removable: true}, t)

	file := uploadFile(upload, "test", readerForUpload, t)
	fileRemovable := uploadFile(uploadRemovable, "test", readerForUpload, t)

	// Should fail on classic upload
	test("removeFile", upload, file, 401, t)

	// Should work on removable upload
	test("removeFile", uploadRemovable, fileRemovable, 200, t)

	// Test if it worked on removable
	test("getFile", uploadRemovable, fileRemovable, 404, t)
}

func TestBasicAuth(t *testing.T) {
	upload := createUpload(&common.Upload{Login: "plik", Password: "plik"}, t)
	file := uploadFile(upload, "test", readerForUpload, t)

	// Without Authorization header
	savedBasic := basicAuth
	basicAuth = ""
	test("getFile", upload, file, 401, t)

	// With Authorization header
	basicAuth = savedBasic
	test("getFile", upload, file, 200, t)
}

func TestTtl(t *testing.T) {
	upload := createUpload(&common.Upload{TTL: 1}, t)
	file := uploadFile(upload, "test", readerForUpload, t)

	// Should work
	test("getFile", upload, file, 200, t)

	// Sleep until upload expire
	time.Sleep(time.Second)

	// Should fail as the ttl is 1second, and we slept 1,5sec
	test("getFile", upload, file, 404, t)
}

//
//// Subs for creating uploads and uploading files
//

func createUpload(uploadParams *common.Upload, t *testing.T) (upload *common.Upload) {
	var URL *url.URL
	URL, err = url.Parse(plikURL + "/upload")
	if err != nil {
		t.Fatalf("Error parsing url : %s", err)
	}

	var j []byte
	j, err = json.Marshal(uploadParams)
	if err != nil {
		t.Fatalf("Error marshalling json : %s", err)
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), bytes.NewBuffer(j))
	if err != nil {
		t.Fatalf("Error creating request : %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ClientApp", "go_test")
	req.Header.Set("Referer", plikURL)

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Error creating upload : %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body : %s", err)
	}

	basicAuth = resp.Header.Get("Authorization")

	// Parse Json
	upload = new(common.Upload)
	err = json.Unmarshal(body, upload)
	if err != nil {
		t.Fatalf("Error unmarshalling json into upload : %s", err)
	}

	return
}

func uploadFile(uploadInfo *common.Upload, name string, reader *strings.Reader, t *testing.T) (file *common.File) {
	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)

	go func() error {
		part, err := multipartWriter.CreateFormFile("file", name)
		if err != nil {
			t.Fatalf("Error creating multipart form : %s", err)
		}

		_, err = io.Copy(part, reader)
		if err != nil {
			t.Fatalf("Error copying file data to multipart part : %s", err)
		}

		err = multipartWriter.Close()
		return pipeWriter.CloseWithError(err)
	}()

	var URL *url.URL
	URL, err = url.Parse(plikURL + "/upload/" + uploadInfo.ID + "/file")
	if err != nil {
		t.Fatalf("Error parsing url : %s", err)
	}

	var req *http.Request
	req, err = http.NewRequest("POST", URL.String(), pipeReader)
	if err != nil {
		t.Fatalf("Error creating file upload request : %s", err)
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("X-ClientApp", "cli_client")
	req.Header.Set("X-UploadToken", uploadInfo.UploadToken)

	if uploadInfo.ProtectedByPassword {
		req.Header.Set("Authorization", basicAuth)
	}

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Error making file upload request : %s", err)
	}

	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading file upload response body : %s", err)
	}

	// Parse Json
	file = new(common.File)
	err = json.Unmarshal(responseBody, file)
	if err != nil {
		t.Fatalf("Error unmarshalling file upload response : %s", err)
	}

	// Put it in upload infos
	uploadInfo.Files[file.ID] = file

	// Rewind reader
	reader.Seek(0, 0)

	return
}

func getUpload(uploadID string) (httpCode int, upload *common.Upload, err error) {

	var URL *url.URL
	URL, err = url.Parse(plikURL + "/upload/" + uploadID)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "curl")

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	httpCode = resp.StatusCode
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Parse return
	upload = new(common.Upload)
	err = json.Unmarshal(responseBody, upload)
	if err != nil {
		return
	}

	return
}

func getFile(upload *common.Upload, file *common.File) (httpCode int, content string, err error) {

	var URL *url.URL
	URL, err = url.Parse(plikURL + "/file/" + upload.ID + "/" + file.ID + "/" + file.Name)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "curl")

	if upload.ProtectedByPassword {
		req.Header.Set("Authorization", basicAuth)
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	httpCode = resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	content = string(body)

	return
}

func removeFile(upload *common.Upload, file *common.File) (httpCode int, err error) {

	var URL *url.URL
	URL, err = url.Parse(plikURL + "/upload/" + upload.ID + "/file/" + file.ID)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("DELETE", URL.String(), nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "curl")

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	httpCode = resp.StatusCode

	return
}

func test(action string, upload *common.Upload, file *common.File, expectedHTTPCode int, t *testing.T) {

	t.Logf("Try to %s on upload %s. We should get a %d : ", action, upload.ID, expectedHTTPCode)

	switch action {

	case "getUpload":

		code, upload, err := getUpload(upload.ID)
		if err != nil {
			t.Fatalf("Failed to get upload : %s", err)
		}

		// Test code
		if code != expectedHTTPCode {
			t.Fatalf("We got http code %d on action %s on upload %s. We expected %d", code, action, upload.ID, expectedHTTPCode)
		} else {
			t.Logf(" -> Got a %d. Good", code)
		}

	case "getFile":

		code, content, err := getFile(upload, file)
		if err != nil {
			t.Fatalf("Failed to execute action %s on file %s from upload %s : %s", action, file.ID, upload.ID, err)
		}

		// Test code
		if code != expectedHTTPCode {
			t.Fatalf("We got http code %d on action %s on upload %s. We expected %d", code, action, upload.ID, expectedHTTPCode)
		} else {
			t.Logf(" -> Got a %d. Good", code)
		}

		// Test content
		if expectedHTTPCode == 200 {
			if content != contentToUpload {
				t.Fatalf("Did not get expected content (%s) on getting file %s on upload %s. We expected %s", content, file.ID, upload.ID, contentToUpload)
			} else {
				t.Logf(" -> Got content : %s. Good", contentToUpload)
			}
		} else {

			// On a non 200 expected code, it MUST NOT contain file data
			if strings.Contains(content, contentToUpload) {
				t.Fatalf("Warning. Got file content on a 404 upload : %s", content)
			}
		}

	case "removeFile":

		code, err := removeFile(upload, file)
		if err != nil {
			t.Fatalf("Failed to execute action %s on file %s from upload %s : %s", action, file.ID, upload.ID, err)
		}

		if code != expectedHTTPCode {
			t.Fatalf("We got http code %d on action %s on upload %s. We expected %d", code, action, upload.ID, expectedHTTPCode)
		} else {
			t.Logf(" -> Got a %d. Good", code)
		}
	}
}
