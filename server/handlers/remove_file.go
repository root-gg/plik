package handlers

import (
	"fmt"
	"github.com/root-gg/plik/server/common"
	"net/http"

	"github.com/root-gg/plik/server/context"
)

// RemoveFile remove a file from an existing upload
func RemoveFile(ctx *context.Context, resp http.ResponseWriter, req *http.Request) {
	// Get upload from context
	upload := ctx.GetUpload()
	if upload == nil {
		ctx.InternalServerError("missing upload from context", nil)
		return
	}

	// Check authorization
	if !upload.Removable && !upload.IsAdmin {
		ctx.Forbidden("you are not allowed to remove files from this upload")
		return
	}

	// Get file from context
	file := ctx.GetFile()
	if file == nil {
		ctx.InternalServerError("missing file from context", nil)
		return
	}

	// Delete file
	err := ctx.GetMetadataBackend().RemoveFile(file)
	if err != nil {
		ctx.InternalServerError("unable to delete file", err)
		return
	}

	// Remove the file asynchronously
	err = purge(ctx, file)
	if err != nil {
		ctx.GetLogger().Warningf(err.Error())
	}

	_, _ = resp.Write([]byte("ok"))
}

// purge does its best to remove the file right now (before the next cleaning cycle)
func purge(ctx *context.Context, file *common.File) (err error) {
	err = ctx.GetMetadataBackend().RemoveFile(file)
	if err != nil {
		return fmt.Errorf("unable to remove file %s from upload %s : %s", file.ID, file.UploadID, err)
	}

	if file.Status != common.FileRemoved {
		return nil
	}

	err = ctx.GetDataBackend().RemoveFile(file)
	if err != nil {
		return fmt.Errorf("unable to delete file %s from upload %s : %s", file.ID, file.UploadID, err)
	}

	err = ctx.GetMetadataBackend().UpdateFileStatus(file, common.FileRemoved, common.FileDeleted)
	if err != nil {
		return fmt.Errorf("unable to update file status for deleted file %s from upload %s : %s", file.ID, file.UploadID, err)
	}

	return nil
}
