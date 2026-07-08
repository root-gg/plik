package metadata

import (
	"encoding/gob"
	"io"
	"os"

	"github.com/golang/snappy"

	"github.com/root-gg/plik/server/common"
)

type metadataType int

const (
	metadataTypeUpload metadataType = iota
	metadataTypeFile
	metadataTypeUser
	metadataTypeToken
	metadataTypeSetting
	metadataTypeDownloadStatsDaily
	metadataTypeUploadStatsDaily
	// metadataTypeUsageStatsTombstone carries the ("__deleted__","") usage row.
	// It is appended last so its numeric value stays stable and old exports that
	// predate it simply lack the record. It is the only usage_stats row exported
	// as a record: user/token/anonymous rows are rebuilt from imported metadata,
	// but a deleted user's uploads are gone, so its folded counters have no other
	// rebuild source.
	metadataTypeUsageStatsTombstone
)

type object struct {
	Type   metadataType
	Object any
}

type exporter struct {
	writer     io.WriteCloser
	compressor *snappy.Writer
	encoder    *gob.Encoder
}

func newExporter(path string) (e *exporter, err error) {
	e = &exporter{}

	// Open file for writing
	e.writer, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}

	// Snappy compressor
	e.compressor = snappy.NewBufferedWriter(e.writer)

	// Gob encoder
	gob.Register(&common.Upload{})
	gob.Register(&common.File{})
	gob.Register(&common.User{})
	gob.Register(&common.Token{})
	gob.Register(&common.Setting{})
	gob.Register(&common.DownloadStatsDaily{})
	gob.Register(&common.UploadStatsDaily{})
	gob.Register(&common.UsageStats{})
	e.encoder = gob.NewEncoder(e.compressor)

	return e, nil
}

func (e *exporter) addUpload(upload *common.Upload) (err error) {
	obj := &object{Type: metadataTypeUpload, Object: upload}
	return e.encoder.Encode(obj)
}

func (e *exporter) addFile(file *common.File) (err error) {
	obj := &object{Type: metadataTypeFile, Object: file}
	return e.encoder.Encode(obj)
}

func (e *exporter) addUser(user *common.User) (err error) {
	obj := &object{Type: metadataTypeUser, Object: user}
	return e.encoder.Encode(obj)
}

func (e *exporter) addToken(token *common.Token) (err error) {
	obj := &object{Type: metadataTypeToken, Object: token}
	return e.encoder.Encode(obj)
}

func (e *exporter) addSetting(setting *common.Setting) (err error) {
	obj := &object{Type: metadataTypeSetting, Object: setting}
	return e.encoder.Encode(obj)
}

func (e *exporter) addDownloadStatsDaily(stats *common.DownloadStatsDaily) (err error) {
	obj := &object{Type: metadataTypeDownloadStatsDaily, Object: stats}
	return e.encoder.Encode(obj)
}

func (e *exporter) addUploadStatsDaily(stats *common.UploadStatsDaily) (err error) {
	obj := &object{Type: metadataTypeUploadStatsDaily, Object: stats}
	return e.encoder.Encode(obj)
}

func (e *exporter) addUsageStatsTombstone(stats *common.UsageStats) (err error) {
	obj := &object{Type: metadataTypeUsageStatsTombstone, Object: stats}
	return e.encoder.Encode(obj)
}

func (e *exporter) close() (err error) {
	err = e.compressor.Close()
	if err != nil {
		return err
	}
	err = e.writer.Close()
	if err != nil {
		return err
	}
	return nil
}

// Export exports all metadata from the backend to a compressed binary file
func (b *Backend) Export(path string) (err error) {
	e, err := newExporter(path)
	if err != nil {
		return err
	}

	defer func() {
		closeErr := e.close()
		if err == nil {
			err = closeErr
		}
	}()

	count := 0
	err = b.ForEachUsers(func(user *common.User) error {
		count++
		return e.addUser(user)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d users", count)

	count = 0
	err = b.ForEachToken(func(token *common.Token) error {
		count++
		return e.addToken(token)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d tokens", count)

	count = 0
	// Need to export "soft deleted" uploads too else some removed/deleted files will have broken foreign keys
	err = b.ForEachUploadUnscoped(func(upload *common.Upload) error {
		count++
		return e.addUpload(upload)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d uploads", count)

	count = 0
	err = b.ForEachFile(func(file *common.File) error {
		count++
		return e.addFile(file)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d files", count)

	count = 0
	err = b.ForEachSetting(func(setting *common.Setting) error {
		count++
		return e.addSetting(setting)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d settings", count)

	count = 0
	err = b.ForEachDownloadStatsDaily(func(stats *common.DownloadStatsDaily) error {
		count++
		return e.addDownloadStatsDaily(stats)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d daily download stats", count)

	count = 0
	err = b.ForEachUploadStatsDaily(func(stats *common.UploadStatsDaily) error {
		count++
		return e.addUploadStatsDaily(stats)
	})
	if err != nil {
		return err
	}
	b.log.Infof("exported %d daily upload stats", count)

	// The deleted-user tombstone is the only usage_stats row not rebuildable from
	// imported metadata, so it is exported as its own record.
	tombstone, err := b.GetDeletedUsageTombstone()
	if err != nil {
		return err
	}
	if tombstone != nil {
		err = e.addUsageStatsTombstone(tombstone)
		if err != nil {
			return err
		}
		b.log.Infof("exported deleted-user usage tombstone")
	}

	return nil
}
