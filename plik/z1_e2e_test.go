package plik

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/data"
	fileBackend "github.com/root-gg/plik/server/data/file"
	"github.com/root-gg/plik/server/server"
)

func TestUploadFileTwice(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := &common.Upload{}
	uploadParams, err := pc.create(upload)
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, uploadParams, "invalid nil uploads params")
	require.NotZero(t, uploadParams.ID, "invalid upload id")

	file := &common.File{}
	file.Name = "filename"

	fileParams, err := pc.uploadFile(uploadParams, file, bytes.NewBufferString("data"))
	require.NoError(t, err, "unable to upload file")
	require.NotNil(t, fileParams, "invalid nil file params")
	require.NotZero(t, fileParams.ID, "invalid file id")

	_, err = pc.uploadFile(uploadParams, fileParams, bytes.NewBufferString("data"))
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), "invalid file status uploaded, expected missing", "invalid error")
}

func TestDownloadDuringUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.OneShot = true

	data := "data data data"
	lockedReader := NewLockedReader(bytes.NewBufferString(data))
	file := upload.AddFileFromReader("filename", lockedReader)

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.True(t, upload.Metadata().OneShot, "invalid upload non oneshot")

	// The file has not been uploaded
	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : missing", file.Name, file.metadata.ID), "invalid error")

	var wg sync.WaitGroup
	wg.Go(func() {
		err = upload.Upload()
		require.NoError(t, err, "unable to upload file")
	})

	time.Sleep(time.Second)

	// The file is being uploaded
	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : uploading", file.Name, file.metadata.ID), "invalid error")

	lockedReader.Unleash()
	wg.Wait()

	// The file has been uploaded
	reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to download file")
	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")
}

func TestOneShot(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.OneShot = true

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")

	require.True(t, upload.Metadata().OneShot, "invalid upload non oneshot")

	reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to download file")
	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")

	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : deleted", file.Name, file.metadata.ID), "invalid error")
}

func TestDownloadOneShotBeforeUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.OneShot = true

	data := "data data data"
	file := upload.AddFileFromReader("filename", bytes.NewBufferString(data))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.True(t, upload.Metadata().OneShot, "invalid upload non oneshot")

	// This should not trigger a file status change and make it impossible to download the file afterwards
	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : missing", file.Name, file.metadata.ID), "invalid error")

	err = upload.Upload()
	require.NoError(t, err, "unable to upload file")

	reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to download file")
	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file")
	require.Equal(t, data, string(content), "invalid file content")

	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : deleted", file.Name, file.metadata.ID), "invalid error")
}

func TestRemoveFileWithoutUploadToken(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Files(), 1, "invalid file count")

	upload.Metadata().UploadToken = ""
	err = pc.removeFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to remove file")
	require.Contains(t, err.Error(), "you are not allowed to remove files from this upload", "invalid error")
}

func TestRemovable(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.Removable = true

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"
	upload, file, err := pc.UploadReader("filename", bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file")
	require.Len(t, upload.Files(), 1, "invalid file count")

	upload.Metadata().UploadToken = ""
	err = pc.removeFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to upload file")
}

func TestUploadWithoutUploadToken(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	file := upload.AddFileFromReader("filename", bytes.NewBufferString("data"))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, upload.Metadata(), "upload has not been created")
	require.NotZero(t, upload.ID(), "invalid upload id")

	upload.Metadata().UploadToken = ""
	err = file.Upload()
	require.Error(t, err, "should not be able to upload file to anonymous upload")
	require.Contains(t, err.Error(), "you are not allowed to add file to this upload", "invalid error")
}

func TestStream(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.Stream = true

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	data := "data data data"

	upload := pc.NewUpload()
	file := upload.AddFileFromReader("filename", bytes.NewBufferString(data))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.True(t, upload.Stream, "invalid nil error params")

	errors := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		time.Sleep(50 * time.Millisecond)
		errors <- upload.Upload()
	})

	f := func() {
		for {
			time.Sleep(20 * time.Millisecond)
			reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
			if err != nil {
				continue
			}
			content, err := io.ReadAll(reader)
			require.NoError(t, err, "unable to read file")
			require.Equal(t, data, string(content), "invalid file content")
			break
		}
		wg.Wait()
	}

	err = common.TestTimeout(f, time.Second)
	require.NoError(t, err, "timeout")

	err = <-errors
	require.NoError(t, err, "upload error")

	time.Sleep(time.Second)

	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "unable to download file")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : deleted", file.Name, file.metadata.ID), "invalid error")
}

