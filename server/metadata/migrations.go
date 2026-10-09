package metadata

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/root-gg/plik/server/common"
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
			ID: "0009-file-is-text",
			Migrate: func(tx *gorm.DB) error {
				type File struct {
					IsText bool `json:"isText"`
				}

				b.log.Warning("Applying database migration 0009-file-is-text")
				return b.setupTxForMigration(tx).AutoMigrate(&File{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			ID: "0010-user-language",
			Migrate: func(tx *gorm.DB) error {
				type User struct {
					Language string `json:"language,omitempty"`
				}

				b.log.Warning("Applying database migration 0010-user-language")
				return b.setupTxForMigration(tx).AutoMigrate(&User{})
			},
			Rollback: func(tx *gorm.DB) error {
				b.log.Criticalf("Something went wrong. Please check database status manually")
				return nil
			},
		}, {
			// 0011-stats introduces the usage counter ledger and daily rollups.
			ID: "0011-stats",
			Migrate: func(tx *gorm.DB) error {
				type Upload struct {
					DownloadCount    int64      `json:"downloadCount" gorm:"index:idx_upload_download_count"`
					LastDownloadedAt *time.Time `json:"lastDownloadedAt,omitempty"`
					DownloadedBytes  int64      `json:"downloadedBytes" gorm:"index:idx_upload_downloaded_bytes"`
				}

				type File struct {
					UploadID         string     `json:"-" gorm:"size:256;index:idx_file_upload_id;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`
					DownloadCount    int64      `json:"downloadCount"`
					LastDownloadedAt *time.Time `json:"lastDownloadedAt,omitempty"`
				}

				b.log.Warning("Applying database migration 0011-stats")

				// AutoMigrate creates the new stats table, adds only the upload/file
				// columns represented by the narrow local structs above, and indexes
				// files.upload_id before the backfill scans file rows by upload scope.
				err := b.setupTxForMigration(tx).AutoMigrate(&Upload{}, &File{}, &common.UsageStats{}, &common.DownloadStatsDaily{}, &common.UploadStatsDaily{})
				if err != nil {
					return err
				}

				// gormigrate runs with UseTransaction=false, so every statement below
				// autocommits and the backfill is not atomic. If a previous attempt
				// died mid-backfill it left partial usage_stats rows behind without
				// recording the migration id, so this migration re-runs on the next
				// start. Wipe the table first — it is owned by this migration — so a
				// re-run recomputes correct counters instead of failing on a duplicate
				// (user_id, token) primary key.
				err = tx.Where("1 = 1").Delete(&common.UsageStats{}).Error
				if err != nil {
					return err
				}

				// Use one timestamp as the "stats since" marker for every backfilled
				// row. The counters describe retained metadata at migration time plus
				// future mutations; data already purged before this point is not
				// reconstructable.
				startedAt := time.Now()

				var users []*common.User
				err = tx.Model(&common.User{}).Find(&users).Error
				if err != nil {
					return err
				}

				// Seed one usage_stats row per existing authenticated user.
				// Current counters use only retained uploads; lifetime counters include
				// soft-deleted uploads/files that still exist in metadata. lifetime_users
				// is a uniform per-user counter (1 per user row); server lifetime_users
				// is their sum, so it counts every user that exists at migration time —
				// users deleted before the migration ran cannot be recovered.
				for _, user := range users {
					userCurrentScope := backfillScopeUploadIDQuery(tx, "user", user.ID, false)
					userLifetimeScope := backfillScopeUploadIDQuery(tx, "user", user.ID, true)
					userCurrentUploads, err := backfillScopeUploadCount(tx, "user", user.ID, false)
					if err != nil {
						return err
					}
					userLifetimeUploads, err := backfillScopeUploadCount(tx, "user", user.ID, true)
					if err != nil {
						return err
					}
					err = backfillUsageStatsRow(tx, user.ID, "", userCurrentScope, userCurrentUploads, userLifetimeScope, userLifetimeUploads, 1, startedAt)
					if err != nil {
						return err
					}
				}

				// Anonymous usage is tracked through a synthetic user row so API code
				// can read it through the same counter-backed path as real users.
				anonymousCurrentScope := backfillScopeUploadIDQuery(tx, "user", "", false)
				anonymousLifetimeScope := backfillScopeUploadIDQuery(tx, "user", "", true)
				anonymousCurrentUploads, err := backfillScopeUploadCount(tx, "user", "", false)
				if err != nil {
					return err
				}
				anonymousLifetimeUploads, err := backfillScopeUploadCount(tx, "user", "", true)
				if err != nil {
					return err
				}
				err = backfillUsageStatsRow(tx, common.AnonymousUserUsageStatsID, "", anonymousCurrentScope, anonymousCurrentUploads, anonymousLifetimeScope, anonymousLifetimeUploads, 0, startedAt)
				if err != nil {
					return err
				}

				// There is no server row: server totals are summed on read over the
				// token='' rows created above (users + anonymous + the tombstone below).
				// In place of the old server backfill, create the deleted-user tombstone
				// with zero counters. It receives DeleteUser folds so server lifetime
				// totals survive account deletion, and anchors the server startedAt MIN.
				err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&common.UsageStats{
					UserID:    common.DeletedUserUsageStatsID,
					StartedAt: startedAt,
				}).Error
				if err != nil {
					return err
				}

				// Token rows reuse the same stats shape but carry both owner user id
				// and token string as their scope key.
				var tokens []*common.Token
				err = tx.Model(&common.Token{}).Find(&tokens).Error
				if err != nil {
					return err
				}
				for _, token := range tokens {
					tokenCurrentScope := backfillScopeUploadIDQuery(tx, "token", token.Token, false)
					tokenLifetimeScope := backfillScopeUploadIDQuery(tx, "token", token.Token, true)
					tokenCurrentUploads, err := backfillScopeUploadCount(tx, "token", token.Token, false)
					if err != nil {
						return err
					}
					tokenLifetimeUploads, err := backfillScopeUploadCount(tx, "token", token.Token, true)
					if err != nil {
						return err
					}
					err = backfillUsageStatsRow(tx, token.UserID, token.Token, tokenCurrentScope, tokenCurrentUploads, tokenLifetimeScope, tokenLifetimeUploads, 0, startedAt)
					if err != nil {
						return err
					}
				}

				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				// usage_stats, download_stats_daily and upload_stats_daily are created
				// by this migration, so rolling back simply drops them. The upload/file
				// download columns added above are left in place; they are harmless
				// without the tables.
				b.log.Warning("Rolling back database migration 0011-stats")
				return b.setupTxForMigration(tx).Migrator().DropTable(&common.UsageStats{}, &common.DownloadStatsDaily{}, &common.UploadStatsDaily{})
			},
		},
	}

	if b.Config.migrationFilter != nil {
		migrations = b.Config.migrationFilter(migrations)
	}

	return migrations
}

