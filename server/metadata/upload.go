package metadata

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	paginator "github.com/pilagod/gorm-cursor-paginator"

	"github.com/root-gg/plik/server/common"
)

// CreateUpload create a new upload in DB
func (b *Backend) CreateUpload(upload *common.Upload) (err error) {
	return b.db.Create(upload).Error
}

// GetUpload return an upload from the DB ( return nil and no error if not found )
func (b *Backend) GetUpload(ID string) (upload *common.Upload, err error) {
	upload = &common.Upload{}

	err = b.db.Take(upload, &common.Upload{ID: ID}).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return upload, err
}

// GetUploads return uploads from DB
// userID and tokenStr are filters
// set withFiles to also fetch the files
func (b *Backend) GetUploads(userID string, tokenStr string, withFiles bool, pagingQuery *common.PagingQuery) (uploads []*common.Upload, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	whereClause := &common.Upload{}
	if userID != "" {
		whereClause.User = userID
	}
	if tokenStr != "" {
		whereClause.Token = tokenStr
	}

	stmt := b.db.Model(&common.Upload{}).Where(whereClause)

	if withFiles {
		stmt = stmt.Preload("Files")
	}

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "ID")

	err = p.Paginate(stmt, &uploads).Error
	if err != nil {
		return nil, nil, err
	}

	c := p.GetNextCursor()

	return uploads, &c, err
}

// RemoveUploadFiles set the file status to removed for all files of an upload
// The files are then deleted by the servers and their status set to removed
func (b *Backend) RemoveUploadFiles(uploadID string) (err error) {
	var errors []error
	f := func(file *common.File) (err error) {
		err = b.RemoveFile(file)
		if err != nil {
			errors = append(errors, err)
		}
		return nil
	}

	err = b.ForEachUploadFiles(uploadID, f)
	if err != nil {
		return err
	}
	if len(errors) > 0 {
		return fmt.Errorf("unable to remove %d files", len(errors))
	}

	return nil
}

// DeleteUpload soft delete upload ( just set upload.DeletedAt field ) and remove all files
func (b *Backend) DeleteUpload(uploadID string) (err error) {
	err = b.db.Delete(&common.Upload{ID: uploadID}).Error
	if err != nil {
		return fmt.Errorf("unable to delete upload")
	}

	err = b.RemoveUploadFiles(uploadID)
	if err != nil {
		return fmt.Errorf("unable to delete upload files")
	}

	return nil
}

// DeleteExpiredUploads soft delete all expired uploads
func (b *Backend) DeleteExpiredUploads() (removed int, err error) {
	rows, err := b.db.Model(&common.Upload{}).Where("expire_at < ?", time.Now()).Rows()
	if err != nil {
		return 0, fmt.Errorf("unable to fetch expired uploads : %s", err)
	}
	defer func() { _ = rows.Close() }()

	var errors []error
	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return 0, fmt.Errorf("unable to fetch next expired upload : %s", err)
		}

		err := b.DeleteUpload(upload.ID)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		removed++
	}

	if len(errors) > 0 {
		return removed, fmt.Errorf("unable to remove %d expired uploads", len(errors))
	}

	return removed, nil
}

// PurgeDeletedUploads ensure all files from an expired upload have been deleted
// Then delete the upload and files for good
func (b *Backend) PurgeDeletedUploads() (removed int, err error) {
	rows, err := b.db.Model(&common.Upload{}).Unscoped().Where("deleted_at IS NOT NULL").Rows()
	if err != nil {
		return 0, fmt.Errorf("unable to fetch deletred uploads : %s", err)
	}
	defer func() { _ = rows.Close() }()

	var errors []error
	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return removed, fmt.Errorf("unable to fetch next expired upload : %s", err)
		}

		var count int
		err := b.db.Model(&common.File{}).Not(&common.File{Status: common.FileDeleted}).Where(&common.File{UploadID: upload.ID}).Count(&count).Error
		if err != nil {
			return removed, err
		}
		if count > 0 {
			// TODO log properly
			fmt.Printf("Can't remove upload %s because %d files are still not deleted\n", upload.ID, count)
			continue
		}

		// Delete the upload files from the database
		err = b.db.Where(&common.File{UploadID: upload.ID}).Delete(&common.File{}).Error
		if err != nil {
			errors = append(errors, err)
			continue
		}

		// Delete the upload from the database
		err = b.db.Unscoped().Delete(upload).Error
		if err != nil {
			errors = append(errors, err)
			continue
		}
		removed++
	}

	if len(errors) > 0 {
		return removed, fmt.Errorf("unable to purge %d deleted uploads", len(errors))
	}

	return removed, nil
}

// ForEachUpload execute f for every upload in the database
func (b *Backend) ForEachUpload(f func(upload *common.Upload) error) (err error) {
	stmt := b.db.Model(&common.Upload{})

	rows, err := stmt.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return err
		}
		err = f(upload)
		if err != nil {
			return err
		}
	}

	return nil
}
