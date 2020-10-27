package metadata

import (
	"time"

	"github.com/jinzhu/gorm"
	"gopkg.in/gormigrate.v1"
)

var migrations = []*gormigrate.Migration{
	{
		ID: "0001-initial",
		Migrate: func(tx *gorm.DB) error {
			// Initial database schema
			type File struct {
				ID       string `json:"id"`
				UploadID string `json:"-"  gorm:"type:varchar(255) REFERENCES uploads(id) ON UPDATE RESTRICT ON DELETE RESTRICT"`
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

				DownloadDomain string `json:"downloadDomain"`
				RemoteIP       string `json:"uploadIp,omitempty"`
				Comments       string `json:"comments"`

				Files []*File `json:"files"`

				UploadToken string `json:"uploadToken,omitempty"`
				User        string `json:"user,omitempty" gorm:"index:idx_upload_user"`
				Token       string `json:"token,omitempty" gorm:"index:idx_upload_user_token"`
				IsAdmin     bool   `json:"admin"`

				Stream    bool `json:"stream"`
				OneShot   bool `json:"oneShot"`
				Removable bool `json:"removable"`

				ProtectedByPassword bool   `json:"protectedByPassword"`
				Login               string `json:"login,omitempty"`
				Password            string `json:"password,omitempty"`

				CreatedAt time.Time  `json:"createdAt"`
				DeletedAt *time.Time `json:"-" gorm:"index:idx_upload_deleted_at"`
				ExpireAt  *time.Time `json:"expireAt" gorm:"index:idx_upload_expire_at"`
			}

			type Token struct {
				Token   string `json:"token" gorm:"primary_key"`
				Comment string `json:"comment,omitempty"`

				UserID string `json:"-" gorm:"type:varchar(255) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE CASCADE"`

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

			return tx.AutoMigrate(
				&Upload{},
				&File{},
				&User{},
				&Token{},
				&Setting{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("uploads", "files", "users", "tokens", "settings").Error
		},
	},
	{
		ID: "0002-user-verification",
		Migrate: func(tx *gorm.DB) error {
			type Token struct {
				Token   string `json:"token" gorm:"primary_key"`
				Comment string `json:"comment,omitempty"`

				UserID string `json:"-" gorm:"type:varchar(255) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE CASCADE"`

				CreatedAt time.Time `json:"createdAt"`
			}

			type User struct {
				ID               string `json:"id,omitempty"`
				Provider         string `json:"provider"`
				Login            string `json:"login,omitempty"`
				Password         string `json:"-"`
				Name             string `json:"name,omitempty"`
				Email            string `json:"email,omitempty" gorm:"index:idx_user_email"`
				IsAdmin          bool   `json:"admin"`
				VerificationCode string `json:"-"`
				Verified         bool   `json:"verified"`

				Tokens []*Token `json:"tokens,omitempty"`

				CreatedAt time.Time `json:"createdAt"`
			}

			return tx.AutoMigrate(&User{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			// TODO implement correct rollback strategy
			//return tx.DropTable("users", "tokens").Error
			return nil
		},
	},
	{
		ID: "0003-create-invite-table",
		Migrate: func(tx *gorm.DB) error {
			type Invite struct {
				ID     string  `json:"id,omitempty"`
				Issuer *string `json:"-" gorm:"type:varchar(255) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE CASCADE;index:idx_invite_issuer"`
				Email  string  `json:"email,omitempty"`

				Admin    bool `json:"admin"`
				Verified bool `json:"verified"`

				ExpireAt  *time.Time `json:"expireAt" gorm:"index:idx_invite_expire_at"`
				CreatedAt time.Time  `json:"createdAt"`
			}
			return tx.AutoMigrate(&Invite{}).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.DropTable("invites").Error
		},
	},
}