// backfillScopeUploadIDQuery returns a subquery selecting the upload ids that
// define a user, anonymous, or token scope. column is "user" or "token" and
// value is the scope key (userID, "" for anonymous, or the token string). The
// subquery is fed straight into the aggregate helpers so upload ids stay in the
// database instead of being expanded into an IN-list of bind parameters, which
// would overflow SQLite's 32,766 / PostgreSQL's 65,535 variable limits on large
// scopes. unscoped=false means current retained uploads; unscoped=true includes
// soft-deleted uploads for lifetime counters.
func backfillScopeUploadIDQuery(tx *gorm.DB, column string, value string, unscoped bool) *gorm.DB {
	stmt := tx.Model(&common.Upload{})
	if unscoped {
		stmt = stmt.Unscoped()
	}
	return stmt.Where(clause.Eq{Column: clause.Column{Name: column}, Value: value}).Select("id")
}

// backfillScopeUploadCount counts the uploads in the same scope as
// backfillScopeUploadIDQuery without loading any upload id into memory.
func backfillScopeUploadCount(tx *gorm.DB, column string, value string, unscoped bool) (uploads int, err error) {
	stmt := tx.Model(&common.Upload{})
	if unscoped {
		stmt = stmt.Unscoped()
	}
	var count int64
	err = stmt.Where(clause.Eq{Column: clause.Column{Name: column}, Value: value}).Count(&count).Error
	return int(count), err
}

