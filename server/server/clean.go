package server

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/root-gg/plik/server/common"
)

/*
  Plik cleaning design :
    - Uploads and files can be removed from the server by two actions :
      - Manually from the : UI / API / Command line
      - Automatically when the upload TTL expires

    - When this happens :
      - The upload is only soft deleted from the metadata database
      - All the upload files status are updated to Removed (or directly to deleted if they have never been uploaded)
      - From this moment the upload and its files won't be accessible anymore

    - The background cleaning routine :
      - Is triggered periodically by a running Plik server with IsAutoClean true
      - Can be triggered manually from the CLI

      1 Mark expired uploads and files as removed and ready to be cleaned
      2 Deletes all the removed files from the data backend
      3 Purge (real delete) removed upload and files from the metadata backend
*/

// UploadsCleaningRoutine periodically remove expired uploads
func (ps *PlikServer) uploadsCleaningRoutine() {
	log := ps.config.NewLogger()
	for {
		ps.mu.Lock()
		done := ps.done
		ps.mu.Unlock()

		if done {
			break
		}
		// Sleep between 2 hours and 3 hours
		// This is a dirty trick to avoid frontends doing this at the same time
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(ps.cleaningRandomDelay)))
		randomSleep := r.Int64() + int64(ps.cleaningMinOffset)

		log.Infof("Will clean old uploads in %d seconds.", randomSleep)
		time.Sleep(time.Duration(randomSleep) * time.Second)
		log.Infof("Cleaning expired uploads...")

		ps.Clean()
	}
}

// Clean delete expired data and metadata
func (ps *PlikServer) Clean() {
	log := ps.config.NewLogger()

	// 1 - soft delete expired uploads
	removed, err := ps.metadataBackend.RemoveExpiredUploads()
	if removed > 0 {
		log.Infof("removed %d expired uploads", removed)
	}
	if err != nil {
		log.Warning(err.Error())
	}

	// 2 - delete removed files
	deleted, err := ps.PurgeDeletedFiles()
	if deleted > 0 {
		log.Infof("purged %d deleted files", deleted)
	}
	if err != nil {
		log.Warning(err.Error())
	}

	// 3 - purge deleted uploads

	purged, err := ps.metadataBackend.DeleteRemovedUploads()
	if purged > 0 {
		log.Infof("purged %d deleted uploads", purged)
	}
	if err != nil {
		log.Warning(err.Error())
	}

	// 4 - clean metadata database

	err = ps.metadataBackend.Clean()
	if err != nil {
		log.Warning(err.Error())
	}
}

// PurgeDeletedFiles delete "removed" files from the data backend
func (ps *PlikServer) PurgeDeletedFiles() (deleted int, err error) {
	log := ps.config.NewLogger()

	var errors []error
	f := func(file *common.File) (err error) {
		err = ps.dataBackend.RemoveFile(file)
		if err != nil {
			errors = append(errors, err)
			log.Warningf("unable to delete file %s/%s : %s, will retry", file.UploadID, file.ID, err)
			return nil
		}

		err = ps.metadataBackend.UpdateFileStatus(file, common.FileRemoved, common.FileDeleted)
		if err != nil {
			errors = append(errors, err)
			log.Warningf("unable to update deleted file %s/%s : %s, will retry", file.UploadID, file.ID, err)
			return nil
		}

		deleted++
		return nil
	}

	err = ps.metadataBackend.ForEachRemovedFile(f)
	if err != nil {
		return deleted, err
	}
	if len(errors) > 0 {
		return deleted, fmt.Errorf("unable to delete %d files", len(errors))
	}
	return deleted, nil
}
