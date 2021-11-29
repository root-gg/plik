package metadata

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/root-gg/utils"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/root-gg/plik/server/common"
)

// Config metadata backend configuration
type Config struct {
	Driver           string
	ConnectionString string
	EraseFirst       bool
	MaxOpenConns     int
	MaxIdleConns     int
	Debug            bool
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

	db *gorm.DB
}

// NewBackend instantiate a new File Data Backend
// from configuration passed as argument
func NewBackend(config *Config) (b *Backend, err error) {
	b = new(Backend)
	b.Config = config

	// Setup database logging
	var l logger.Interface
	if config.Debug {
		l = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,       // Disable color
			},
		)
	} else {
		l = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second,   // Slow SQL threshold
				LogLevel:                  logger.Silent, // Log level
				IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,         // Disable color
			},
		)
	}

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
	}

	// Open database connection
	b.db, err = gorm.Open(dial, &gorm.Config{Logger: l})
	if err != nil {
		return nil, fmt.Errorf("unable to open database : %s", err)
	}

	if config.Driver == "sqlite3" {
		err = b.db.Exec("PRAGMA journal_mode=WAL;").Error
		if err != nil {
			if err := b.Shutdown(); err != nil {
				b.db.Logger.Error(context.Background(), "Unable to shutdown metadata backend : %s", err)
			}
			return nil, fmt.Errorf("unable to set wal mode : %s", err)
		}

		err = b.db.Exec("PRAGMA foreign_keys = ON").Error
		if err != nil {
			if err := b.Shutdown(); err != nil {
				b.db.Logger.Error(context.Background(), "Unable to shutdown metadata backend : %s", err)
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

	// Initialize database schema
	err = b.initializeDB()
	if err != nil {
		if err := b.Shutdown(); err != nil {
			b.db.Logger.Error(context.Background(), "Unable to shutdown metadata backend : %s", err)
		}
		return nil, fmt.Errorf("unable to initialize DB : %s", err)
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
func (b *Backend) initializeDB() (err error) {
	m := gormigrate.New(b.db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// your migrations here
	})

	m.InitSchema(func(tx *gorm.DB) error {

		if b.Config.Driver == "mysql" {
			// Enable foreign keys
			tx = tx.Set("gorm:table_options", "ENGINE=InnoDB")
		}

		err := tx.AutoMigrate(
			&common.Upload{},
			&common.File{},
			&common.User{},
			&common.Token{},
			&common.Setting{},
		)

		return err
	})

	if err = m.Migrate(); err != nil {
		return fmt.Errorf("could not migrate: %v", err)
	}

	return nil
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
