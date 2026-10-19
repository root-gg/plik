# Gormigrate

[![Latest Release](https://img.shields.io/github/release/go-gormigrate/gormigrate.svg)](https://github.com/go-gormigrate/gormigrate/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-gormigrate/gormigrate/v2.svg)](https://pkg.go.dev/github.com/go-gormigrate/gormigrate/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-gormigrate/gormigrate/v2)](https://goreportcard.com/report/github.com/go-gormigrate/gormigrate/v2)
[![CI | Lint](https://github.com/go-gormigrate/gormigrate/actions/workflows/lint.yml/badge.svg)](https://github.com/go-gormigrate/gormigrate/actions)
[![CI | Test](https://github.com/go-gormigrate/gormigrate/actions/workflows/integration-test.yml/badge.svg)](https://github.com/go-gormigrate/gormigrate/actions)

Gormigrate is a minimalistic migration helper for [Gorm](http://gorm.io).
Gorm already has useful [migrate functions](https://gorm.io/docs/migration.html), just misses
proper schema versioning and migration rollback support.

> IMPORTANT: If you need support to Gorm v1 (which uses
> `github.com/jinzhu/gorm` as its import path), please import Gormigrate by
> using the `gopkg.in/gormigrate.v1` import path.
>
> The current Gorm version (v2) is supported by using the
> `github.com/go-gormigrate/gormigrate/v2` import path as described in the
> documentation below.

## Supported databases

Gormigrate is expected to support any of the
[databases Gorm supports](https://gorm.io/docs/connecting_to_the_database.html).

The integration tests cover the following databases:

| Database             | Version tested          |
| -------------------- | ----------------------- |
| PostgreSQL           | 18                      |
| MySQL                | 9                       |
| MariaDB              | 12                      |
| Microsoft SQL Server | 2025                    |
| SQLite (CGo)         | via `mattn/go-sqlite3`  |
| SQLite (pure Go)     | via `glebarez/sqlite`   |

## Usage

```go
package main

import (
	"log"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	db, err := gorm.Open(sqlite.Open("./data.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal(err)
	}

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{{
		// create `users` table
		ID: "201608301400",
		Migrate: func(tx *gorm.DB) error {
			// it's a good practice to copy the struct inside the function,
			// so side effects are prevented if the original struct changes during the time
			type user struct {
				ID   uuid.UUID `gorm:"type:uuid;primaryKey;uniqueIndex"`
				Name string
			}
			return tx.Migrator().CreateTable(&user{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("users")
		},
	}, {
		// add `age` column to `users` table
		ID: "201608301415",
		Migrate: func(tx *gorm.DB) error {
			// when table already exists, define only columns that are about to change
			type user struct {
				Age int
			}
			return tx.Migrator().AddColumn(&user{}, "Age")
		},
		Rollback: func(tx *gorm.DB) error {
			type user struct {
				Age int
			}
			return tx.Migrator().DropColumn(&user{}, "Age")
		},
	}, {
		// create `organizations` table where users belong to
		ID: "201608301430",
		Migrate: func(tx *gorm.DB) error {
			type organization struct {
				ID      uuid.UUID `gorm:"type:uuid;primaryKey;uniqueIndex"`
				Name    string
				Address string
			}
			if err := tx.Migrator().CreateTable(&organization{}); err != nil {
				return err
			}
			type user struct {
				OrganizationID uuid.UUID `gorm:"type:uuid"`
			}
			return tx.Migrator().AddColumn(&user{}, "OrganizationID")
		},
		Rollback: func(tx *gorm.DB) error {
			type user struct {
				OrganizationID uuid.UUID `gorm:"type:uuid"`
			}
			if err := tx.Migrator().DropColumn(&user{}, "OrganizationID"); err != nil {
				return err
			}
			return tx.Migrator().DropTable("organizations")
		},
	}})

	if err := m.Migrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migration did run successfully")
}
```

## Having a separate function for initializing the schema

If you have a lot of migrations, it can be a pain to run all them, as example,
when you are deploying a new instance of the app, in a clean database.
To prevent this, you can set a function that will run if no migration was run
before (in a new clean database). Remember to create everything here, all tables,
foreign keys and what more you need in your app.

```go
type Organization struct {
	gorm.Model
	Name    string
	Address string
}

type User struct {
	gorm.Model
	Name string
	Age int
	OrganizationID uint
}

m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
    // your migrations here
})

m.InitSchema(func(tx *gorm.DB) error {
	err := tx.AutoMigrate(
		&Organization{},
		&User{},
		// all other tables of you app
	)
	if err != nil {
		return err
	}

	if err := tx.Exec("ALTER TABLE users ADD CONSTRAINT fk_users_organizations FOREIGN KEY (organization_id) REFERENCES organizations (id)").Error; err != nil {
		return err
	}
	// all other constraints, indexes, etc...
	return nil
})
```

## Options

This is the options struct, in case you don't want the defaults:

```go
type Options struct {
	// TableName is the migration table.
	TableName string
	// IDColumnName is the name of column where the migration id will be stored.
	IDColumnName string
	// IDColumnSize is the length of the migration id column
	IDColumnSize int
	// UseTransaction makes Gormigrate execute migrations inside a single transaction.
	// Keep in mind that not all databases support DDL commands inside transactions.
	UseTransaction bool
	// ValidateUnknownMigrations will cause migrate to fail if there's unknown migration
	// IDs in the database
	ValidateUnknownMigrations bool
}
```

## Who is Gormigrate for?

Gormigrate was born to be a simple and minimalistic migration tool for small
projects that uses [Gorm](http://gorm.io). You may want to take a look at more advanced
solutions like [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
if you plan to scale.

Be aware that Gormigrate has no builtin lock mechanism, so if you're running
it automatically and have a distributed setup (i.e. more than one executable
running at the same time), you might want to use a
[distributed lock/mutex mechanism](https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/)
to prevent race conditions while running migrations.

## Contributing

To run integration tests, some preparations are needed. Please ensure you
have [task](https://taskfile.dev/installation) and [docker](https://docs.docker.com/engine/install) installed.
Then:

1. Ensure target or all databases are available and ready to accept connections.
   You can start databases locally with `task docker:compose:up`
2. Copy `integration-test/.example.env` as `integration-test/.env` and
   adjust the database connection ports and credentials when needed.
3. Run integration test for single database or for all

```bash
# run test for MySQL
task test:mysql

# run test for MariaDB
task test:mariadb

# run test for PostgreSQL
task test:postgres

# run test for SQLite
task test:sqlite

# run test for Microsoft SQL Server
task test:sqlserver

# run test for all databases
task test:all
```

Alternatively, you can run everything in one step: `task docker:test`
