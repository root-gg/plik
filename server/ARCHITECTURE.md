# Architecture — Server (`server/`)

> Internals of the Plik HTTP server. For system-wide overview, see the root [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Package Structure

```
server/
├── main.go         ← entry point (calls cmd.Execute())
├── plikd.cfg       ← default configuration file
├── cmd/            ← CLI commands (cobra)
├── common/         ← shared types, config, feature flags, utilities
├── context/        ← custom request context
├── data/           ← data backend interface + implementations
├── handlers/       ← HTTP handler functions
├── metadata/       ← metadata backend (GORM)
├── middleware/     ← middleware chain components
└── server/         ← HTTP server setup + router
```

---

## `cmd/` — CLI Commands (Cobra)

The server binary `plikd` uses [cobra](https://github.com/spf13/cobra) for CLI management.

| File | Command | Description |
|------|---------|-------------|
| `root.go` | `plikd` | Start the server (default command) |
| `user.go` | `plikd user create/show/update/list/delete` | Manage users |
| `token.go` | `plikd token create/list/delete` | Manage user tokens |
| `file.go` | `plikd file list/show/delete` | Manage uploads/files (`delete` requires `--file`, `--upload`, or `--all`) |
| `clean.go` | `plikd clean` | Run metadata cleanup |
| `import.go` | `plikd import [input-file]` | Import metadata from gob + Snappy binary |
| `export.go` | `plikd export [output-file]` | Export metadata to gob + Snappy binary |

Config loading order: `--config` flag → `PLIKD_CONFIG` env → `./plikd.cfg` → `/etc/plikd.cfg`.

---

## `common/` — Shared Types & Config

Core types used throughout the server:

| File | Content |
|------|---------|
| `upload.go` | `Upload` struct — container for files with TTL, options, password, E2EE scheme |
| `file.go` | `File` struct + status constants (`missing`/`uploading`/`uploaded`/`removed`/`deleted`) |
| `user.go` | `User` struct + provider constants (`local`/`google`/`ovh`/`oidc`/`github`), includes `Theme` field for webapp theme preference |
| `token.go` | `Token` struct — prefixed opaque tokens (`plik_` + Base62 random + CRC32 checksum). Backward-compatible with legacy UUIDv4 tokens. |
| `cli_auth_session.go` | `CLIAuthSession` struct — ephemeral device auth sessions for CLI login |
| `config.go` | `Configuration` struct — TOML parsing + env var override |
| `feature_flags.go` | Feature flag types: `disabled`/`enabled`/`default`/`forced` |
| `settings.go` | `Setting` struct — server-level key/value (e.g., auth signing key) |
| `authentication.go` | `SessionAuthenticator` — JWT session cookie management |
| `paging.go` | `PagingQuery` — pagination parameters |
| `stats.go` | `ServerStats` — upload/file/user counts |
| `metrics.go` | `PlikMetrics` — Prometheus metric registry |
| `version.go` | Build info (version, git commit, build date) |
| `utils.go` | `GenerateRandomID()`, `StripPrefix()`, `IsPlikWebapp()`, etc. |

---

## `context/` — Custom Request Context

> **Historical note**: This package predates Go's stdlib `context.Context` (added in Go 1.7). It provides a typed, mutex-protected struct that carries request-scoped values through the middleware chain.

The `Context` struct holds:

| Field | Type | Set By |
|-------|------|--------|
| `config` | `*Configuration` | Server init |
| `logger` | `*Logger` | Server init |
| `metadataBackend` | `*metadata.Backend` | Server init |
| `dataBackend` | `data.Backend` | Server init |
| `streamBackend` | `data.Backend` | Server init |
| `authenticator` | `*SessionAuthenticator` | Server init |
| `metrics` | `*PlikMetrics` | Server init |
| `sourceIP` | `net.IP` | `SourceIP` middleware |
| `upload` | `*Upload` | `Upload` middleware |
| `file` | `*File` | `File` middleware |
| `user` | `*User` | `Authenticate` middleware |
| `token` | `*Token` | `Authenticate` middleware |
| `pagingQuery` | `*PagingQuery` | `Paginate` middleware |

All fields are accessed via getter/setter methods protected by a `sync.RWMutex`. Getters panic if a required field is nil (fail-fast pattern).

The `context` package also provides `Chain` — a composable middleware chain builder: `NewChain(mw...).Append(mw...).Then(handler)`.

---

## `data/` — Data Backend

The `Backend` interface is minimal (3 methods):

```go
type Backend interface {
    AddFile(file *common.File, reader io.Reader) (err error)
    GetFile(file *common.File) (reader io.ReadSeekCloser, err error)
    RemoveFile(file *common.File) (err error)
}
```

### Implementations

| Package | Backend | Notes |
|---------|---------|-------|
| `data/file` | Local filesystem | Files stored in configurable directory. |
| `data/s3` | Amazon S3 / MinIO | Supports SSE-C/S3 encryption. |
| `data/swift` | OpenStack Swift | |
| `data/gcs` | Google Cloud Storage | |
| `data/stream` | In-memory pipe | Blocks uploader until downloader connects — nothing stored. Configurable `StreamTimeoutStr` releases blocked goroutines. On error, file resets to `missing` (retryable). `RemoveFile` closes the pipe to unblock cancelled uploads. |
| `data/testing` | In-memory map | For tests only |

---

## `metadata/` — Metadata Backend (GORM)

Uses GORM with gormigrate for schema management across SQLite3, PostgreSQL, and MySQL.

### Key behaviors

- **SQLite3**: WAL mode + foreign keys enabled on connect
- **Schema init**: Auto-migrates `Upload`, `File`, `User`, `Token`, `Setting`, `CLIAuthSession` tables
- **Migrations**: Versioned via gormigrate — see `migrations.go`
- **Cleaning**: `Clean()` removes orphan files and tokens and CLI auth sessions
- **Metrics**: GORM Prometheus plugin for DB stats

### Files

| File | Purpose |
|------|---------|
| `metadata.go` | Backend init, config, shutdown, clean |
| `migrations.go` | Schema migration definitions |
| `upload.go` | Upload CRUD + listing + expiration |
| `file.go` | File CRUD + status updates |
| `user.go` | User CRUD + listing; `RemoveUserUploads` bulk soft-deletes uploads + files in a single transaction |
| `token.go` | Token CRUD + listing |
| `cli_auth_session.go` | CLI auth session CRUD (create, get by code, update, delete expired) |
| `setting.go` | Server settings key/value store |
| `stats.go` | Aggregate statistics queries |
| `exporter.go` | gob + Snappy export of all metadata |
| `importer.go` | gob + Snappy import |

### Import / Export

The `plikd export` and `plikd import` commands dump and restore all metadata (users, tokens, uploads, files, settings) to/from a single binary file.

- **Format**: Go [gob](https://pkg.go.dev/encoding/gob) encoding compressed with [Snappy](https://github.com/golang/snappy). Architecture-independent (portable across `amd64`/`arm64`), streaming (constant memory), Go-specific (not human-readable).
- **Export order**: users → tokens → uploads (including soft-deleted) → files → settings. CLI auth sessions are intentionally excluded (ephemeral).
- **Import**: decodes sequentially, calls `Create*` on the metadata backend. Supports `--ignore-errors` to skip problematic records.
- **Use cases**: backend migration (e.g. SQLite → PostgreSQL), backups, disaster recovery.

> **Note**: Only metadata is exported — file data in the data backend must be migrated separately.

### Migration Dump Tests

When adding a new migration, `TestGenerateSQLDump` and `TestGenerateExport` in `migrations_test.go` auto-generate dump files on first run. Each dump captures the full DB state after all migrations and test data have been applied. `TestMigrationsFromSQLDumps` and `TestLoadExports` then load **all** existing dumps and verify migrations can be replayed forward to the current schema.

**Dump directory structure**:
```
metadata/dumps/
├── sqlite3/      ← sqlite3 .dump output (generated by make test)
├── export/       ← gob + Snappy export   (generated by make test)
├── postgres/     ← pg_dump output        (generated by make test-backends)
├── mysql/        ← mysqldump output      (generated by make test-backends)
└── mariadb/      ← mariadb-dump output   (generated by make test-backends)
```

**Two-stage generation process**:

| Stage | Command | Backends | Infrastructure |
|-------|---------|----------|----------------|
| 1 | `make test` | `sqlite3` + `export` | Local, requires `sqlite3` CLI (`apt install sqlite3`) |
| 2 | `make test-backends` | `postgres`, `mysql`, `mariadb` | Docker containers (e.g., `plik.postgres`), dumps via `docker exec` |

1. Run `make test` locally → generates `sqlite3` + `export` dumps
2. Run `make test-backends` locally → generates `postgres`, `mysql`, `mariadb` dumps

> **[!IMPORTANT] Always commit dump files** in `server/metadata/dumps/` After adding a new migration

**CI workflow** (`.github/workflows/tests.yaml`):
Both jobs run a **"Check for uncommitted changes"** step after tests — `git diff --exit-code` + `git ls-files --others` — which **fails the build** if any files were generated but not committed (e.g., missing dump files for a new migration)

**PostgreSQL 18+ compatibility**: `pg_dump` now emits `\restrict`/`\unrestrict` psql meta-commands (CVE-2025-8714 sandbox). Since `loadSQLDump` executes dumps via `sqlDB.Exec()` (not `psql`), these are not valid SQL. The loader filters out all lines starting with `\` before execution.

---

## `middleware/` — Middleware Chain

Each middleware is a function that takes a `context.Context` and optionally calls the next handler.

| File | Middleware | Purpose |
|------|-----------|---------|
| `context.go` | `Context()` | Initialize context with server-level values |
| `log.go` | `Log` | Request/response logging |
| `recover.go` | `Recover` | Panic recovery → HTTP error response |
| `source_ip.go` | `SourceIP` | Extract client IP (supports `X-Forwarded-For` header) |
| `authenticate.go` | `Authenticate(acceptToken)` | Parse session cookie / X-PlikToken header → set user/token |
| `impersonate.go` | `Impersonate` | Admin impersonation support |
| `upload.go` | `Upload` | Resolve `{uploadID}` → load upload + check auth |
| `file.go` | `File` | Resolve `{fileID}` → load file from upload |
| `create_upload.go` | `CreateUpload` | Parse upload creation params for quick upload |
| `paginate.go` | `Paginate` | Parse pagination query params |
| `redirect.go` | `RedirectOnFailure` | Redirect to webapp on error (for browser requests) |
| `block_bot_download.go` | `BlockBotDownload` | Block messaging app link preview bots from downloading one-shot/streaming files (returns 406) |
| `user.go` | `User` | Resolve `{userID}` → load user (admin or self) |
| `cors.go` | `CORSPreflight` | Short-circuits OPTIONS preflight requests with CORS headers (runs before Upload/File middleware) |
| `limit_body.go` | `LimitBody` | Wraps request body with `http.MaxBytesReader` (1 MiB) to reject oversized payloads on JSON API endpoints. In `stdChain` — auto-skips GET/HEAD/DELETE/OPTIONS and file upload paths (`/`, `/file/*`, `/stream/*`) which have their own stream-based size limiting. |
| `download_domain.go` | `RestrictDownloadDomain` | Router-level middleware: blocks non-file routes on the download domain when PlikDomain is also set (redirects to PlikDomain) |

---

## `handlers/` — HTTP Handlers

Each handler file contains one or more `http.Handler` functions.

| File | Handlers | Description |
|------|----------|-------------|
| `create_upload.go` | `CreateUpload` | Create upload with options, validate config/quotas |
| `add_file.go` | `AddFile` | Upload file to existing upload (multipart). Detects content type via [`gabriel-vasile/mimetype`](https://github.com/gabriel-vasile/mimetype) magic-number sniffing (200+ formats). E2EE uploads are forced to `application/octet-stream` via age-header detection. On stream upload error, resets file to `missing` (retryable); on regular upload error, purges partial data and leaves file in `uploading` (not retryable). |
| `get_upload.go` | `GetUpload` | Return upload metadata |
| `get_file.go` | `GetFile` | Download file, handle OneShot, extend TTL, support HTTP range requests (via `http.ServeContent` for non-stream/non-oneshot). E2EE uploads: redirects webapp to download page, forces `application/octet-stream` |
| `get_archive.go` | `GetArchive` | Download all files as zip. Compression method controlled by `EnableArchiveCompression` (default: `zip.Deflate`). Disable to prevent CPU exhaustion DoS on public instances. |
| `remove_file.go` | `RemoveFile` | Mark file as removed. Also mapped to `DELETE /stream/...` to allow cancelling a blocked stream upload (closes the in-memory pipe). |
| `remove_upload.go` | `RemoveUpload` | Soft-delete upload |
| `misc.go` | `GetConfiguration`, `GetVersion`, `GetQrCode`, `Health` | Utility endpoints |
| `local.go` | `LocalLogin`, `Logout` | Local auth |
| `google.go` | `GoogleLogin`, `GoogleCallback` | Google OAuth |
| `ovh.go` | `OvhLogin`, `OvhCallback` | OVH OAuth |
| `oidc.go` | `OIDCLogin`, `OIDCCallback` | OpenID Connect |
| `github.go` | `GitHubLogin`, `GitHubCallback` | GitHub OAuth (optional org restriction) |
| `cli_auth.go` | `CLIAuthInit`, `CLIAuthApprove`, `CLIAuthPoll` | CLI device auth flow |
| `me.go` | `UserInfo`, `PatchMe`, `DeleteAccount`, `GetUserStatistics`, `GetUserUploads`, `RemoveUserUploads` | Current user; token filter param accepts both prefixed and legacy UUID formats (no DB lookup, works for revoked tokens) |
| `token.go` | `GetUserTokens`, `CreateToken`, `RevokeToken` | Token management |
| `user.go` | `GetUsers`, `CreateUser`, `UpdateUser` | User management |
| `admin.go` | `GetServerStatistics`, `GetUploads` | Admin endpoints |

---

## `server/` — HTTP Server Setup

`PlikServer` is the main server struct. It:

1. Initializes backends (metadata, data, stream) and authenticator
2. Calls `ensureDefaultAdmin()` — creates a local admin user if `DefaultAdminLogin` is configured and the user does not yet exist (idempotent)
3. Builds middleware chains (see root ARCHITECTURE.md for chain table)
4. Configures gorilla/mux router with all routes
5. Starts HTTP server via `net.Listen` + `httpServer.Serve` (supports ephemeral port allocation with `ListenPort: 0`)
6. Starts cleaning routine (if auto-clean enabled)
7. Starts metrics HTTP server (if configured)

After start, call `GetListenPort()` to retrieve the actual listen port (useful when configured with port 0).

Shutdown: graceful with configurable timeout, closes HTTP server + metadata backend.
