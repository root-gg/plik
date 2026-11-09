package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gofrs/flock"
	"github.com/gorilla/mux"

	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
	"github.com/root-gg/plik/server/metadata"
)

// Resumable uploads follow the IETF Resumable Uploads for HTTP draft (draft-ietf-httpbis-resumable-upload-15).
// The upload resource of a file is /resumable/{uploadID}/{fileID}/{filename} : HEAD returns its offset, PATCH
// appends data and DELETE cancels it. Files declared at upload creation already have one, POST
// /resumable/{uploadID}/{filename} adds a file to an upload and creates its upload resource. Appended data is
// staged in ChunkDirectory/{fileID} until the client marks the upload complete, then the whole file is saved
// to the data backend like a regular upload.

// lockStagedFile serializes access to staged data across processes sharing the directory.
// Lock files remain until their metadata is terminal, so waiters lock the same inode.
func lockStagedFile(directory, fileID string) (unlock func(), err error) {
	if err = os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(directory, fileID+".lock"), flock.SetFlag(os.O_CREATE|os.O_RDWR))
	if err = lock.Lock(); err != nil {
		return nil, err
	}
	return func() { _ = lock.Close() }, nil
}

// resumableUpload checks that the request may add files to its upload, or returns nil after writing the error response
func resumableUpload(ctx *context.Context) (upload *common.Upload) {
	upload = ctx.GetUpload()
	if upload == nil {
		panic("missing upload from context")
	}

	if !upload.IsAdmin {
		ctx.Forbidden("you are not allowed to add file to this upload")
		return nil
	}

	if upload.Stream {
		ctx.BadRequest("resumable upload is not available for stream uploads")
		return nil
	}

	return upload
}

// resumableFile returns the upload and file of a request to an upload resource, or nil after writing the error response
func resumableFile(ctx *context.Context) (upload *common.Upload, file *common.File) {
	log := ctx.GetLogger()

	upload = resumableUpload(ctx)
	if upload == nil {
		return nil, nil
	}

	file = ctx.GetFile()
	if file == nil {
		panic("missing file from context")
	}

	// Update request logger prefix
	prefix := fmt.Sprintf("%s[%s]", log.Prefix, file.Name)
	log.SetPrefix(prefix)

	return upload, file
}

// currentFile reloads a file, which another request may have changed since the middleware loaded it,
// or returns nil after writing the error response. The caller must hold the lock of the file.
func currentFile(ctx *context.Context, file *common.File) *common.File {
	file, err := ctx.GetMetadataBackend().GetFile(file.ID)
	if err != nil {
		ctx.InternalServerError("unable to get file metadata", err)
		return nil
	}
	if file == nil {
		ctx.NotFound("file not found")
	}
	return file
}

// stagedOffset returns the number of bytes staged for a file, ok is false if the file does not accept appends
func stagedOffset(directory string, file *common.File) (offset int64, ok bool) {
	// Opening the data file refreshes attributes cached by shared filesystems.
	staged, err := os.Open(filepath.Join(directory, file.ID))
	if err != nil {
		return 0, file.Status == common.FileMissing && os.IsNotExist(err)
	}
	defer staged.Close()
	info, err := staged.Stat()
	if err != nil {
		return 0, false
	}
	return info.Size(), file.Status == common.FileMissing || file.Status == common.FileUploading
}

// setUploadLimit advertises the maximum file size in the Upload-Limit header
func setUploadLimit(ctx *context.Context, resp http.ResponseWriter) {
	maxFileSize := ctx.GetMaxFileSize()
	if maxFileSize > 0 {
		resp.Header().Set("Upload-Limit", fmt.Sprintf("max-size=%d", maxFileSize))
	}
}

// structuredItem returns the value of a structured header field item without its parameters
func structuredItem(value string) string {
	item, _, _ := strings.Cut(value, ";")
	return strings.TrimSpace(item)
}

// uploadComplete parses the Upload-Complete header, ok is false if it is missing or invalid
func uploadComplete(req *http.Request) (complete bool, ok bool) {
	switch structuredItem(req.Header.Get("Upload-Complete")) {
	case "?1":
		return true, true
	case "?0":
		return false, true
	}
	return false, false
}

// GetFileOffset returns how many bytes of a file its resumable upload has received
func GetFileOffset(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	_, file := resumableFile(ctx)
	if file == nil {
		return
	}

	unlock, err := lockStagedFile(ctx.GetConfig().ChunkDirectory, file.ID)
	if err != nil {
		ctx.InternalServerError("unable to lock staged file", err)
		return
	}
	defer unlock()
	file = currentFile(ctx, file)
	if file == nil {
		return
	}

	resp.Header().Set("Cache-Control", "no-store")
	setUploadLimit(ctx, resp)

	if file.Status == common.FileUploaded {
		resp.Header().Set("Upload-Complete", "?1")
		resp.Header().Set("Upload-Offset", strconv.FormatInt(file.Size, 10))
		resp.WriteHeader(http.StatusNoContent)
		return
	}

	offset, ok := stagedOffset(ctx.GetConfig().ChunkDirectory, file)
	if !ok {
		ctx.BadRequest("invalid file status %s", file.Status)
		return
	}

	resp.Header().Set("Upload-Complete", "?0")
	resp.Header().Set("Upload-Offset", strconv.FormatInt(offset, 10))
	resp.WriteHeader(http.StatusNoContent)
}

