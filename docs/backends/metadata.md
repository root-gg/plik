# Metadata Backends

Plik uses GORM for metadata storage, supporting multiple SQL databases.

## SQLite3 (Default)

Best for standalone deployments.

```toml
[MetadataBackendConfig]
    Driver = "sqlite3"
    ConnectionString = "plik.db"
    Debug = false
```

SQLite3 is configured with WAL mode and foreign keys enabled for optimal performance and data integrity.

## PostgreSQL

Best for distributed or high-availability deployments.

```toml
[MetadataBackendConfig]
    Driver = "postgres"
    ConnectionString = "host=localhost user=plik password=plik dbname=plik port=5432 sslmode=disable"
    Debug = false
```

## MySQL / MariaDB

Also suitable for distributed deployments.

```toml
[MetadataBackendConfig]
    Driver = "mysql"
    ConnectionString = "plik:plik@tcp(localhost:3306)/plik?charset=utf8mb4&parseTime=True"
    Debug = false
```

## Connection Pool

For PostgreSQL and MySQL, you can tune the connection pool:

```toml
[MetadataBackendConfig]
    MaxOpenConns = 25
    MaxIdleConns = 10
```

## Slow Query Logging

Enable slow query detection:

```toml
[MetadataBackendConfig]
    SlowQueryThreshold = "200ms"
```

## Schema Migrations

Plik uses [gormigrate](https://github.com/go-gormigrate/gormigrate) for automatic schema migrations. The database schema is created or updated automatically on server start.

## Migrating Between Backends

Plik provides two approaches for migrating between metadata backends:

- **`plikd export` / `plikd import`** — Export metadata to a portable file and import it into a new backend. Good for offline backups or scheduled migrations. See the [Import / Export](/operations/import-export) guide.
- **`plikd migrate`** — Live, direct backend-to-backend migration that also handles file data blobs in parallel. No intermediate files needed. See the [Migration](/operations/migration) guide.
