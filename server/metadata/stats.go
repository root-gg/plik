package metadata

import "github.com/root-gg/plik/server/common"

// GetUploadStatistics return statistics about uploads
// for userID and tokenStr params : nil doesn't activate the filter, empty string enables the filter with an empty value to generate statistics about anonymous upload
func (b *Backend) GetUploadStatistics(userID *string, tokenStr *string) (uploads int, files int, size int64, err error) {

	// Count uploads
	stmt := b.db.Model(&common.Upload{})
	if userID != nil {
		stmt = stmt.Where("uploads.user = ?", userID)
	}
	if tokenStr != nil {
		stmt = stmt.Where("uploads.token = ?", tokenStr)
	}

	var uploadsCount int64 // Gorm V2 requires int64 for counts
	err = stmt.Count(&uploadsCount).Error
	if err != nil {
		return 0, 0, 0, err
	}

	// Count files
	stmt = b.db.Model(&common.File{}).Select("count(files.id), coalesce(sum(size),0)").Where("files.status = ?", common.FileUploaded)
	if userID != nil || tokenStr != nil {
		stmt = stmt.Joins("join uploads on uploads.id = files.upload_id")
		if userID != nil {
			stmt = stmt.Where("uploads.user = ?", userID)
		}
		if tokenStr != nil {
			stmt = stmt.Where("uploads.token = ?", tokenStr)
		}
	}

	err = stmt.Row().Scan(&files, &size)
	if err != nil {
		return 0, 0, 0, err
	}

	return int(uploadsCount), files, size, nil
}

// GetUserStatistics return statistics about user uploads
// for tokenStr params : nil doesn't activate the filter, empty string enables the filter with an empty value to generate statistics about upload without a token
func (b *Backend) GetUserStatistics(userID string, tokenStr *string) (stats *common.UserStats, err error) {
	uploads, files, size, err := b.GetUploadStatistics(&userID, tokenStr)
	if err != nil {
		return nil, err
	}

	stats = &common.UserStats{
		Uploads:   uploads,
		Files:     files,
		TotalSize: size,
	}

	return stats, nil
}

// GetServerStatistics return statistics about user all uploads
func (b *Backend) GetServerStatistics() (stats *common.ServerStats, err error) {
	users, err := b.CountUsers()
	if err != nil {
		return nil, err
	}

	uploads, files, size, err := b.GetUploadStatistics(nil, nil)
	if err != nil {
		return nil, err
	}

	anonID := ""
	anonUploads, _, anonSize, err := b.GetUploadStatistics(&anonID, nil)
	if err != nil {
		return nil, err
	}

	stats = &common.ServerStats{
		Users:            users,
		Uploads:          uploads,
		AnonymousUploads: anonUploads,
		Files:            files,
		TotalSize:        size,
		AnonymousSize:    anonSize,
	}

	return stats, nil
}