func TestTTL(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	upload.TTL = 1
	err = upload.Create()
	require.NoError(t, err, "unable to upload file")
	require.NotNil(t, upload.Metadata(), "upload has not been created")

	time.Sleep(2 * time.Second)

	_, err = pc.GetUpload(upload.ID())
	require.Error(t, err, "unable to get upload")
	require.Contains(t, err.Error(), "has expired", "upload has not been created")
}

func TestQuickUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://127.0.0.1:%d", ps.GetConfig().ListenPort)

	defer shutdown(ps)
	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	content := "data data data"

	var buf bytes.Buffer
	multipartWriter := multipart.NewWriter(&buf)
	writer, err := multipartWriter.CreateFormFile("file", "filename")
	require.NoError(t, err, "create multipart form file error : %s", err)

	_, err = io.Copy(writer, bytes.NewBufferString(content))
	require.NoError(t, err, "io copy error : %s", err)

	err = multipartWriter.Close()
	require.NoError(t, err, "multipart writer close error : %s", err)

	req, err := http.NewRequest("POST", pc.URL, &buf)
	require.NoError(t, err, "unable to create plik request")

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	resp, err := pc.makeRequest(req)
	require.NoError(t, err, "unable to make quick request (%s) %s", req.Method, req.URL.String())
	require.Equal(t, 200, resp.StatusCode, "invalid HTTP response status %s", resp.Status)

	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read response body")

	u, err := url.Parse(strings.TrimSpace(string(respBody)))
	require.NoError(t, err, "unable to parse url from response body")

	req, err = http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err, "unable to create new request")
	resp, err = pc.makeRequest(req)
	require.NoError(t, err, "unable to make quick request (%s) %s : %s", req.Method, req.URL.String())
	require.Equal(t, 200, resp.StatusCode, "invalid HTTP response status %s", resp.Status)

	defer func() { _ = resp.Body.Close() }()
	respBody, err = io.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, content, string(respBody), "invalid file content")
}

func TestCreateUploadWithForbidenOptions(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	uploadToCreate := &common.Upload{}
	uploadToCreate.IsAdmin = true
	uploadToCreate.DownloadDomain = "hack.me"
	uploadToCreate.RemoteIP = "1.3.3.7"
	uploadToCreate.UploadToken = "my-own-token"
	uploadToCreate.CreatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	uploadParams, err := pc.create(uploadToCreate)
	require.NoError(t, err, "unable to create upload")
	require.NotNil(t, uploadParams, "invalid nil uploads params")
	require.NotZero(t, uploadParams.ID, "invalid upload id")

	upload, err := pc.getUploadWithParams(&common.Upload{ID: uploadParams.ID})
	require.False(t, upload.Metadata().IsAdmin, "invalid upload admin status")
	require.Equal(t, "", upload.Metadata().DownloadDomain, "invalid upload download domain")
	require.Equal(t, "", upload.Metadata().RemoteIP, "invalid upload download domain")
	require.NotEqual(t, uploadToCreate.UploadToken, upload.Metadata().UploadToken, "invalid upload download domain")
	require.NotEqual(t, uploadToCreate.CreatedAt, upload.Metadata().CreatedAt, "invalid upload download domain")
}

func TestUploadFileWithSpecialChars(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// Filename with #, parentheses, and spaces — all problematic for URLs
	fileName := "#671488 (monitoring-configuration_grafana-dashboards).txt"
	data := "test data with special filename"

	upload, file, err := pc.UploadReader(fileName, bytes.NewBufferString(data))
	require.NoError(t, err, "unable to upload file with special characters in name")
	require.Equal(t, fileName, file.Name, "file name should be preserved")

	// Download the file and verify content
	reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to download file with special characters in name")
	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read file content")
	require.Equal(t, data, string(content), "file content mismatch")

	// Verify the URL is properly encoded
	fileURL, err := file.GetURL()
	require.NoError(t, err, "unable to get file URL")
	require.Contains(t, fileURL.String(), "%23", "# should be percent-encoded in URL")
	require.NotContains(t, fileURL.String(), "#671488", "raw # should not appear in URL path")

	// Remove should also work
	err = pc.removeFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to remove file with special characters in name")
}

