package metadata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/root-gg/logger"
	"github.com/root-gg/plik/server/common"
	"github.com/root-gg/utils"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormPrometheus "gorm.io/plugin/prometheus"
)

// Config metadata backend configuration
type Config struct {
	Driver             string
	ConnectionString   string
	EraseFirst         bool
	MaxOpenConns       int
	MaxIdleConns       int
	Debug              bool
	SlowQueryThreshold string // Duration string
	noMigrations       bool   // For testing
	migrationFilter    func([]*gormigrate.Migration) []*gormigrate.Migration
	disableSchemaInit  bool // For testing
}

// NewConfig instantiate a new default configuration
// and override it with configuration passed as argument
func NewConfig(params map[string]any) (config *Config) {
	config = new(Config)
	config.Driver = "sqlite3"
	config.ConnectionString = "plik.db"
	utils.Assign(config, params)
	return
}

// Backend object
type Backend struct {
	Config *Config

	log     *logger.Logger
	db      *gorm.DB
	dbStats *gormPrometheus.DBStats
}

// NewBackend instantiate a new File Data Backend
// from configuration passed as argument
func NewBackend(config *Config, log *logger.Logger) (b *Backend, err error) {
	b = new(Backend)
	b.Config = config

	// Prepare database connection depending on driver type
	var dial gorm.Dialector
	switch config.Driver {
	case "sqlite3":
		dial = sqlite.Open(sqliteConnectionString(config.ConnectionString))
	case "postgres":
		dial = postgres.Open(config.ConnectionString)
	case "mysql":
		dial = mysql.New(mysql.Config{
			DSN:                       config.ConnectionString,
			DefaultStringSize:         256,  // default size for string fields
			SkipInitializeWithVersion: true, // auto configure based on currently MySQL version
		})

	//case "sqlserver":
	//	dial = sqlserver.Open(config.ConnectionString)
	//
	// There is currently an issue with the reserved keyword user not being correctly escaped
	// "SELECT count(*) FROM "uploads" WHERE uploads.user == "user" AND "uploads"."deleted_at" IS NULL"
	//  -> returns : Incorrect syntax near the keyword 'user'
	// "SELECT count(*) FROM "uploads" WHERE uploads.[user] = "user" AND "uploads"."deleted_at" IS NULL"
	//  -> Would be OK
	// TODO investigate how the query is generated and maybe open issue in https://github.com/denisenkom/go-mssqldb ?
	default:
		return nil, fmt.Errorf("Invalid metadata backend driver : %s", config.Driver)
	}

	// Setup logging adaptor
	b.log = log
	gormLoggerAdapter := NewGormLoggerAdapter(log.Copy())

	if b.Config.Debug {
		// Display all Gorm log messages
		gormLoggerAdapter.logger.SetMinLevel(logger.DEBUG)
	} else {
		// Display only Gorm errors
		gormLoggerAdapter.logger.SetMinLevel(logger.WARNING)
	}

	// Set slow query threshold
	if config.SlowQueryThreshold != "" {
		duration, err := time.ParseDuration(config.SlowQueryThreshold)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse SlowQueryThreshold : %s", err)
		}
		gormLoggerAdapter.SlowQueryThreshold = duration
	}

	// Open database connection
	b.db, err = gorm.Open(dial, &gorm.Config{Logger: gormLoggerAdapter})
	if err != nil {
		return nil, fmt.Errorf("Unable to open database : %s", err)
	}

	if config.Driver == "sqlite3" {
		err = b.db.Exec("PRAGMA journal_mode=WAL;").Error
		if err != nil {
			if err := b.Shutdown(); err != nil {
				b.log.Criticalf("Unable to shutdown metadata backend : %s", err)
			}
			return nil, fmt.Errorf("unable to set wal mode : %s", err)
		}

		err = b.db.Exec("PRAGMA foreign_keys = ON").Error
		if err != nil {
			if err := b.Shutdown(); err != nil {
				b.log.Criticalf("Unable to shutdown metadata backend : %s", err)
			}
			return nil, fmt.Errorf("unable to enable foreign keys : %s", err)
		}
	}

	// Setup metrics
	dbMetrics := gormPrometheus.New(gormPrometheus.Config{})
	err = b.db.Use(dbMetrics)
	if err != nil {
		return nil, fmt.Errorf("unable to enable gorm metrics : %s", err)
	}
	b.dbStats = dbMetrics.DBStats

	// For testing
	if config.EraseFirst {
		err = b.db.Migrator().DropTable(
			"download_stats_daily",
			"upload_stats_daily",
			"usage_stats",
			"files",
			"uploads",
			"tokens",
			"users",
			"settings",
			"cli_auth_sessions",
			"migrations",
		)
		if err != nil {
			return nil, fmt.Errorf("unable to drop tables : %s", err)
		}
	}

	if !b.Config.noMigrations {
		// Initialize database schema
		err = b.initializeSchema()
		if err != nil {
			if err := b.Shutdown(); err != nil {
				b.db.Logger.Error(context.Background(), "Unable to shutdown metadata backend : %s", err)
			}
			return nil, fmt.Errorf("unable to initialize DB : %s", err)
		}
	}

	// Adjust max idle/open connection pool size
	err = b.adjustConnectionPoolParameters()
	if err != nil {
		return nil, err
	}

	return b, err
}

