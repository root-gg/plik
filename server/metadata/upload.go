package metadata

import (
	"fmt"
	"time"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"

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
	if err == gorm.ErrRecordNotFound {
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

	result, c, err := p.Paginate(stmt, &uploads)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return uploads, &c, err
}

// RemoveUpload soft delete upload ( just set upload.DeletedAt field ) and remove all files
// The upload metadata will still be present in the metadata database as well as all the files
// Until all the files are deleted from the data backend and
func (b *Backend) RemoveUpload(uploadID string) (err error) {
	err = b.db.Transaction(func(tx *gorm.DB) (err error) {
		err = b.removeUploadFiles(tx, uploadID)
		if err != nil {
			return fmt.Errorf("unable to delete upload files : %s", err)
		}

		err = tx.Delete(&common.Upload{ID: uploadID}).Error
		if err != nil {
			return fmt.Errorf("unable to (soft) delete upload : %s", err)
		}

		return nil
	})

	return err
}

// RemoveExpiredUploads soft delete all expired uploads and remove all their files
func (b *Backend) RemoveExpiredUploads() (removed int, err error) {
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

		err := b.RemoveUpload(upload.ID)
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

// DeleteRemovedUploads delete upload and file metadata from the database once :
//  - The upload has been removed (soft delete) either manually or because it expired
//  - All the upload files have been deleted from the data backend (status Deleted)
func (b *Backend) DeleteRemovedUploads() (removed int, err error) {
	b.log.Infof("Purging deleted uploads")

	rows, err := b.db.Model(&common.Upload{}).Unscoped().Where("deleted_at IS NOT NULL").Rows()
	if err != nil {
		return 0, fmt.Errorf("unable to fetch deleted uploads : %s", err)
	}
	defer func() { _ = rows.Close() }()

	errors := 0
	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return removed, fmt.Errorf("unable to fetch next expired upload : %s", err)
		}

		b.log.Debugf("Purging upload %s", upload.ID)

		// One transaction per upload
		err = b.db.Transaction(func(tx *gorm.DB) (err error) {

			// Ensure all files have been deleted from the data backend
			var count int64
			err = tx.Model(&common.File{}).Not(&common.File{Status: common.FileDeleted}).Where(&common.File{UploadID: upload.ID}).Count(&count).Error
			if err != nil {
				return fmt.Errorf("Unable to count files for upload %s : %s", upload.ID, err)
			}

			if count > 0 {
				b.log.Warningf("Unable to remove upload %s because %d files are still not deleted", upload.ID, count)

				// This should not happen anymore but in the past there was a possibility
				// for upload to be removed without having all their files removed.
				// In this case simply remove the files again and stop here.
				// The files will be deleted from the data backend during the next cleaning cycle
				err = b.removeUploadFiles(tx, upload.ID)
				if err != nil {
					return err
				}

				// Hack the counters
				errors++
				removed--

				// We have to return nil to let the transaction commit to update the files status
				return nil
			}

			// Delete the upload files from the database
			err = tx.Where(&common.File{UploadID: upload.ID}).Delete(&common.File{}).Error
			if err != nil {
				return fmt.Errorf("Unable to delete files for upload %s : %s", upload.ID, err)
			}

			// Delete the upload from the database
			err = tx.Unscoped().Delete(upload).Error
			if err != nil {
				return fmt.Errorf("Unable to delete upload %s : %s", upload.ID, err)
			}

			return nil
		})
		if err != nil {
			errors++
			b.log.Warningf(err.Error())
		} else {
			removed++
		}
	}

	if errors > 0 {
		return removed, fmt.Errorf("unable to purge %d deleted uploads", errors)
	}

	return removed, nil
}

// ForEachUpload execute f for every upload in the database
func (b *Backend) forEachUpload(f func(upload *common.Upload) error, unscoped bool) (err error) {
	stmt := b.db.Model(&common.Upload{})
	if unscoped {
		stmt = stmt.Unscoped()
	}

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

// ForEachUpload execute f for every upload in the database
func (b *Backend) ForEachUpload(f func(upload *common.Upload) error) (err error) {
	return b.forEachUpload(f, false)
}

// ForEachUploadUnscoped execute f for every upload in the database even soft deleted ones
func (b *Backend) ForEachUploadUnscoped(f func(upload *common.Upload) error) (err error) {
	return b.forEachUpload(f, true)
}

// Deleted upload files can only have two status :
//  - Removed meaning that the file should not be served anymore but still has to be deleted from the server
//  - Deleted meaning that the file has been removed from the server
// An upload can only safely be purged (hard deleted) once all its files have been deleted
func (b *Backend) removeUploadFiles(tx *gorm.DB, uploadID string) (err error) {

	// Same logic as in RemoveFile but in batch

	err = tx.Model(&common.File{}).
		Where(&common.File{UploadID: uploadID}).
		Where(tx.Where(&common.File{Status: common.FileMissing}).Or(&common.File{Status: ""})).
		Update("status", common.FileDeleted).Error

	if err != nil {
		return err
	}

	err = tx.Model(&common.File{}).
		Where(&common.File{UploadID: uploadID}).
		Where(tx.Where(&common.File{Status: common.FileUploading}).Or(&common.File{Status: common.FileUploaded})).
		Update("status", common.FileRemoved).Error

	if err != nil {
		return err
	}

	return nil
}
