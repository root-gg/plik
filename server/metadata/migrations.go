package metadata

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func (b *Backend) getMigrations() []*gormigrate.Migration {
	migrations := []*gormigrate.Migration{
		{
			ID: "0001-initial",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					ID       string `json:"id"`
					UploadID string `json:"-" gorm:"size:256;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
					Name     string `json:"fileName"`

					Status string `json:"status"`

					Md5       string `json:"fileMd5"`
					Type      string `json:"fileType"`
					Size      int64  `json:"fileSize"`
					Reference string `json:"reference"`

					BackendDetails string `json:"-"`

					CreatedAt time.Time `json:"createdAt"`
				}

				type Upload struct {
					ID  string `json:"id"`
					TTL int    `json:"ttl"`

					DownloadDomain string `json:"downloadDomain" gorm:"-"`
					RemoteIP       string `json:"uploadIp,omitempty"`
					Comments       string `json:"comments"`

					Files []*File `json:"files"`

					UploadToken string `json:"uploadToken,omitempty"`
					User        string `json:"user,omitempty" gorm:"index:idx_upload_user"`
					Token       string `json:"token,omitempty" gorm:"index:idx_upload_user_token"`

					IsAdmin bool `json:"admin" gorm:"-"`

					Stream    bool `json:"stream"`
					OneShot   bool `json:"oneShot"`
					Removable bool `json:"removable"`

					ProtectedByPassword bool   `json:"protectedByPassword"`
					Login               string `json:"login,omitempty"`
					Password            string `json:"password,omitempty"`

					CreatedAt time.Time      `json:"createdAt"`
					DeletedAt gorm.DeletedAt `json:"-" gorm:"index:idx_upload_deleted_at"`
					ExpireAt  *time.Time     `json:"expireAt" gorm:"index:idx_upload_expire_at"`
				}

				type Token struct {
					Token   string `json:"token" gorm:"primary_key"`
					Comment string `json:"comment,omitempty"`

					UserID string `json:"-" gorm:"size:256;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`

					CreatedAt time.Time `json:"createdAt"`
				}

				type User struct {
					ID       string `json:"id,omitempty"`
					Provider string `json:"provider"`
					Login    string `json:"login,omitempty"`
					Password string `json:"-"`
					Name     string `json:"name,omitempty"`
					Email    string `json:"email,omitempty"`
					IsAdmin  bool   `json:"admin"`

					Tokens []*Token `json:"tokens,omitempty"`

					CreatedAt time.Time `json:"createdAt"`
				}

				type Setting struct {
					Key   string `gorm:"primary_key"`
					Value string
				}

				err := b.clean(tx)
				if err != nil {
					return err
				}

				b.log.Warning("Applying database migration 0001-initial")
				return b.setupTxForMigration(tx).AutoMigrate(&Upload{}, &File{}, &User{}, &Token{}, &Setting{})
			},
			Rollback: func(tx *gorm.DB) error {
				// return tx.Migrator().DropTable("uploads", "files", "users", "tokens", "settings")
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0002-user-limits",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					MaxFileSize int64 `json:"maxFileSize"`
					MaxTTL      int   `json:"maxTTL"`
				}

				err := b.clean(tx)
				if err != nil {
					return err
				}

				b.log.Warning("Applying database migration 0002-user-limits")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
				// return tx.Migrator().DropTable("uploads", "files", "users", "tokens", "settings")
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		},
	}

	if b.Config.migrationFilter != nil {
		migrations = b.Config.migrationFilter(migrations)
	}

	return migrations
}
