package metadata

import (
	"fmt"

	"github.com/root-gg/plik/server/common"
)

// MigrateOptions controls the behavior of Migrate
type MigrateOptions struct {
	IgnoreErrors bool
}

// MigrateStats holds counters returned by Migrate
type MigrateStats struct {
	Users         int
	UserErrors    int
	Tokens        int
	TokenErrors   int
	Uploads       int
	UploadErrors  int
	Files         int
	FileErrors    int
	Settings      int
	SettingErrors int
}

// Migrate copies all metadata from src to dst backend.
// Order: users → tokens → uploads (including soft-deleted) → files → settings.
// CLI auth sessions are excluded (they are ephemeral).
func Migrate(src *Backend, dst *Backend, options *MigrateOptions) (stats MigrateStats, err error) {
	if options == nil {
		options = &MigrateOptions{}
	}

	handleErr := func(label string, e error) error {
		if e == nil {
			return nil
		}
		if options.IgnoreErrors {
			src.log.Warningf("error migrating %s: %s", label, e)
			return nil
		}
		return e
	}

	// --- Users ---
	err = src.ForEachUsers(func(user *common.User) error {
		e := dst.CreateUser(user)
		if e != nil {
			stats.UserErrors++
			return handleErr(fmt.Sprintf("user %s", user.ID), e)
		}
		stats.Users++
		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("error iterating users: %w", err)
	}
	src.log.Infof("migrated %d/%d users", stats.Users, stats.Users+stats.UserErrors)

	// --- Tokens ---
	err = src.ForEachToken(func(token *common.Token) error {
		e := dst.CreateToken(token)
		if e != nil {
			stats.TokenErrors++
			return handleErr(fmt.Sprintf("token %s", token.Token), e)
		}
		stats.Tokens++
		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("error iterating tokens: %w", err)
	}
	src.log.Infof("migrated %d/%d tokens", stats.Tokens, stats.Tokens+stats.TokenErrors)

	// --- Uploads (including soft-deleted, to preserve FK integrity with files) ---
	err = src.ForEachUploadUnscoped(func(upload *common.Upload) error {
		e := dst.CreateUpload(upload)
		if e != nil {
			stats.UploadErrors++
			return handleErr(fmt.Sprintf("upload %s", upload.ID), e)
		}
		stats.Uploads++
		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("error iterating uploads: %w", err)
	}
	src.log.Infof("migrated %d/%d uploads", stats.Uploads, stats.Uploads+stats.UploadErrors)

	// --- Files ---
	err = src.ForEachFile(func(file *common.File) error {
		e := dst.CreateFile(file)
		if e != nil {
			stats.FileErrors++
			return handleErr(fmt.Sprintf("file %s", file.ID), e)
		}
		stats.Files++
		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("error iterating files: %w", err)
	}
	src.log.Infof("migrated %d/%d files", stats.Files, stats.Files+stats.FileErrors)

	// --- Settings ---
	err = src.ForEachSetting(func(setting *common.Setting) error {
		e := dst.CreateSetting(setting)
		if e != nil {
			stats.SettingErrors++
			return handleErr(fmt.Sprintf("setting %s", setting.Key), e)
		}
		stats.Settings++
		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("error iterating settings: %w", err)
	}
	src.log.Infof("migrated %d/%d settings", stats.Settings, stats.Settings+stats.SettingErrors)

	return stats, nil
}