func backfillUsageStatsRow(tx *gorm.DB, userID string, token string, currentUploadScope any, currentUploads int, lifetimeUploadScope any, lifetimeUploads int, lifetimeUsers int, startedAt time.Time) error {
	currentFiles, currentSize, err := backfillFileStats(tx, currentUploadScope, []string{common.FileUploaded})
	if err != nil {
		return err
	}
	lifetimeFiles, lifetimeSize, err := backfillFileStats(tx, lifetimeUploadScope, []string{common.FileUploaded, common.FileRemoved, common.FileDeleted})
	if err != nil {
		return err
	}
	// The four aggregates below share one delta: each touches a disjoint set of
	// counter fields (current vs. lifetime feature/TTL buckets, current vs.
	// lifetime file-size buckets), so accumulating them in place needs no merge
	// step.
	var delta usageDelta
	err = addUploadFeatureStatsForUploads(tx, &delta, currentUploadScope, 1, 0)
	if err != nil {
		return err
	}
	err = addUploadFeatureStatsForUploads(tx, &delta, lifetimeUploadScope, 0, 1)
	if err != nil {
		return err
	}
	err = addFileSizeStatsForUploads(tx, &delta, currentUploadScope, []string{common.FileUploaded}, 1, 0)
	if err != nil {
		return err
	}
	err = addFileSizeStatsForUploads(tx, &delta, lifetimeUploadScope, []string{common.FileUploaded, common.FileRemoved, common.FileDeleted}, 0, 1)
	if err != nil {
		return err
	}

	downloads, err := backfillDownloadStatsForUploads(tx, lifetimeUploadScope)
	if err != nil {
		return err
	}
	stats := usageStatsFromDelta(userID, token, delta, startedAt)
	stats.CurrentUploads = currentUploads
	stats.CurrentFiles = currentFiles
	stats.CurrentSize = currentSize
	stats.LifetimeUploads = lifetimeUploads
	stats.LifetimeFiles = lifetimeFiles
	stats.LifetimeSize = lifetimeSize
	stats.Downloads = downloads
	stats.LifetimeUsers = lifetimeUsers
	stats.LastUploadAt = backfillLastUploadAtForUploads(tx, lifetimeUploadScope)
	return tx.Create(stats).Error
}

// usageStatsFromDelta builds a backfill row from the aggregate feature/TTL/
// file-size delta. The counter columns are populated from the shared counter
// registry (stats_counters.go); backfillUsageStatsRow overwrites the core
// current/lifetime totals it computes separately (uploads, files, size,
// downloads, users, last upload).
func usageStatsFromDelta(userID string, token string, delta usageDelta, startedAt time.Time) *common.UsageStats {
	stats := &common.UsageStats{
		UserID:    userID,
		Token:     token,
		StartedAt: startedAt,
	}
	applyDeltaToUsageStats(stats, &delta)
	return stats
}

func backfillDownloadStatsForUploads(tx *gorm.DB, uploadIDs any) (downloads int64, err error) {
	err = tx.Unscoped().Model(&common.Upload{}).
		Select("coalesce(sum(download_count),0)").
		Where("id IN (?)", uploadIDs).
		Row().
		Scan(&downloads)
	return downloads, err
}

func backfillLastUploadAtForUploads(tx *gorm.DB, uploadIDs any) *time.Time {
	var upload common.Upload
	err := tx.Unscoped().
		Model(&common.Upload{}).
		Select("created_at").
		Where("id IN (?)", uploadIDs).
		Order("created_at DESC").
		Take(&upload).Error
	if err != nil {
		return nil
	}
	return &upload.CreatedAt
}

// backfillFileStats aggregates file counters for a known upload set. Callers
// choose the accepted statuses to distinguish current counters from lifetime
// counters that include removed/deleted files still present in metadata.
// uploadIDs is a GORM subquery selecting the scope's upload ids.
func backfillFileStats(tx *gorm.DB, uploadIDs any, statuses []string) (files int, size int64, err error) {
	err = tx.Model(&common.File{}).
		Select("count(files.id), coalesce(sum(size),0)").
		Where("upload_id IN (?)", uploadIDs).
		Where("status IN ?", statuses).
		Row().
		Scan(&files, &size)
	return files, size, err
}
