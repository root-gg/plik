package metadata

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateFile persist a new file to the database
//
// Unlike UpdateFile/UpdateFileStatus/RemoveUpload, CreateFile mutates usage
// counters without first taking the parent upload row's lock (see
// lockUploadRow and the canonical lock order in stats_download.go). That is
// safe today only because every caller that creates a file already in a
// completed status (FileUploaded/FileRemoved/FileDeleted) is the importer,
// which runs single-threaded against an otherwise idle backend. A future
// live/concurrent caller that creates an already-completed file would need
// to take the upload row lock first, like those other writers do.
func (b *Backend) CreateFile(file *common.File) (err error) {
	return b.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(file).Error
		if err != nil {
			return err
		}

		// CreateFile is the import/backfill path: it never fabricates wire-byte
		// ingress (wireBytes=0), so usage_stats.uploaded_bytes stays "since upgrade".
		switch file.Status {
		case common.FileUploaded:
			return b.incrementUsageForCompletedFile(tx, file, true, 0)
		case common.FileRemoved, common.FileDeleted:
			return b.incrementUsageForCompletedFile(tx, file, false, 0)
		default:
			return nil
		}
	})
}

// GetFile return a file from the database ( nil and no error if not found )
func (b *Backend) GetFile(fileID string) (file *common.File, err error) {
	file = &common.File{}
	err = b.db.Where(&common.File{ID: fileID}).Take(file).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return file, err
}

// GetFiles return all files for the given upload ID
func (b *Backend) GetFiles(uploadID string) (files []*common.File, err error) {
	err = b.db.Where(&common.File{UploadID: uploadID}).Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, err
}

// updatableFileColumns are the columns UpdateFile is responsible for
// persisting. Select() is used instead of a plain struct Updates() because
// GORM's struct-based update skips zero-value fields ; for a completed file
// with an actual size of 0 (e.g. an empty upload) that would silently leave
// the previous (e.g. client-declared) size in the row while usage counters
// are credited with the real, in-memory size, so the persisted size and the
// counted size would drift apart. ID, UploadID and CreatedAt are immutable
// identity/audit columns, and DownloadCount/LastDownloadedAt are owned by the
// download recording path : none of these are set by UpdateFile callers and
// must not be clobbered here.
var updatableFileColumns = []string{"Name", "Status", "Type", "Size", "Md5", "IsText", "Reference", "BackendDetails"}

// UpdateFile update a file in DB. Status ensure the file status has not changed since loaded
func (b *Backend) UpdateFile(file *common.File, status string) error {
	return b.db.Transaction(func(tx *gorm.DB) error {
		// Canonical lock order (see stats_download.go): lock the parent upload row
		// before touching this file row or its usage counters, so completions and
		// removals of the same upload serialize on the upload row.
		if _, err := b.lockUploadRow(tx, file.UploadID); err != nil {
			return err
		}

		result := tx.Model(&common.File{}).Where(&common.File{ID: file.ID, Status: status}).Select(updatableFileColumns).Updates(file)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(1) {
			return fmt.Errorf("invalid file status")
		}

		return b.updateUsageForFileStatusTransition(tx, file, status, file.Status)
	})
}

