package plik

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/root-gg/plik/server/common"
)

func downloadSequence(file *File, delete bool) (err error) {
	url, err := file.upload.GetURL()
	if err != nil || url == nil {
		return fmt.Errorf("upload url error : %s", err)
	}

	url, err = file.GetURL()
	if err != nil || url == nil {
		return fmt.Errorf("file url error : %s", err)
	}

	reader, err := file.Download()
	if err != nil {
		return fmt.Errorf("download error : %s", err)
	}
	defer func() { _ = reader.Close() }()

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read error : %s", err)
	}

	if string(content) != fmt.Sprintf("data data data %s", file.Name) {
		return fmt.Errorf("file content missmatch")
	}

	if delete {
		err = file.Delete()
		if err != nil {
			return fmt.Errorf("delete error : %s", err)
		}

		_, err = file.Download()
		if err == nil {
			return fmt.Errorf("download deleted file missing error")
		}
	}

	return nil
}

func drainErrors(errors chan error) (err error) {
	close(errors)
	for err := range errors {
		return err
	}
	return nil
}

func TestMultipleUploadsInParallel(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	count := 10
	errors := make(chan error, count)
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			upload := pc.NewUpload()
			filename := fmt.Sprintf("file_%d", i)
			data := fmt.Sprintf("data data data %s", filename)
			file := upload.AddFileFromReader(filename, NewSlowReaderRandom(bytes.NewBufferString(data)))

			err := upload.Upload()
			if err != nil {
				errors <- fmt.Errorf("upload error : %s", err)
				return
			}

			errors <- downloadSequence(file, true)
		}(i)
	}

	wg.Wait()
	err = drainErrors(errors)
	require.NoError(t, err, "an error occurred")
}

func TestMultipleFilesInParallel(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()

	count := 30
	errors := make(chan error, count)
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			filename := fmt.Sprintf("file_%d", i)
			data := fmt.Sprintf("data data data %s", filename)
			file := upload.AddFileFromReader(filename, NewSlowReaderRandom(bytes.NewBufferString(data)))

			err := file.Upload()
			if err != nil {
				errors <- fmt.Errorf("upload error : %s", err)
				return
			}

			errors <- downloadSequence(file, false)
		}(i)
	}

	wg.Wait()
	err = drainErrors(errors)
	require.NoError(t, err, "an error occurred")
}

func TestMultipleFilesInParallelBusyLoop(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()

	var uploader sync.WaitGroup
	uploader.Add(1)
	done := make(chan struct{})
	go func() {
		defer uploader.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			//time.Sleep(100 * time.Millisecond)
			err := upload.Upload()
			if err != nil {
				panic(err)
			}
		}
	}()

	count := 30
	errors := make(chan error, count)
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			filename := fmt.Sprintf("file_%d", i)
			data := fmt.Sprintf("data data data %s", filename)
			file := newFileFromReader(upload, filename, NewSlowReaderRandom(bytes.NewBufferString(data)))

			var downloader sync.WaitGroup
			downloader.Add(1)
			download := func(metadata *common.File, err error) {
				defer downloader.Done()

				if err != nil {
					errors <- fmt.Errorf("upload error : %s", err)
					return
				}

				errors <- downloadSequence(file, true)
			}

			file.RegisterUploadCallback(download)
			upload.add(file)
			downloader.Wait()
		}(i)
	}

	wg.Wait()
	err = drainErrors(errors)
	require.NoError(t, err, "an error occurred")

	close(done)
	uploader.Wait()
}

func TestCreateAndGetUploadFilesInParallel(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	for i := 1; i <= 30; i++ {
		filename := fmt.Sprintf("file_%d", i)
		data := fmt.Sprintf("data data data %s", filename)
		upload.AddFileFromReader(filename, NewSlowReaderRandom(bytes.NewBufferString(data)))
	}

	err = upload.Upload()
	require.NoError(t, err, "unable to upload files")

	uploadResult, err := pc.GetUpload(upload.ID())
	require.NoError(t, err, "unable to upload file")
	require.Equal(t, upload.ID(), uploadResult.ID(), "upload has not been created")
	require.Len(t, uploadResult.Files(), len(upload.Files()), "file count mismatch")

	uploadResult.Metadata().UploadToken = upload.metadata.UploadToken

	files := uploadResult.Files()
	errors := make(chan error, len(files))
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file *File) {
			defer wg.Done()

			errors <- downloadSequence(file, true)
		}(file)
	}

	wg.Wait()
	err = drainErrors(errors)
	require.NoError(t, err, "an error occurred")
}

func TestUploadDownloadSameFileInParallel(t *testing.T) {
	ps, pc := newPlikServerAndClient()
	defer shutdown(ps)

	err := start(ps)
	require.NoError(t, err, "unable to start plik server")

	upload := pc.NewUpload()
	filename := fmt.Sprintf("file")
	data := fmt.Sprintf("data data data %s", filename)
	file := upload.AddFileFromReader(filename, NewSlowReaderRandom(bytes.NewBufferString(data)))

	count := 30
	errors := make(chan error, count)
	var wg sync.WaitGroup
	for i := 1; i <= count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			err := file.Upload()
			if err != nil {
				errors <- fmt.Errorf("upload error : %s", err)
				return
			}

			errors <- downloadSequence(file, false)
		}(i)
	}

	wg.Wait()
	err = drainErrors(errors)
	require.NoError(t, err, "an error occurred")

}