// AppendFile appends the request content to the resumable upload of a file. With "Upload-Complete: ?1"
// the request content ends the file, which is then saved to the data backend.
func AppendFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Upload-Complete", "?0")

	upload, file := resumableFile(ctx)
	if file == nil {
		return
	}

	mediaType, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if mediaType != "application/partial-upload" || req.Header.Get("Content-Encoding") != "" {
		ctx.Fail("expected application/partial-upload content without content coding", nil, http.StatusUnsupportedMediaType)
		return
	}

	offset, err := strconv.ParseInt(structuredItem(req.Header.Get("Upload-Offset")), 10, 64)
	if err != nil || offset < 0 {
		ctx.InvalidParameter("Upload-Offset header")
		return
	}

	complete, ok := uploadComplete(req)
	if !ok {
		ctx.InvalidParameter("Upload-Complete header")
		return
	}

	appendData(ctx, resp, req, upload, file, offset, complete, http.StatusNoContent)
}

// AddResumableFile adds a file to an existing upload and creates its upload resource (upload creation), which
// the Location header of the response points to. The request content is the beginning of the file, the whole
// file with "Upload-Complete: ?1". A creation with the reference of a file of the upload returns that file, so
// a client retries a creation without adding the file twice.
func AddResumableFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	log := ctx.GetLogger()
	config := ctx.GetConfig()
	resp.Header().Set("Upload-Complete", "?0")

	upload := resumableUpload(ctx)
	if upload == nil {
		return
	}

	if req.Header.Get("Content-Encoding") != "" {
		ctx.Fail("content codings are not supported", nil, http.StatusUnsupportedMediaType)
		return
	}

	complete, ok := uploadComplete(req)
	if !ok {
		ctx.InvalidParameter("Upload-Complete header")
		return
	}

	var length int64
	if value := req.Header.Get("Upload-Length"); value != "" {
		var err error
		length, err = strconv.ParseInt(structuredItem(value), 10, 64)
		if err != nil || length < 0 {
			ctx.InvalidParameter("Upload-Length header")
			return
		}
	}

	name := mux.Vars(req)["filename"]
	reference := req.URL.Query().Get("reference")

	var file *common.File
	if reference != "" {
		files, err := ctx.GetMetadataBackend().GetFiles(upload.ID)
		if err != nil {
			ctx.InternalServerError("unable to get upload files", err)
			return
		}
		for _, f := range files {
			if f.Reference == reference && f.Name == name {
				file = f
			}
		}
	}

	if file == nil {
		count, err := ctx.GetMetadataBackend().CountUploadFiles(upload.ID)
		if err != nil {
			ctx.InternalServerError("unable get upload file count", err)
			return
		}
		if count >= config.MaxFilePerUpload {
			ctx.BadRequest("maximum number file per upload reached, limit is %d", config.MaxFilePerUpload)
			return
		}

		file, err = ctx.CreateFile(upload, &common.File{Name: name, Size: length, Reference: reference})
		if err != nil {
			ctx.BadRequest("unable to create file : %s", err.Error())
			return
		}

		err = ctx.GetMetadataBackend().CreateFile(file)
		if err != nil {
			ctx.InternalServerError("unable to create file", err)
			return
		}
	}

	// Update request logger prefix
	prefix := fmt.Sprintf("%s[%s]", log.Prefix, file.Name)
	log.SetPrefix(prefix)

	resp.Header().Set("Location", fmt.Sprintf("%s/resumable/%s/%s/%s", config.Path, upload.ID, file.ID, url.PathEscape(file.Name)))
	setUploadLimit(ctx, resp)

	appendData(ctx, resp, req, upload, file, 0, complete, http.StatusCreated)
}

