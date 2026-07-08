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
	return b.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(upload).Error
		if err != nil {
			return err
		}

		lastUploadAt := upload.CreatedAt
		if lastUploadAt.IsZero() {
			lastUploadAt = time.Now()
		}
		delta := uploadCreationDelta(upload, lastUploadAt)

		// Record the upload creation in its UTC day's rollup bucket. Canonical lock
		// order (see stats_download.go): rollups are written before the usage rows
		// below. On import this regenerates the day's Uploads count (bytes stay 0 —
		// the wire-byte path is not replayed); CreateUploadStatsDaily then restores
		// the authoritative row.
		err = b.recordDailyUploads(tx, statsDay(*delta.lastUploadAt), upload.User, upload.Token, 1, 0)
		if err != nil {
			return err
		}

		return b.applyUsageDelta(tx, upload.User, upload.Token, delta)
	})
}

// UpdateUploadExpirationDate updates an upload expiration date in DB
func (b *Backend) UpdateUploadExpirationDate(upload *common.Upload) (err error) {
	return b.db.Model(upload).Update("expire_at", upload.ExpireAt).Error
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

// UploadFilters holds optional filters for querying uploads.
// Boolean pointers: nil = no filter, true = only matching.
type UploadFilters struct {
	User      string
	Token     string
	OneShot   *bool
	Removable *bool
	Stream    *bool
	ExtendTTL *bool
	Password  *bool // maps to ProtectedByPassword column
	E2EE      *bool // maps to e2ee != '' check
}

// applyUploadFilters returns a scoped *gorm.DB with the given filters applied.
// Uses struct-based Where for User/Token (portable quoting across PG/MySQL/SQLite)
// and explicit Where clauses for booleans (GORM ignores zero-value bools in structs).
func applyUploadFilters(stmt *gorm.DB, f UploadFilters) *gorm.DB {
	if f.User != "" {
		stmt = stmt.Where(&common.Upload{User: f.User})
	}
	if f.Token != "" {
		stmt = stmt.Where(&common.Upload{Token: f.Token})
	}
	if f.OneShot != nil {
		stmt = stmt.Where("one_shot = ?", *f.OneShot)
	}
	if f.Removable != nil {
		stmt = stmt.Where("removable = ?", *f.Removable)
	}
	if f.Stream != nil {
		stmt = stmt.Where("stream = ?", *f.Stream)
	}
	if f.ExtendTTL != nil {
		stmt = stmt.Where("extend_ttl = ?", *f.ExtendTTL)
	}
	if f.Password != nil {
		stmt = stmt.Where("protected_by_password = ?", *f.Password)
	}
	if f.E2EE != nil {
		if *f.E2EE {
			stmt = stmt.Where("e2ee != ''")
		} else {
			stmt = stmt.Where("e2ee = '' OR e2ee IS NULL")
		}
	}
	return stmt
}

// CountUploads return the total number of uploads matching the optional filters
func (b *Backend) CountUploads(filters UploadFilters) (count int64, err error) {
	stmt := b.db.Model(&common.Upload{})
	stmt = applyUploadFilters(stmt, filters)

	err = stmt.Count(&count).Error
	return count, err
}

// Upload-list sort values shared by GET /uploads (server/handlers/admin.go)
// and GET /me/uploads (server/handlers/me.go). isUploadSort
// (server/handlers/misc.go) validates against these before the handlers'
// shared dispatch helper (getUploadsSorted) picks the matching sorted-fetch
// method below; the empty value is the default date ordering (GetUploads).
const (
	UploadSortDate            = "date"
	UploadSortSize            = "size"
	UploadSortDownloads       = "downloads"
	UploadSortDownloadedBytes = "downloadedBytes"
)

// GetUploads return uploads from DB
// set withFiles to also fetch the files
func (b *Backend) GetUploads(filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) (uploads []*common.Upload, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	stmt := b.db.Model(&common.Upload{})
	stmt = applyUploadFilters(stmt, filters)

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

// GetUploadsSortedBySize return uploads from DB sorted by size
// set withFiles to also fetch the files
func (b *Backend) GetUploadsSortedBySize(filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) (uploads []*common.Upload, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	// Lightweight struct for cursor pagination — only needs ID and the sort key
	type uploadRef struct {
		ID   string
		Size int64
	}
	var refs []*uploadRef

	// Subquery: compute total file size per upload (only counting uploaded files)
	sizeSubquery := b.db.
		Table("files").
		Select("upload_id, SUM(size) as total_size").
		Where("status = ?", common.FileUploaded).
		Group("upload_id")

	stmt := b.db.
		Model(&common.Upload{}).
		Select("uploads.id, COALESCE(sub.total_size, 0) as size").
		Joins("LEFT JOIN (?) AS sub ON sub.upload_id = uploads.id", sizeSubquery)
	stmt = applyUploadFilters(stmt, filters)

	// Setup paginator
	p := pagingQuery.Paginator()
	p.SetRules([]paginator.Rule{
		{
			Key:     "Size",
			SQLRepr: "COALESCE(sub.total_size, 0)",
		},
		{
			Key:     "ID",
			SQLRepr: "uploads.id",
		},
	}...)

	// Phase 1: paginate to get sorted upload IDs
	result, c, err := p.Paginate(stmt, &refs)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	if len(refs) == 0 {
		return uploads, &c, nil
	}

	// Phase 2: load full uploads by IDs with native Preload
	ids := make([]string, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}

	query := b.db.Where("id IN ?", ids)
	if withFiles {
		query = query.Preload("Files")
	}

	err = query.Find(&uploads).Error
	if err != nil {
		return nil, nil, err
	}

	// Reorder to match the pagination sort order
	uploads = reorderByRefs(ids, uploads, func(u *common.Upload) string { return u.ID })

	return uploads, &c, nil
}

// getUploadsSortedByColumn is the shared body of GetUploadsSortedByDownloads
// and GetUploadsSortedByDownloadedBytes, which differ by exactly one paginator
// Rule: the primary sort column. key is the Go-side paginator key (matching the
// field named by sqlColumn) and sqlColumn is its table-qualified SQL
// representation.
func (b *Backend) getUploadsSortedByColumn(filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery, key string, sqlColumn string) (uploads []*common.Upload, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}

	stmt := b.db.Model(&common.Upload{})
	stmt = applyUploadFilters(stmt, filters)

	if withFiles {
		stmt = stmt.Preload("Files")
	}

	p := pagingQuery.Paginator()
	p.SetRules(
		paginator.Rule{Key: key, SQLRepr: sqlColumn},
		paginator.Rule{Key: "CreatedAt", SQLRepr: "uploads.created_at"},
		paginator.Rule{Key: "ID", SQLRepr: "uploads.id"},
	)

	result, c, err := p.Paginate(stmt, &uploads)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return uploads, &c, err
}

// GetUploadsSortedByDownloads return uploads from DB sorted by lifetime download count.
// set withFiles to also fetch the files.
func (b *Backend) GetUploadsSortedByDownloads(filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) (uploads []*common.Upload, cursor *paginator.Cursor, err error) {
	return b.getUploadsSortedByColumn(filters, withFiles, pagingQuery, "DownloadCount", "uploads.download_count")
}

// GetUploadsSortedByDownloadedBytes return uploads from DB sorted by lifetime
// bytes served (downloaded_bytes). set withFiles to also fetch the files.
func (b *Backend) GetUploadsSortedByDownloadedBytes(filters UploadFilters, withFiles bool, pagingQuery *common.PagingQuery) (uploads []*common.Upload, cursor *paginator.Cursor, err error) {
	return b.getUploadsSortedByColumn(filters, withFiles, pagingQuery, "DownloadedBytes", "uploads.downloaded_bytes")
}

// RemoveUpload soft delete upload ( just set upload.DeletedAt field ) and remove all files.
// The upload metadata will still be present in the metadata database as well as all the files
// until all the files are deleted from the data backend and DeleteRemovedUploads purges them.
func (b *Backend) RemoveUpload(uploadID string) (err error) {
	err = b.db.Transaction(func(tx *gorm.DB) (err error) {
		// Canonical lock order (see stats_download.go): take the parent upload
		// row's write lock first, before reading or transitioning its file rows.
		// While we hold it no concurrent transition/download can change this
		// upload's file rows, so the decrement deltas derived from the aggregate
		// reads below match exactly the files this transaction moves out of the
		// uploaded state — no double decrement, no leaked increment.
		upload, err := b.lockUploadRow(tx, uploadID)
		if err != nil {
			return err
		}
		if upload == nil || upload.DeletedAt.Valid {
			// Never existed or already removed: nothing to do (idempotent).
			return nil
		}

		var uploadIDs = []string{uploadID}
		files, size, err := uploadedFileStatsForUploads(tx, uploadIDs)
		if err != nil {
			return err
		}
		var delta usageDelta
		err = addFileSizeStatsForUploads(tx, &delta, uploadIDs, []string{common.FileUploaded}, -1, 0)
		if err != nil {
			return err
		}

		err = b.removeUploadFiles(tx, uploadID)
		if err != nil {
			return fmt.Errorf("unable to delete upload files : %s", err)
		}

		result := tx.Delete(&common.Upload{ID: uploadID})
		if result.Error != nil {
			return fmt.Errorf("unable to (soft) delete upload : %s", result.Error)
		}

		if result.RowsAffected == 1 {
			addUploadFeatureUsage(&delta, upload, -1, 0)
			addUploadTTLUsage(&delta, upload, -1, 0)
			delta.currentUploads = -1
			delta.currentFiles = -files
			delta.currentSize = -size
			err = b.applyUsageDelta(tx, upload.User, upload.Token, delta)
			if err != nil {
				return fmt.Errorf("unable to update usage stats : %s", err)
			}
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
			b.log.Warningf("unable to remove expired upload %s : %s", upload.ID, err)
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
//   - The upload has been removed (soft delete) either manually or because it expired
//   - All the upload files have been deleted from the data backend (status Deleted)
func (b *Backend) DeleteRemovedUploads() (removed int, err error) {
	b.log.Infof("Purging deleted uploads")

	rows, err := b.db.Model(&common.Upload{}).Unscoped().Where("deleted_at IS NOT NULL").Rows()
	if err != nil {
		return 0, fmt.Errorf("unable to fetch deleted uploads : %s", err)
	}
	defer func() { _ = rows.Close() }()

	errors := 0
	fixups := 0
	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return removed, fmt.Errorf("unable to fetch next expired upload : %s", err)
		}

		b.log.Debugf("Purging upload %s", upload.ID)

		// One transaction per upload
		err = b.db.Transaction(func(tx *gorm.DB) (err error) {

			// Canonical lock order (see stats_download.go): lock the parent upload
			// row before reading, transitioning, or deleting its file rows. This
			// purge path deletes file rows before the upload row, so without the
			// lock it would acquire file/upload row locks in the reverse of the
			// canonical order and could AB-BA deadlock with a concurrent file
			// transition (e.g. removed -> deleted) on the same upload.
			if _, err = b.lockUploadRow(tx, upload.ID); err != nil {
				return err
			}

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

				// This upload needs another cleaning cycle, don't count it as removed or as an error
				fixups++

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
			b.log.Warningf("%s", err.Error())
		} else {
			removed++
		}
	}

	// Fixup transactions return nil to commit, so they increment removed.
	// Subtract them to get the actual purged count.
	removed -= fixups

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
//   - Removed meaning that the file should not be served anymore but still has to be deleted from the server
//   - Deleted meaning that the file has been removed from the server
//
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
