# Migration Guide

Plik supports live, direct backend-to-backend migration via the `plikd migrate` command. This lets you move all data and/or metadata from one backend configuration to another **without creating intermediate archive files**. File data is streamed directly from the source backend to the target backend.

## Overview

The `plikd migrate` command:

- Reads the source backends from the current `plikd.cfg` (same config used to start the server)
- Reads the **target** backends from a second config file (`--to` flag)
- Migrates metadata (users, tokens, uploads, files, settings) in order, respecting foreign key dependencies
- Migrates file blobs in parallel (configurable workers, default: 4)
- Supports dry-run mode, error tolerance, and selective migration

> [!IMPORTANT]
> Stop the Plik server before running `plikd migrate`. Running a migration against a live server may produce inconsistent results.

## Basic Syntax

```sh
plikd migrate --to /path/to/target.cfg [options]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--to` | *(required)* | Path to the target `plikd.cfg` |
| `--metadata-only` | false | Only migrate metadata (skip file blobs) |
| `--data-only` | false | Only migrate file blobs (skip metadata) |
| `--workers` | 4 | Number of parallel file copy workers |
| `--dry-run` | false | Print every item that would be migrated, write nothing |
| `--ignore-errors` | false | Log errors per item and continue (don't abort) |

## Common Scenarios

### 1. SQLite → PostgreSQL (metadata only)

Your current `plikd.cfg`:
```toml
MetadataBackend = "sqlite3"
[MetadataBackendConfig]
  ConnectionString = "/home/plik/server/db/plik.db"
```

Create a `plikd-pg.cfg` with just the fields that differ:
```toml
MetadataBackend = "postgres"
[MetadataBackendConfig]
  ConnectionString = "host=localhost user=plik password=secret dbname=plik sslmode=disable"
```

Run:
```sh
plikd migrate --to plikd-pg.cfg --metadata-only
```

### 2. Local Files → S3 (data only)

You already have PostgreSQL running. Create `plikd-s3.cfg`:
```toml
DataBackend = "s3"
[DataBackendConfig]
  Bucket     = "my-plik-bucket"
  Region     = "us-east-1"
  # Credentials via AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars
```

Run:
```sh
plikd migrate --to plikd-s3.cfg --data-only --workers 8
```

### 3. Full Migration (SQLite + Local → PostgreSQL + S3)

1. Prepare `plikd-new.cfg` with both new backends configured.
2. First preview with dry-run:
   ```sh
   plikd migrate --to plikd-new.cfg --dry-run
   ```
3. Run the full migration:
   ```sh
   plikd migrate --to plikd-new.cfg
   ```
4. Update your main `plikd.cfg` to use the new backends and start the server.

### 4. Resuming a Failed Migration

If a migration is interrupted, re-run with `--ignore-errors` to skip already-migrated records (unique constraint violations) and continue:
```sh
plikd migrate --to plikd-new.cfg --ignore-errors
```

> [!WARNING]
> With `--ignore-errors`, migration errors are silently skipped and logged to stdout. Always review the output to ensure no critical data was lost.

## Migration Order

Metadata is migrated in dependency order to avoid foreign key violations:

1. **Users**
2. **Tokens** (depend on users)
3. **Uploads** (including soft-deleted, to maintain FK integrity with files)
4. **Files** (metadata records)
5. **Settings**

File **data blobs** are migrated after metadata, in parallel. Files with status `uploaded` or `removed` are copied (both still exist in the data backend). Files with status `missing`, `uploading`, or `deleted` are skipped.

## Verifying After Migration

After migration, verify the target by querying the HTTP API (with the server running against the new config):

```sh
# Check server stats via API
curl http://localhost:8080/version
```

Or query record counts directly in your database. For example with SQLite:

```sh
sqlite3 /path/to/new/plik.db "SELECT COUNT(*) FROM uploads; SELECT COUNT(*) FROM files;"
```
