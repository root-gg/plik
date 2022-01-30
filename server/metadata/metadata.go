package metadata

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/root-gg/logger"
	"github.com/root-gg/utils"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
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
func NewConfig(params map[string]interface{}) (config *Config) {
	config = new(Config)
	config.Driver = "sqlite3"
	config.ConnectionString = "plik.db"
	utils.Assign(config, params)
	return
}

// Backend object
type Backend struct {
	Config *Config

	log *logger.Logger
	db  *gorm.DB
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
		dial = sqlite.Open(config.ConnectionString)
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

	// For testing
	if config.EraseFirst {
		err = b.db.Migrator().DropTable("files", "uploads", "tokens", "users", "settings", "migrations")
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

// Initialize the metadata backend.
//  - Create or update the database schema if needed
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
			)

			return err
		})
	}

	if err = m.Migrate(); err != nil {
		return fmt.Errorf("could not migrate: %v", err)
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

// Shutdown the the metadata backend, close all connections to the database.
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
//  - Remove orphan files and tokens
func (b *Backend) Clean() error {
	return b.clean(b.db)
}

func (b *Backend) clean(tx *gorm.DB) error {
	if !tx.Migrator().HasTable("uploads") {
		// Empty database
		return nil
	}

	b.log.Infof("Cleaning up SQL database")

	result := tx.Exec("delete from files where upload_id not in (select id from uploads);")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		b.log.Warningf("deleted %d orphan files", result.RowsAffected)
	}

	result = tx.Exec("delete from tokens where user_id not in (select id from users);")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		b.log.Warningf("deleted %d orphan tokens", result.RowsAffected)
	}

	return nil
}
