# Gormigrate

[![GoDoc](https://godoc.org/gopkg.in/gormigrate.v1?status.svg)](https://godoc.org/gopkg.in/gormigrate.v1)
[![Go Report Card](https://goreportcard.com/badge/gopkg.in/gormigrate.v1)](https://goreportcard.com/report/gopkg.in/gormigrate.v1)
[![Build Status](https://travis-ci.org/go-gormigrate/gormigrate.svg?branch=master)](https://travis-ci.org/go-gormigrate/gormigrate)
[![Build status](https://ci.appveyor.com/api/projects/status/89e414sklbwefyyp?svg=true)](https://ci.appveyor.com/project/andreynering/gormigrate)

Gormigrate is a minimalistic migration helper for [Gorm][gorm].
Gorm already has useful [migrate functions][gormmigrate], just misses
proper schema versioning and migration rollback support.

> IMPORTANT: If you need support to Gorm v1 (which uses
> `github.com/jinzhu/gorm` as its import path), please import Gormigrate by
> using the `gopkg.in/gormigrate.v1` import path.
>
> The current Gorm version (v2) is supported by using the
> `github.com/go-gormigrate/gormigrate/v2` import path as described in the
> documentation below.

## Supported databases

It supports any of the [databases Gorm supports][gormdatabases]:

- PostgreSQL
- MySQL
- SQLite
- Microsoft SQL Server

## Usage

```go
package main

import (
	"log"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func main() {
	db, err := gorm.Open("sqlite3", "mydb.sqlite3")
	if err != nil {
		log.Fatal(err)
	}

	db.LogMode(true)

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		// create persons table
		{
			ID: "201608301400",
			Migrate: func(tx *gorm.DB) error {
				// it's a good pratice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				type Person struct {
					gorm.Model
					Name string
				}
				return tx.AutoMigrate(&Person{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("people")
			},
		},
		// add age column to persons
		{
			ID: "201608301415",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, it just adds fields as columns
				type Person struct {
					Age int
				}
				return tx.AutoMigrate(&Person{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn("people", "age")
			},
		},
		// add pets table
		{
			ID: "201608301430",
			Migrate: func(tx *gorm.DB) error {
				type Pet struct {
					gorm.Model
					Name     string
					PersonID int
				}
				return tx.AutoMigrate(&Pet{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("pets")
			},
		},
	})

	if err = m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Printf("Migration did run successfully")
}
```

## Having a separated function for initializing the schema

If you have a lot of migrations, it can be a pain to run all them, as example,
when you are deploying a new instance of the app, in a clean database.
To prevent this, you can set a function that will run if no migration was run
before (in a new clean database). Remember to create everything here, all tables,
foreign keys and what more you need in your app.

```go
type Person struct {
	gorm.Model
	Name string
	Age int
}

type Pet struct {
	gorm.Model
	Name     string
	PersonID int
}

m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
    // you migrations here
})

m.InitSchema(func(tx *gorm.DB) error {
	err := tx.AutoMigrate(
		&Person{},
		&Pet{},
		// all other tables of you app
	)
	if err != nil {
		return err
	}

  if err := tx.Exec("ALTER TABLE pets ADD CONSTRAINT fk_pets_people FOREIGN KEY (person_id) REFERENCES people (id)").Error; err != nil {
		return err
	}
	// all other foreign keys...
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
projects that uses [Gorm][gorm]. You may want to take a look at more advanced
solutions like [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
if you plan to scale.

Be aware that Gormigrate has no builtin lock mechanism, so if you're running
it automatically and have a distributed setup (i.e. more than one executable
running running at the same time), you might want to use a
[distributed lock/mutex mechanism](https://redis.io/topics/distlock) to
prevent race conditions while running migrations.

## Contributing

To run tests, first copy `.sample.env` as `sample.env` and edit the connection
string of the database you want to run tests against. Then, run tests like
below:

```bash
# running tests for PostgreSQL
go test -tags postgresql

# running test for MySQL
go test -tags mysql

# running tests for SQLite
go test -tags sqlite

# running tests for SQL Server
go test -tags sqlserver

# running test for multiple databases at once
go test -tags 'sqlite postgresql mysql'
```

Or altenatively, you could use Docker to easily run tests on all databases
at once. To do that, make sure Docker is installed and running in your machine
and then run:

```bash
task docker
```

[gorm]: http://gorm.io/
[gormmigrate]: https://gorm.io/docs/migration.html
[gormdatabases]: https://gorm.io/docs/connecting_to_the_database.html