func TestQuickUploadWithSpecialChars(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	ps.GetConfig().DownloadDomain = fmt.Sprintf("http://127.0.0.1:%d", ps.GetConfig().ListenPort)

	defer shutdown(ps)
	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	fileName := "#test (special).txt"
	content := "test data"

	var buf bytes.Buffer
	multipartWriter := multipart.NewWriter(&buf)
	writer, err := multipartWriter.CreateFormFile("file", fileName)
	require.NoError(t, err, "create multipart form file error : %s", err)

	_, err = io.Copy(writer, bytes.NewBufferString(content))
	require.NoError(t, err, "io copy error : %s", err)

	err = multipartWriter.Close()
	require.NoError(t, err, "multipart writer close error : %s", err)

	req, err := http.NewRequest("POST", pc.URL, &buf)
	require.NoError(t, err, "unable to create plik request")

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	resp, err := pc.makeRequest(req)
	require.NoError(t, err, "unable to make quick request (%s) %s", req.Method, req.URL.String())
	require.Equal(t, 200, resp.StatusCode, "invalid HTTP response status %s", resp.Status)

	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read response body")

	// The URL should contain encoded special characters
	rawURL := strings.TrimSpace(string(respBody))
	require.Contains(t, rawURL, "%23", "# should be encoded in quick mode URL")

	u, err := url.Parse(rawURL)
	require.NoError(t, err, "unable to parse url from response body")

	// Downloading via the encoded URL should work
	req, err = http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err, "unable to create new request")
	resp, err = pc.makeRequest(req)
	require.NoError(t, err, "unable to download file via quick mode URL")
	require.Equal(t, 200, resp.StatusCode, "invalid HTTP response status %s", resp.Status)

	defer func() { _ = resp.Body.Close() }()
	respBody, err = io.ReadAll(resp.Body)
	require.NoError(t, err, "unable to read response body")

	require.Equal(t, content, string(respBody), "invalid file content")
}

func TestEmptyFile(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload, file, err := pc.UploadReader("empty.txt", bytes.NewBufferString(""))
	require.NoError(t, err, "unable to upload empty file")
	require.NotNil(t, upload, "invalid nil upload")
	require.NotNil(t, file, "invalid nil file")

	reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to download empty file")
	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read empty file")
	require.Equal(t, "", string(content), "empty file content should be empty")
	require.Equal(t, 0, len(content), "empty file size should be 0")
}

func TestEmptyFileOneShot(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.OneShot = true

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload, file, err := pc.UploadReader("empty.txt", bytes.NewBufferString(""))
	require.NoError(t, err, "unable to upload empty file")

	require.True(t, upload.Metadata().OneShot, "invalid upload non oneshot")

	reader, err := pc.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err, "unable to download empty file")
	content, err := io.ReadAll(reader)
	require.NoError(t, err, "unable to read empty file")
	require.Equal(t, "", string(content), "empty file content should be empty")

	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : deleted", file.Name, file.metadata.ID), "invalid error")
}

func TestEmptyFileStream(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	pc.Stream = true

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	file := upload.AddFileFromReader("empty.txt", bytes.NewBufferString(""))

	err = upload.Create()
	require.NoError(t, err, "unable to create upload")
	require.True(t, upload.Stream, "invalid stream flag")

	// A 0-byte stream completes almost instantly — the upload pipe closes
	// immediately and the file transitions to "deleted" before any download
	// can connect.  This is expected behavior: streaming is ephemeral.
	err = upload.Upload()
	require.NoError(t, err, "unable to upload empty file in stream mode")

	// The file should already be deleted
	_, err = pc.downloadFile(upload.Metadata(), file.Metadata())
	require.Error(t, err, "missing error")
	require.Contains(t, err.Error(), fmt.Sprintf("file %s (%s) is not available : deleted", file.Name, file.metadata.ID), "invalid error")
}

