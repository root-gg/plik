package context

import (
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
)

// CreateUpload from params and context (check configuration and default values, generate upload and file IDs, ... )
func (ctx *Context) CreateUpload(params *common.Upload) (upload *common.Upload, err error) {
	upload = common.NewUpload()

	if ctx.GetSourceIP() != nil {
		upload.RemoteIP = ctx.GetSourceIP().String()
	}

	// Set user
	err = ctx.setUser(upload)
	if err != nil {
		return nil, err
	}

	// Set user configurable parameters
	err = ctx.setParams(upload, params)
	if err != nil {
		return nil, err
	}

	// Set TTL
	err = ctx.setTTL(upload, params.TTL)
	if err != nil {
		return nil, err
	}

	// Handle Basic Auth parameters
	err = ctx.setBasicAuth(upload, params.Login, params.Password)
	if err != nil {
		return nil, err
	}

	// Handle files
	err = ctx.setFiles(upload, params.Files)
	if err != nil {
		return nil, err
	}

	return upload, nil
}

func (ctx *Context) setUser(upload *common.Upload) (err error) {
	config := ctx.GetConfig()
	user := ctx.GetUser()
	token := ctx.GetToken()

	if user == nil {
		if config.NoAnonymousUploads {
			return fmt.Errorf("anonymous uploads are disabled")
		}
	} else {
		if !config.Authentication {
			return fmt.Errorf("authentication is disabled")
		}
		upload.User = user.ID
		if token != nil {
			upload.Token = token.Token
		}
	}
	return nil
}

func (ctx *Context) setParams(upload *common.Upload, params *common.Upload) (err error) {
	config := ctx.GetConfig()

	upload.OneShot = params.OneShot
	if upload.OneShot && !config.OneShot {
		return fmt.Errorf("one shot uploads are not enabled")
	}

	upload.Removable = params.Removable
	if upload.Removable && !config.Removable {
		return fmt.Errorf("removable uploads are not enabled")
	}

	upload.Stream = params.Stream
	if upload.Stream && !config.Stream {
		return fmt.Errorf("stream mode is not enabled")
	}

	upload.TTL = params.TTL
	upload.Comments = params.Comments

	return nil
}

// SetTTL adjust TTL parameters accordingly to default and max TTL
func (ctx *Context) setTTL(upload *common.Upload, TTL int) (err error) {
	config := ctx.GetConfig()
	user := ctx.GetUser()

	// TTL = Time in second before the upload expiration
	// >0 	-> TTL specified
	// 0 	-> No TTL specified : default value from configuration
	// <0	-> No expiration
	if TTL == 0 {
		TTL = config.DefaultTTL
	}

	maxTTL := config.MaxTTL
	if user != nil && user.MaxTTL != 0 {
		maxTTL = user.MaxTTL
	}

	if maxTTL > 0 {
		if TTL <= 0 {
			return fmt.Errorf("cannot set infinite TTL (maximum allowed is : %d)", maxTTL)
		}
		if TTL > maxTTL {
			return fmt.Errorf("invalid TTL. (maximum allowed is : %d)", maxTTL)
		}
	}

	upload.CreatedAt = time.Now()
	upload.TTL = TTL
	if upload.TTL > 0 {
		deadline := upload.CreatedAt.Add(time.Duration(upload.TTL) * time.Second)
		upload.ExpireAt = &deadline
	}

	return nil
}

func (ctx *Context) setBasicAuth(upload *common.Upload, login string, password string) (err error) {
	config := ctx.GetConfig()
	if !config.ProtectedByPassword && (login != "" || password != "") {
		return fmt.Errorf("password protection is not enabled")
	}

	if login != "" {
		upload.Login = login
	} else {
		upload.Login = "plik"
	}

	if password == "" {
		return nil
	}

	upload.ProtectedByPassword = true

	// Save only the md5sum of this string to authenticate further requests
	upload.Password, err = utils.Md5sum(common.EncodeAuthBasicHeader(login, password))
	if err != nil {
		return fmt.Errorf("unable to generate password hash : %s", err)
	}

	return nil
}

func (ctx *Context) setFiles(upload *common.Upload, files []*common.File) (err error) {
	config := ctx.GetConfig()

	// Limit number of files per upload
	if len(files) > config.MaxFilePerUpload {
		return fmt.Errorf("too many files. maximum is %d", config.MaxFilePerUpload)
	}

	// Create and check files
	for _, fileParams := range files {
		file, err := ctx.CreateFile(upload, fileParams)
		if err != nil {
			return err
		}
		upload.Files = append(upload.Files, file)
	}

	return nil
}

// CreateFile prepares a new file object to be persisted in DB ( create file ID, link upload ID, check name, ... )
func (ctx *Context) CreateFile(upload *common.Upload, params *common.File) (file *common.File, err error) {
	if upload.ID == "" {
		return nil, fmt.Errorf("upload not initialized")
	}

	file = common.NewFile()
	file.Status = common.FileMissing
	file.UploadID = upload.ID

	file.Name = params.Name
	file.Type = params.Type
	file.Size = params.Size
	file.Reference = params.Reference

	if file.Name == "" {
		return nil, fmt.Errorf("missing file name")
	}

	// Check file name length
	if len(file.Name) > 1024 {
		return nil, fmt.Errorf("file name %s... is too long, maximum length is 1024 characters", file.Name[:20])
	}

	// Check file size
	maxFileSize := ctx.GetMaxFileSize()
	if file.Size > 0 && maxFileSize > 0 && file.Size > maxFileSize {
		return nil, fmt.Errorf("file is too big (%s), maximum file size is %s", humanize.Bytes(uint64(file.Size)), humanize.Bytes(uint64(maxFileSize)))
	}

	return file, nil
}

// GetMaxFileSize return the maximum allowed file size for the context
func (ctx *Context) GetMaxFileSize() int64 {
	user := ctx.GetUser()
	if user != nil && user.MaxFileSize != 0 {
		return user.MaxFileSize
	}

	return ctx.GetConfig().MaxFileSize
}
