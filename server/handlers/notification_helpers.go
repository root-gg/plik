package handlers

import (
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/plik/server/context"
)

// checkAllFilesUploaded checks if all files in the upload have been uploaded
func checkAllFilesUploaded(ctx *context.Context, uploadID string) (bool, error) {
	files, err := ctx.GetMetadataBackend().GetFiles(uploadID)
	if err != nil {
		return false, err
	}

	if len(files) == 0 {
		return false, nil
	}

	for _, f := range files {
		if f.Status != common.FileUploaded {
			return false, nil
		}
	}

	return true, nil
}

// checkAllFilesDownloaded checks if all uploaded files in the upload have been downloaded at least once
func checkAllFilesDownloaded(ctx *context.Context, uploadID string) (bool, error) {
	files, err := ctx.GetMetadataBackend().GetFiles(uploadID)
	if err != nil {
		return false, err
	}

	if len(files) == 0 {
		return false, nil
	}

	for _, f := range files {
		if f.Status == common.FileUploaded && f.DownloadedAt == nil {
			return false, nil
		}
	}

	return true, nil
}
