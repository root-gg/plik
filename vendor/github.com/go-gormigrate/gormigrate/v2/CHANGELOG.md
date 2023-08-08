# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]


## [2.1.0] - 2023-06-01
### Changed
- Refactor plain sql mutation statements (create, insert, delete) into native gorm methods
- Update dependencies

## [2.0.3] - 2023-05-29
### Changed
- Update dependencies

## [2.0.2] - 2022-05-29
### Changed
- Update dependencies

## [2.0.1] - 2022-05-15
### Changed
- Update dependencies

## [2.0.0] - 2020-09-05
### Changed
- Make it compatible with Gorm v2, which uses a new import path and has
  breaking changes on its API
  ([#45](https://github.com/go-gormigrate/gormigrate/issues/45), [#46](https://github.com/go-gormigrate/gormigrate/pull/46)).

## [1.6.0] - 2019-07-07
### Added
- Add option to return an error if the database have unknown migrations
  (defaults to `false`)
  ([#37](https://github.com/go-gormigrate/gormigrate/pull/37)).

## [1.5.0] - 2019-04-29
### Changed
- Making the code more safe by checking more errors
  ([#35](https://github.com/go-gormigrate/gormigrate/pull/35)).
### Fixed
- Fixed and written tests for transaction handling
  ([#34](https://github.com/go-gormigrate/gormigrate/pull/34), [#10](https://github.com/go-gormigrate/gormigrate/issues/10)).
  Enabling transation is recommend, but only supported for databases that
  support DDL transactions (PostgreSQL, Microsoft SQL Server and SQLite).

## [1.4.0] - 2019-02-03
### Changed
- Allow an empty migration list if a `InitSchema` function is defined
  ([#28](https://github.com/go-gormigrate/gormigrate/pull/28)).

## [1.3.1] - 2019-01-26
### Fixed
- Fixed `testify` import path from `gopkg.in/stretchr/testify.v1` to
  `github.com/stretchr/testify` ([#27](https://github.com/go-gormigrate/gormigrate/pull/27)).

## [1.3.0] - 2018-12-02
### Changed
- Starting from this release, this package is available as a [Go Module](https://github.com/golang/go/wiki/Modules).
  Import path is still `gopkg.in/gormigrate.v1` in this major version, but will
  change to `github.com/go-gormigrate/gormigrate/v2` in the next major release;
- Validate the ID exists on the migration list (#20, #21).

## [1.2.1] - 2018-09-07
### Added
- Added `MigrateTo` and `RollbackTo` methods (#15);
- CI now runs tests for SQLite, PostgreSQL, MySQL and Microsoft SQL Server.
### Changed
- An empty migration list is not allowed anymore. Please, make sure that you
  have at least one migration, even if dummy;

## [1.2.0] - 2018-07-12
### Added
- Add `IDColumnSize` options, which defaults to `255` (#7);

## [1.1.4] - 2018-05-06
### Changed
- Assuming default options if blank;
- Returning an error if the migration list has a duplicated migration ID.

## [1.1.3] - 2018-02-25
### Added
- Introduce changelog
### Fixed
- Fix `RollbackLast` (#4).
