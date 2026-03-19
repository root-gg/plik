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

				_, _, err := b.clean(tx)
				if err != nil {
					return err
				}

				b.log.Warning("Applying database migration 0001-initial")
				return b.setupTxForMigration(tx).AutoMigrate(&Upload{}, &File{}, &User{}, &Token{}, &Setting{})
			},
			Rollback: func(tx *gorm.DB) error {
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

				_, _, err := b.clean(tx)
				if err != nil {
					return err
				}

				b.log.Warning("Applying database migration 0002-user-limits")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0003-extend-ttl",
			Migrate: func(tx *gorm.DB) error {
				type Upload struct {
					ExtendTTL bool `json:"extend_ttl"`
				}

				_, _, err := b.clean(tx)
				if err != nil {
					return err
				}

				b.log.Warning("Applying database migration 0003-extend-ttl")
				return b.setupTxForMigration(tx).AutoMigrate(&Upload{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0004-max-user-size",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					MaxUserSize int64 `json:"maxUserSize"`
				}

				_, _, err := b.clean(tx)
				if err != nil {
					return err
				}

				b.log.Warning("Applying database migration 0004-user-max-user-size")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0005-cli-auth-sessions",
			Migrate: func(tx *gorm.DB) error {
				type CLIAuthSession struct {
					Code      string `gorm:"primaryKey;size:16"`
					Secret    string `gorm:"size:64"`
					Status    string `gorm:"size:16;default:pending"`
					Token     string `gorm:"size:64"`
					CreatedAt time.Time
					ExpiresAt time.Time `gorm:"index"`
				}

				b.log.Warning("Applying database migration 0005-cli-auth-sessions")
				return b.setupTxForMigration(tx).AutoMigrate(&CLIAuthSession{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0006-user-profile-picture",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					ProfilePicture string `json:"profilePicture,omitempty"`
				}

				b.log.Warning("Applying database migration 0006-user-profile-picture")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0007-upload-e2ee",
			Migrate: func(tx *gorm.DB) error {
				type Upload struct {
					E2EE string `json:"e2ee,omitempty" gorm:"column:e2ee"`
				}

				b.log.Warning("Applying database migration 0007-upload-e2ee")
				return b.setupTxForMigration(tx).AutoMigrate(&Upload{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0008-user-theme",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					Theme string `json:"theme,omitempty"`
				}

				b.log.Warning("Applying database migration 0008-user-theme")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0009-user-language",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					Language string `json:"language,omitempty"`
				}

				b.log.Warning("Applying database migration 0009-user-language")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
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