func TestResumableUpload(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := startWithClient(ps, pc)
	require.NoError(t, err, "unable to start plik server")

	// The file is either declared at upload creation or added to the upload by an upload creation request
	for _, added := range []bool{false, true} {
		t.Run(fmt.Sprintf("added=%t", added), func(t *testing.T) {
			data := make([]byte, 5<<20)
			_, err := rand.Read(data)
			require.NoError(t, err, "unable to generate file content")

			upload := pc.NewUpload()
			if !added {
				upload.AddFileFromReader("filename", bytes.NewReader(data))
			}
			err = upload.Create()
			require.NoError(t, err, "unable to create upload")

			var file *common.File
			if !added {
				file = upload.Files()[0].Metadata()
			}

			var URL string
			request := func(method string, offset int, body io.Reader, length int) (*http.Response, error) {
				req, err := pc.uploadRequest(upload.Metadata(), method, URL, body)
				require.NoError(t, err, "unable to create request")
				if method == "PATCH" {
					req.Header.Set("Content-Type", "application/partial-upload")
					req.Header.Set("Upload-Offset", strconv.Itoa(offset))
				}
				req.Header.Set("Upload-Complete", "?0")
				if offset+length == len(data) {
					req.Header.Set("Upload-Complete", "?1")
				}
				req.ContentLength = int64(length)
				return pc.HTTPClient.Do(req)
			}

			// Appends are larger than the 1 MiB limit of API request bodies
			chunk := 2 << 20
			if added {
				URL = pc.URL + "/resumable/" + upload.ID() + "/filename?reference=ref"
				resp, err := request("POST", 0, bytes.NewReader(data[:chunk]), chunk)
				require.NoError(t, err, "unable to create file")
				_ = resp.Body.Close()
				require.Equal(t, http.StatusCreated, resp.StatusCode, "invalid HTTP response status %s", resp.Status)
				URL = pc.URL + resp.Header.Get("Location")
				file = &common.File{ID: strings.Split(URL, "/")[5], Name: "filename"}
			} else {
				URL = pc.URL + "/resumable/" + upload.ID() + "/" + file.ID + "/" + file.Name
				resp, err := request("PATCH", 0, bytes.NewReader(data[:chunk]), chunk)
				require.NoError(t, err, "unable to append")
				_ = resp.Body.Close()
				require.Equal(t, http.StatusNoContent, resp.StatusCode, "invalid HTTP response status %s", resp.Status)
			}

			// The connection drops in the middle of the second append
			_, err = request("PATCH", chunk, io.MultiReader(bytes.NewReader(data[chunk:chunk+chunk/2]), iotest.ErrReader(errors.New("connection lost"))), chunk)
			require.Error(t, err, "interrupted append should fail")

			// The client asks where to resume
			resp, err := request("HEAD", 0, nil, 0)
			require.NoError(t, err, "unable to get upload offset")
			require.Equal(t, http.StatusNoContent, resp.StatusCode, "invalid HTTP response status %s", resp.Status)
			offset, err := strconv.Atoi(resp.Header.Get("Upload-Offset"))
			require.NoError(t, err, "invalid upload offset")
			require.Equal(t, chunk, offset, "interrupted append should not be staged")

			for offset < len(data) {
				length := min(chunk, len(data)-offset)
				resp, err = request("PATCH", offset, bytes.NewReader(data[offset:offset+length]), length)
				require.NoError(t, err, "unable to append")
				_ = resp.Body.Close()
				offset += length
				if offset < len(data) {
					require.Equal(t, http.StatusNoContent, resp.StatusCode, "invalid HTTP response status %s", resp.Status)
				}
			}
			require.Equal(t, http.StatusOK, resp.StatusCode, "invalid HTTP response status %s", resp.Status)
			require.Equal(t, "?1", resp.Header.Get("Upload-Complete"), "upload should be complete")

			reader, err := pc.downloadFile(upload.Metadata(), file)
			require.NoError(t, err, "unable to download file")
			downloaded, err := io.ReadAll(reader)
			require.NoError(t, err, "unable to read file")
			require.True(t, bytes.Equal(data, downloaded), "downloaded file differs from the uploaded file")
		})
	}
}

type resumableReplicaBackend struct {
	data.Backend
	saving chan struct{}
	resume <-chan struct{}
}

func (b *resumableReplicaBackend) AddFile(file *common.File, reader io.Reader) error {
	close(b.saving)
	<-b.resume
	return b.Backend.AddFile(file, reader)
}

