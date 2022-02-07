package handlers

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"

	"github.com/dustin/go-humanize"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/data"
)

type preprocessOutputReturn struct {
	size     int64
	md5sum   string
	mimeType string
	err      error
}

// AddFile add a file to an existing upload.
func AddFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()
	config := ctx.GetConfig()

	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
	}

	// Check authorization
	if !upload.IsAdmin {
		ctx.Forbidden("you are not allowed to add file to this upload")
		return
	}

	// Get file handle form multipart request
	var fileReader io.Reader
	multiPartReader, err := req.MultipartReader()
	if err != nil {
		ctx.InvalidParameter("multipart form : %s", err)
		return
	}

	// Read multipart body until the "file" part
	var fileName string
	for {
		part, errPart := multiPartReader.NextPart()
		if errPart == io.EOF {
			break
		}
		if errPart != nil {
			ctx.InvalidParameter("multipart form : %s", errPart)
			return
		}
		if part.FormName() == "file" {
			fileReader = part
			fileName = part.FileName()
			break
		}
	}
	if fileReader == nil {
		ctx.MissingParameter("file from multipart form")
		return
	}
	if fileName == "" {
		ctx.MissingParameter("file name from multipart form")
		return
	}

	// Get file from context
	file := ctx.GetFile()
	if file == nil {
		count, err := ctx.GetMetadataBackend().CountUploadFiles(upload.ID)
		if err != nil {
			ctx.InternalServerError("unable get upload file count", err)
			return
		}

		if count >= config.MaxFilePerUpload {
			// TODO there is a slight race condition here
			// THIS SHOULD BE A DB CONSTRAINT
			ctx.BadRequest("maximum number file per upload reached, limit is %d", config.MaxFilePerUpload)
			return
		}

		// Create a new file object
		file, err = ctx.CreateFile(upload, &common.File{Name: fileName})
		if err != nil {
			ctx.BadRequest("unable to create file : %s", err.Error())
			return
		}

		// Update metadata
		err = ctx.GetMetadataBackend().CreateFile(file)
		if err != nil {
			ctx.InternalServerError("unable to create file", err)
			return
		}
	} else {
		if file.Name != fileName {
			ctx.BadRequest("invalid file name")
			return
		}
	}

	// Update request logger prefix
	prefix := fmt.Sprintf("%s[%s]", log.Prefix, file.Name)
	log.SetPrefix(prefix)

	if file.Status != common.FileMissing {
		ctx.BadRequest("invalid file status %s, expected %s", file.Status, common.FileMissing)
		return
	}

	// Update file status
	err = ctx.GetMetadataBackend().UpdateFileStatus(file, file.Status, common.FileUploading)
	if err != nil {
		ctx.InternalServerError("unable to update file status", err)
		return
	}

	// Pipe file data from the request body to a preprocessing goroutine
	//  - Guess content type
	//  - Compute/Limit upload size
	//  - Compute md5sum
	preprocessReader, preprocessWriter := io.Pipe()
	preprocessOutputCh := make(chan preprocessOutputReturn)
	go preprocessor(ctx, fileReader, preprocessWriter, preprocessOutputCh)

	// Save file in the data backend
	var backend data.Backend
	if upload.Stream {
		backend = ctx.GetStreamBackend()
	} else {
		backend = ctx.GetDataBackend()
	}

	err = backend.AddFile(file, preprocessReader)
	if err != nil {
		// TODO : file status is left to common.FileUploading we should set it to some common.FileUploadError
		// TODO : or we can set it back to common.FileMissing if we are sure data backends will handle that
		ctx.InternalServerError("unable to save file", err)
		return
	}

	// Get preprocessor goroutine output
	preprocessOutput := <-preprocessOutputCh
	if preprocessOutput.err != nil {
		// TODO : file status is left to common.FileUploading we should set it to some common.FileUploadError
		// TODO : or we can set it back to common.FileMissing if we are sure data backends will handle that
		handleHTTPError(ctx, preprocessOutput.err)
		return
	}

	// Fill-in file information
	file.Type = preprocessOutput.mimeType
	file.Size = preprocessOutput.size
	file.Md5 = preprocessOutput.md5sum

	// Update file status
	if upload.Stream {
		file.Status = common.FileDeleted
	} else {
		file.Status = common.FileUploaded
	}

	// Update file metadata
	err = ctx.GetMetadataBackend().UpdateFile(file, common.FileUploading)
	if err != nil {
		ctx.InternalServerError("unable to update file metadata", err)
		return
	}

	// Remove all private information (ip, data backend details, ...) before
	// sending metadata back to the client
	file.Sanitize()

	if ctx.IsQuick() {
		// Print the file url in the response.
		var url string
		if ctx.GetConfig().GetDownloadDomain() != nil {
			url = ctx.GetConfig().GetDownloadDomain().String()
		} else {
			url = ctx.GetConfig().GetServerURL().String()
		}

		url += fmt.Sprintf("/file/%s/%s/%s", upload.ID, file.ID, file.Name)

		_, _ = resp.Write([]byte(url + "\n"))
	} else {
		common.WriteJSONResponse(resp, file)
	}
}

//  - Guess content type
//  - Compute/Limit upload size
//  - Compute md5sum
func preprocessor(ctx *context.Context, file io.Reader, preprocessWriter io.WriteCloser, outputCh chan preprocessOutputReturn) {
	log := ctx.GetLogger()
	maxFileSize := ctx.GetMaxFileSize()

	var err error
	var totalBytes int64
	var mimeType string
	var md5sum string

	md5Hash := md5.New()
	buf := make([]byte, 1048)

	eof := false
	for !eof {
		bytesRead := 0
		bytesRead, err = file.Read(buf)
		if err == io.EOF {
			eof = true
			err = nil
			if bytesRead <= 0 {
				break
			}
		} else if err != nil {
			err = common.NewHTTPError("unable to read data from request body : %s", err, http.StatusInternalServerError)
			break
		}

		// Detect the content-type using the 512 first bytes
		if totalBytes == 0 {
			mimeType = http.DetectContentType(buf)
		}

		// Increment size
		totalBytes += int64(bytesRead)

		// Check upload max size limit
		if totalBytes > maxFileSize {
			err = common.NewHTTPError(fmt.Sprintf("file too big (limit is set to %s)", humanize.Bytes(uint64(maxFileSize))), nil, http.StatusBadRequest)
			break
		}

		// Compute md5sum
		_, err = md5Hash.Write(buf[:bytesRead])
		if err != nil {
			err = fmt.Errorf(err.Error())
			break
		}

		// Forward data to the data backend
		bytesWritten, err := preprocessWriter.Write(buf[:bytesRead])
		if err != nil {
			err = fmt.Errorf(err.Error())
			break
		}
		if bytesWritten != bytesRead {
			err = fmt.Errorf("invalid number of bytes written. Expected %d but got %d", bytesRead, bytesWritten)
			break
		}
	}

	errClose := preprocessWriter.Close()
	if errClose != nil {
		log.Warningf("unable to close preprocessWriter : %s", err)
	}

	if err != nil {
		outputCh <- preprocessOutputReturn{err: err}
	} else {
		md5sum = fmt.Sprintf("%x", md5Hash.Sum(nil))
		outputCh <- preprocessOutputReturn{size: totalBytes, md5sum: md5sum, mimeType: mimeType}
	}

	close(outputCh)
}
