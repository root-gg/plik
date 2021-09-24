# Changelog

## v1.6.0 - 2019-07-07

- Add option to return an error if the database have unknown migrations
  (defaults to `false`)
  ([#37](https://github.com/go-gormigrate/gormigrate/pull/37)).

## v1.5.0 - 2019-04-29

- Fixed and written tests for transaction handling
  ([#34](https://github.com/go-gormigrate/gormigrate/pull/34), [#10](https://github.com/go-gormigrate/gormigrate/issues/10)).
  Enabling transation is recommend, but only supported for databases that
  support DDL transactions (PostgreSQL, Microsoft SQL Server and SQLite).
- Making the code more safe by checking more errors
  ([#35](https://github.com/go-gormigrate/gormigrate/pull/35)).

## v1.4.0 - 2019-02-03

- Allow an empty migration list if a `InitSchema` function is defined
  ([#28](https://github.com/go-gormigrate/gormigrate/pull/28)).

## v1.3.1 - 2019-01-26

- Fixed `testify` import path from `gopkg.in/stretchr/testify.v1` to
  `github.com/stretchr/testify` ([#27](https://github.com/go-gormigrate/gormigrate/pull/27)).

## v1.3.0 - 2018-12-02

- Starting from this release, this package is available as a [Go Module](https://github.com/golang/go/wiki/Modules).
  Import path is still `gopkg.in/gormigrate.v1` in this major version, but will
  change to `github.com/go-gormigrate/gormigrate/v2` in the next major release;
- Validate the ID exists on the migration list (#20, #21).

## v1.2.1 - 2018-09-07

- An empty migration list is not allowed anymore. Please, make sure that you
  have at least one migration, even if dummy;
- Added `MigrateTo` and `RollbackTo` methods (#15);
- CI now runs tests for SQLite, PostgreSQL, MySQL and Microsoft SQL Server.

## v1.2.0 - 2018-07-12

- Add `IDColumnSize` options, which defaults to `255` (#7);

## v1.1.4 - 2018-05-06

- Assuming default options if blank;
- Returning an error if the migration list has a duplicated migration ID.

## v1.1.3 - 2018-02-25

- Fix `RollbackLast` (#4).

---

Sorry, we don't have changelog for older releases 😢.