// sqliteConnectionString ensures the SQLite DSN opens transactions with the
// write lock held from BEGIN (`_txlock=immediate`) and waits on a busy database
// instead of erroring (`_busy_timeout`). Both are load-bearing because fused
// metadata+counter transactions are never retried on contention: several writers
// (RemoveUpload, UpdateFile/UpdateFileStatus) start by reading the parent upload
// row (lockUploadRow skips FOR UPDATE on SQLite) and then write. Under the
// default deferred BEGIN two such transactions can each hold a read snapshot and
// then deadlock trying to upgrade to a writer — SQLITE_BUSY that no busy_timeout
// can resolve. Beginning IMMEDIATE makes each transaction acquire the single WAL
// write lock up front, so writers serialize as ordered lock waits (bounded by
// busy_timeout) instead of deadlocking — the same "prevent, don't retry"
// guarantee the canonical lock order gives on PostgreSQL/MySQL. Existing
// parameters are preserved and never overridden.
func sqliteConnectionString(cs string) string {
	const defaultBusyTimeout = "5000"

	// Split off any existing query string so we only add missing parameters.
	base, query, hasQuery := strings.Cut(cs, "?")
	existing := map[string]bool{}
	if hasQuery {
		for param := range strings.SplitSeq(query, "&") {
			if key, _, ok := strings.Cut(param, "="); ok {
				existing[strings.TrimSpace(key)] = true
			}
		}
	}

	var params []string
	if query != "" {
		params = append(params, query)
	}
	if !existing["_txlock"] {
		params = append(params, "_txlock=immediate")
	}
	if !existing["_busy_timeout"] {
		params = append(params, "_busy_timeout="+defaultBusyTimeout)
	}

	if len(params) == 0 {
		return base
	}
	return base + "?" + strings.Join(params, "&")
}

// Initialize the metadata backend.
//   - Create or update the database schema if needed
func (b *Backend) initializeSchema() (err error) {
	m := gormigrate.New(b.db, gormigrate.DefaultOptions, b.getMigrations())

	if !b.Config.disableSchemaInit {
		// Skip migrations if initializing database for the first time
		m.InitSchema(func(tx *gorm.DB) error {
			b.log.Warningf("Initializing %s database", b.Config.Driver)

			err := b.setupTxForMigration(tx).AutoMigrate(
				&common.Upload{},
				&common.File{},
				&common.User{},
				&common.Token{},
				&common.Setting{},
				&common.CLIAuthSession{},
				&common.UsageStats{},
				&common.DownloadStatsDaily{},
				&common.UploadStatsDaily{},
			)

			return err
		})
	}

	if err = m.Migrate(); err != nil {
		return fmt.Errorf("could not migrate: %v", err)
	}

	if b.db.Migrator().HasTable(&common.UsageStats{}) {
		if err = b.ensureBaseUsageStatsRows(b.db); err != nil {
			return fmt.Errorf("could not initialize usage stats rows: %v", err)
		}
	}

	return nil
}

func (b *Backend) setupTxForMigration(tx *gorm.DB) *gorm.DB {
	if b.Config.Driver == "mysql" {
		// Enable foreign keys and set utf8 charset
		return tx.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	}

	return tx
}

// Adjust max idle/open connection pool size
func (b *Backend) adjustConnectionPoolParameters() (err error) {
	// Get generic "database/sql" database handle
	sqlDB, err := b.db.DB()
	if err != nil {
		return fmt.Errorf("unable to get SQL DB handle : %s", err)
	}

	if b.Config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(b.Config.MaxIdleConns)
	}

	if b.Config.MaxOpenConns > 0 {
		// Need at least a few because of https://github.com/mattn/go-sqlite3/issues/569
		sqlDB.SetMaxOpenConns(b.Config.MaxOpenConns)
	}

	return nil
}

// Shutdown the metadata backend, close all connections to the database.
func (b *Backend) Shutdown() (err error) {

	// Close database connection
	if b.db != nil {
		db, err := b.db.DB()
		if err != nil {
			return err
		}
		err = db.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// Clean metadata database
//   - Remove orphan files and tokens
func (b *Backend) Clean() (files int, tokens int, err error) {
	return b.clean(b.db)
}

func (b *Backend) clean(tx *gorm.DB) (files int, tokens int, err error) {
	if !tx.Migrator().HasTable("uploads") {
		// Empty database
		return 0, 0, nil
	}

	b.log.Infof("Cleaning up SQL database")

	result := tx.Exec("delete from files where upload_id not in (select id from uploads);")
	if result.Error != nil {
		return 0, 0, result.Error
	}
	if result.RowsAffected > 0 {
		b.log.Warningf("deleted %d orphan files", result.RowsAffected)
	}
	files = int(result.RowsAffected)

	result = tx.Exec("delete from tokens where user_id not in (select id from users);")
	if result.Error != nil {
		return 0, 0, result.Error
	}
	if result.RowsAffected > 0 {
		b.log.Warningf("deleted %d orphan tokens", result.RowsAffected)
	}
	tokens = int(result.RowsAffected)

	return files, tokens, nil
}

// GetMetricsCollectors return Gorm metrics
func (b *Backend) GetMetricsCollectors() []prometheus.Collector {
	if b.dbStats != nil {
		return b.dbStats.Collectors()
	}
	return nil
}

// reorderByRefs restores hydrated items into the order given by refs, a slice
// of keys taken from a first-phase query that did the sorting/pagination
// (cursor paginators sort on joined/computed columns but still need to return
// full models, so callers page a lightweight ref slice, hydrate full rows by
// key with a second query, then call this to put them back in the ref order).
// A ref whose key is missing from items — hydrate did not return it, e.g. a
// row removed between the two phases — is silently dropped.
func reorderByRefs[K comparable, T any](refs []K, items []T, key func(T) K) []T {
	byKey := make(map[K]T, len(items))
	for _, item := range items {
		byKey[key(item)] = item
	}

	ordered := items[:0]
	for _, ref := range refs {
		if item, ok := byKey[ref]; ok {
			ordered = append(ordered, item)
		}
	}
	return ordered
}