// UpdateFileStatus update a file status in DB. oldStatus ensure the file status has not changed since loaded
func (b *Backend) UpdateFileStatus(file *common.File, oldStatus string, newStatus string) error {
	err := b.db.Transaction(func(tx *gorm.DB) error {
		// Canonical lock order (see stats_download.go): lock the parent upload row
		// before touching this file row or its usage counters, so transitions and
		// removals of the same upload serialize on the upload row.
		if _, err := b.lockUploadRow(tx, file.UploadID); err != nil {
			return err
		}

		result := tx.Model(&common.File{}).Where(&common.File{ID: file.ID, Status: oldStatus}).Update("status", newStatus)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(1) {
			return fmt.Errorf("%s file not found", oldStatus)
		}

		err := b.updateUsageForFileStatusTransition(tx, file, oldStatus, newStatus)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	file.Status = newStatus
	return nil
}

// RemoveFile change the file status to removed
// The file will then be deleted from the data backend by the server and the status changed to deleted.
func (b *Backend) RemoveFile(file *common.File) error {
	switch file.Status {
	case common.FileMissing, "":
		// Missing files were never uploaded, even partially it is safe to update the status to deleted directly
		return b.UpdateFileStatus(file, file.Status, common.FileDeleted)
	case common.FileUploaded, common.FileUploading:
		// Uploaded, Uploading files have been at least partially uploaded
		// by setting the status to Removed we mark the files as ready to be deleted from the Data backend
		// which will occur during the next cleaning cycle
		return b.UpdateFileStatus(file, file.Status, common.FileRemoved)
	//case common.FileRemoved, common.FileDeleted:
	//	return nil
	default:
		return nil
	}
}

// updateUsageForFileStatusTransition keeps usage counters in sync with the
// file status state machine. Only files in the uploaded state count toward
// current usage; lifetime usage is counted once, when a file first reaches a
// completed terminal state.
//
// Non-stream uploads complete by moving to FileUploaded, so they increment both
// current and lifetime counters on the transition into FileUploaded. Stream
// uploads are different: they stay FileUploading while data is being consumed
// and successful completion is represented by FileUploading -> FileDeleted, so
// that path increments lifetime counters only. Failed non-stream uploads can
// also move FileUploading -> FileDeleted during cleanup, but must not inflate
// lifetime stats.
func (b *Backend) updateUsageForFileStatusTransition(tx *gorm.DB, file *common.File, oldStatus string, newStatus string) error {
	if oldStatus == newStatus {
		return nil
	}

	if oldStatus == common.FileUploaded && newStatus != common.FileUploaded {
		// Leaving FileUploaded removes retained usage, but never rewrites
		// lifetime counters.
		return b.decrementUsageForUploadedFile(tx, file)
	}

	if oldStatus != common.FileUploaded && newStatus == common.FileUploaded {
		// Regular completed uploads become current retained files and lifetime
		// files at the same time. This is the live AddFile completion (the only
		// caller reaching this transition), so the fully received stream's
		// file.Size is exactly the wire bytes read off the client — record it as
		// uploaded-bytes ingress in this same fused transaction.
		return b.incrementUsageForCompletedFile(tx, file, true, file.Size)
	}

	if oldStatus == common.FileUploading && newStatus == common.FileDeleted {
		upload, err := b.lockUploadRow(tx, file.UploadID)
		if err != nil || upload == nil || !upload.Stream {
			return err
		}
		// Streamed files are never retained after completion, but a successful
		// stream still counts toward lifetime usage; file.Size is the wire bytes
		// received for the streamed transfer.
		return b.incrementUsageForCompletedFile(tx, file, false, file.Size)
	}

	return nil
}

// ForEachUploadFiles execute f for each file of the upload
func (b *Backend) ForEachUploadFiles(uploadID string, f func(file *common.File) error) (err error) {
	rows, err := b.db.Model(&common.File{}).Where(&common.File{UploadID: uploadID}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		file := &common.File{}
		err = b.db.ScanRows(rows, file)
		if err != nil {
			return err
		}
		err = f(file)
		if err != nil {
			return err
		}
	}

	return nil
}

// ForEachRemovedFile execute f for each file with the status "removed"
func (b *Backend) ForEachRemovedFile(f func(file *common.File) error) (err error) {
	rows, err := b.db.Model(&common.File{}).Where(&common.File{Status: common.FileRemoved}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		file := &common.File{}
		err = b.db.ScanRows(rows, file)
		if err != nil {
			return err
		}
		err = f(file)
		if err != nil {
			return err
		}
	}

	return nil
}

// CountUploadFiles count how many files have been added to an upload
func (b *Backend) CountUploadFiles(uploadID string) (count int, err error) {
	var c int64 // Gorm V2 requires int64 for counts

	err = b.db.Model(&common.File{}).Where(&common.File{UploadID: uploadID}).Count(&c).Error
	if err != nil {
		return -1, err
	}

	return int(c), nil
}

// ForEachFile execute f for every file in the database
func (b *Backend) ForEachFile(f func(file *common.File) error) (err error) {
	stmt := b.db.Model(&common.File{})

	rows, err := stmt.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		file := &common.File{}
		err = b.db.ScanRows(rows, file)
		if err != nil {
			return err
		}
		err = f(file)
		if err != nil {
			return err
		}
	}

	return nil
}