// appendData appends the request content to the resumable upload of a file at the given offset, and saves the
// file to the data backend once complete. An incomplete upload gets a response with the given status code.
func appendData(ctx *context.Context, resp http.ResponseWriter, req *http.Request, upload *common.Upload, file *common.File, offset int64, complete bool, status int) {
	directory := ctx.GetConfig().ChunkDirectory
	err := os.MkdirAll(directory, 0700)
	if err != nil {
		ctx.InternalServerError("unable to create chunk directory", err)
		return
	}

	// Receive the request content in a file of its own, so an interrupted request leaves the staged data intact
	part, err := os.CreateTemp(directory, file.ID+".*.part")
	if err != nil {
		ctx.InternalServerError("unable to create chunk file", err)
		return
	}
	defer func() {
		_ = part.Close()
		_ = os.Remove(part.Name())
	}()

	reader := io.Reader(req.Body)
	maxFileSize := ctx.GetMaxFileSize()
	if maxFileSize > 0 {
		reader = io.LimitReader(reader, max(maxFileSize-offset+1, 0))
	}

	received, err := io.Copy(part, reader)
	if err != nil {
		ctx.InternalServerError("unable to read data from request body", err)
		return
	}
	if maxFileSize > 0 && offset+received > maxFileSize {
		setUploadLimit(ctx, resp)
		ctx.BadRequest("file too big (limit is set to %s)", humanize.Bytes(uint64(maxFileSize)))
		return
	}

	unlock, err := lockStagedFile(directory, file.ID)
	if err != nil {
		ctx.InternalServerError("unable to lock staged file", err)
		return
	}
	defer unlock()
	file = currentFile(ctx, file)
	if file == nil {
		return
	}

	if file.Status == common.FileUploaded && offset == file.Size && received == 0 {
		// The response to the request that completed the upload was lost
		resp.Header().Set("Upload-Complete", "?1")
		file.Sanitize()
		common.WriteJSONResponse(resp, file)
		return
	}

	current, ok := stagedOffset(directory, file)
	if !ok {
		ctx.BadRequest("invalid file status %s", file.Status)
		return
	}

	if offset != current {
		resp.Header().Set("Upload-Offset", strconv.FormatInt(current, 10))
		resp.Header().Set("Content-Type", "application/problem+json")
		resp.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(resp).Encode(map[string]any{
			"type":            "https://iana.org/assignments/http-problem-types#mismatching-upload-offset",
			"title":           "offset from request does not match offset of resource",
			"expected-offset": current,
			"provided-offset": offset,
		})
		return
	}

	staged := filepath.Join(directory, file.ID)
	stagedFile, err := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		ctx.InternalServerError("unable to open staged file", err)
		return
	}
	_, err = io.Copy(stagedFile, io.NewSectionReader(part, 0, received))
	errClose := stagedFile.Close()
	if err == nil {
		err = errClose
	}
	if err != nil {
		_ = os.Truncate(staged, current)
		ctx.InternalServerError("unable to append to staged file", err)
		return
	}

	if file.Status == common.FileMissing {
		err = ctx.GetMetadataBackend().UpdateFileStatus(file, common.FileMissing, common.FileUploading)
		if err != nil {
			_ = os.Remove(staged)
			ctx.InternalServerError("unable to update file status", err)
			return
		}
	}

	if !complete {
		resp.Header().Set("Upload-Offset", strconv.FormatInt(current+received, 10))
		resp.WriteHeader(status)
		return
	}

	stagedFile, err = os.Open(staged)
	if err != nil {
		ctx.InternalServerError("unable to open staged file", err)
		return
	}
	defer func() {
		_ = stagedFile.Close()
		_ = os.Remove(staged)
	}()

	resp.Header().Set("Upload-Complete", "?1")
	saveFile(ctx, resp, upload, file, stagedFile)
}

// PurgeStaleChunks deletes the staged data of files that are no longer uploading, and of
// files that received no data for a day, which are then marked as removed.
func PurgeStaleChunks(directory string, metadataBackend *metadata.Backend) (purged int, err error) {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		removed, err := purgeStaleChunk(directory, entry.Name(), metadataBackend)
		if err != nil {
			return purged, err
		}
		if removed {
			purged++
		}
	}

	return purged, nil
}

// purgeStaleChunk deletes a staged file or a part left behind by a stopped server if it is stale
func purgeStaleChunk(directory string, name string, metadataBackend *metadata.Backend) (removed bool, err error) {
	isPart := strings.HasSuffix(name, ".part")
	isLock := strings.HasSuffix(name, ".lock")
	fileID := strings.TrimSuffix(name, ".lock")
	var unlock func()
	if !isPart {
		unlock, err = lockStagedFile(directory, fileID)
		if err != nil {
			return false, err
		}
		defer unlock()
	}

	var stale bool
	if !isLock {
		staged, err := os.Open(filepath.Join(directory, name))
		if os.IsNotExist(err) {
			// Deleted once its upload completed
			return false, nil
		}
		if err != nil {
			return false, err
		}
		info, err := staged.Stat()
		_ = staged.Close()
		if err != nil {
			return false, err
		}
		stale = time.Since(info.ModTime()) > 24*time.Hour
	}

	if !isPart {
		file, err := metadataBackend.GetFile(fileID)
		if err != nil {
			return false, err
		}
		if isLock && file != nil && (file.Status == common.FileMissing || file.Status == common.FileUploading) {
			return false, nil
		}
		if !isLock && file != nil && file.Status == common.FileUploading {
			if !stale {
				return false, nil
			}
			err = metadataBackend.RemoveFile(file)
			if err != nil {
				return false, err
			}
		}
	} else if !stale {
		return false, nil
	}

	if isLock {
		unlock()
	}
	err = os.Remove(filepath.Join(directory, name))
	if err != nil {
		return false, err
	}
	return !isLock, nil
}