func TestResumableUploadAcrossReplicas(t *testing.T) {
	directory := os.Getenv("PLIK_RESUMABLE_REPLICA_DIRECTORY")
	child := directory != ""
	if !child {
		directory = t.TempDir()
	}
	config := common.NewConfiguration()
	config.ListenAddress, config.ListenPort = "127.0.0.1", 0
	config.AutoClean(false)
	config.LogOutput = io.Discard
	config.ChunkDirectory = filepath.Join(directory, "chunks")
	config.MetadataBackendConfig = map[string]any{"Driver": "sqlite3", "ConnectionString": filepath.Join(directory, "metadata.db")}
	require.NoError(t, config.Initialize())
	ps := server.NewPlikServer(config)
	t.Cleanup(func() { _ = ps.ShutdownNow() })
	backend := fileBackend.NewBackend(&fileBackend.Config{Directory: filepath.Join(directory, "files")})
	gate, resume := context.WithCancel(context.Background())
	t.Cleanup(resume)
	gated := &resumableReplicaBackend{Backend: backend, saving: make(chan struct{}), resume: gate.Done()}
	if child {
		ps.WithDataBackend(backend)
	} else {
		ps.WithDataBackend(gated)
	}
	require.NoError(t, ps.Start())
	if child {
		fmt.Println(ps.GetConfig().ListenPort)
		_, _ = io.Copy(io.Discard, os.Stdin)
		return
	}

	executable, err := os.Executable()
	require.NoError(t, err)
	command := exec.Command(executable, "-test.run=^TestResumableUploadAcrossReplicas$")
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		if name != "PLIKD_CONFIG" && name != "data_backend" && name != "data_backend_config" {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "PLIK_RESUMABLE_REPLICA_DIRECTORY="+directory)
	command.Stderr = os.Stderr
	input, err := command.StdinPipe()
	require.NoError(t, err)
	output, err := command.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, command.Start())
	t.Cleanup(func() {
		_ = input.Close()
		_ = command.Process.Kill()
		_ = command.Wait()
	})
	port := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		if scanner.Scan() {
			port <- scanner.Text()
		}
	}()
	var replica *Client
	select {
	case value := <-port:
		replica = NewClient("http://127.0.0.1:" + value)
	case <-time.After(10 * time.Second):
		t.Fatal("replica did not start")
	}
	pc := NewClient(config.GetServerURL().String())
	pc.HTTPClient.Timeout, replica.HTTPClient.Timeout = 10*time.Second, 10*time.Second
	upload := pc.NewUpload()
	file := upload.AddFileFromReader("filename", strings.NewReader("hello world"))
	require.NoError(t, upload.Create())
	path := "/resumable/" + upload.ID() + "/" + file.Metadata().ID + "/filename"
	request := func(client *Client, method string, offset int, complete, body string) *http.Request {
		req, err := client.uploadRequest(upload.Metadata(), method, client.URL+path, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/partial-upload")
		req.Header.Set("Upload-Offset", strconv.Itoa(offset))
		req.Header.Set("Upload-Complete", complete)
		return req
	}
	resp, err := replica.HTTPClient.Do(request(replica, "PATCH", 0, "?0", "hello "))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	completion := request(pc, "PATCH", 6, "?1", "world")
	completed := make(chan error, 1)
	go func() {
		resp, err := pc.HTTPClient.Do(completion)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("completion returned %s", resp.Status)
			}
		}
		completed <- err
	}()
	select {
	case <-gated.saving:
	case <-time.After(10 * time.Second):
		t.Fatal("upload did not reach the data backend")
	}

	// A different process must wait while the completing append saves the staged bytes.
	deadline, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err = replica.HTTPClient.Do(request(replica, "PATCH", 11, "?0", "unexpected").WithContext(deadline))
	if resp != nil {
		_ = resp.Body.Close()
	}
	require.ErrorIs(t, err, context.DeadlineExceeded)
	resume()
	require.NoError(t, <-completed)
	resp, err = replica.HTTPClient.Do(request(replica, "HEAD", 0, "?0", ""))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, "11", resp.Header.Get("Upload-Offset"))
	require.Equal(t, "?1", resp.Header.Get("Upload-Complete"))
	reader, err := replica.downloadFile(upload.Metadata(), file.Metadata())
	require.NoError(t, err)
	defer reader.Close()
	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(downloaded))
}
