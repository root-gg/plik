package metadata

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateSetting create a new setting in DB
func (b *Backend) CreateSetting(setting *common.Setting) (err error) {
	return b.db.Create(setting).Error
}

// GetSetting get a setting from DB
func (b *Backend) GetSetting(key string) (setting *common.Setting, err error) {
	setting = &common.Setting{}

	err = b.db.Take(setting, &common.Setting{Key: key}).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return setting, nil
}

// UpdateSetting update a setting in DB
func (b *Backend) UpdateSetting(key string, oldValue string, newValue string) (err error) {
	result := b.db.Model(&common.Setting{}).Where(&common.Setting{Key: key, Value: oldValue}).Update("value", newValue)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(1) {
		return fmt.Errorf("setting not found")
	}

	return nil
}

// DeleteSetting delete a setting from DB
func (b *Backend) DeleteSetting(key string) (err error) {
	return b.db.Delete(&common.Setting{Key: key}).Error
}

// ForEachSetting execute f for every setting in the database
func (b *Backend) ForEachSetting(f func(setting *common.Setting) error) (err error) {
	rows, err := b.db.Model(&common.Setting{}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		setting := &common.Setting{}
		err = b.db.ScanRows(rows, setting)
		if err != nil {
			return err
		}
		err = f(setting)
		if err != nil {
			return err
		}
	}

	return nil
}
