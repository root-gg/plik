package metadata

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/root-gg/utils"
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
	}

	// Open database connection
	b.db, err = gorm.Open(dial, &gorm.Config{Logger: l})
	if err != nil {
		return nil, fmt.Errorf("unable to open database : %s", err)
	}

	if config.Driver == "sqlite3" {
		err = b.db.Exec("PRAGMA journal_mode=WAL;").Error
		if err != nil {
			_ = b.Shutdown()
			return nil, fmt.Errorf("unable to set wal mode : %s", err)
		}

		err = b.db.Exec("PRAGMA foreign_keys = ON").Error
		if err != nil {
			_ = b.Shutdown()
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

	err = b.initializeDB()
	if err != nil {
		return nil, fmt.Errorf("unable to initialize DB : %s", err)
	}

	return b, err
}

func (b *Backend) initializeDB() (err error) {
	m := gormigrate.New(b.db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// your migrations here
	})

	m.InitSchema(func(tx *gorm.DB) error {

		//if b.Config.Driver == "mysql" {
		//	// Enable foreign keys
		//	tx = tx.Set("gorm:table_options", "ENGINE=InnoDB")
		//}

		err := tx.AutoMigrate(
			&common.Upload{},
			&common.File{},
			&common.User{},
			&common.Token{},
			&common.Setting{},
		)
		if err != nil {
			return err
		}

		//if b.Config.Driver == "mysql" {
		//	err = tx.Model(&common.File{}).AddForeignKey("upload_id", "uploads(id)", "RESTRICT", "RESTRICT").Error
		//	if err != nil {
		//		return err
		//	}
		//	err = tx.Model(&common.Token{}).AddForeignKey("user_id", "users(id)", "RESTRICT", "RESTRICT").Error
		//	if err != nil {
		//		return err
		//	}
		//}

		// all other foreign keys...
		return nil
	})

	if err = m.Migrate(); err != nil {
		return fmt.Errorf("could not migrate: %v", err)
	}

	return nil
}

// Shutdown close the metadata backend
func (b *Backend) Shutdown() (err error) {

	// Close database connection if needed
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
