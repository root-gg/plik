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
| `fakedb.go` | `plikd fakedb` | Generate a fake SQLite metadata database for UI testing and benchmarks; seeds usage stats and bounded fake download rollups by default |

Config loading order: `--config` flag → `PLIKD_CONFIG` env → `./plikd.cfg` → `/etc/plikd.cfg`.

---

## `common/` — Shared Types & Config

Core types used throughout the server:

| File | Content |
|------|---------|
| `upload.go` | `Upload` struct — container for files with TTL, options, password, E2EE scheme. `Sanitize()` populates `DownloadURL` (= `config.DownloadURL`) alongside the legacy `DownloadDomain` field for backward compatibility. |
| `file.go` | `File` struct + status constants (`missing`/`uploading`/`uploaded`/`removed`/`deleted`) |
| `user.go` | `User` struct + provider constants (`local`/`google`/`ovh`/`oidc`/`github`), includes `Theme` field for webapp theme preference |
| `token.go` | `Token` struct — prefixed opaque tokens (`plik_` + Base62 random + CRC32 checksum). Backward-compatible with legacy UUIDv4 tokens. |
| `cli_auth_session.go` | `CLIAuthSession` struct — ephemeral device auth sessions for CLI login |
| `config.go` | `Configuration` struct — TOML parsing + env var override. `Initialize(*logger.Logger)` strips path components from `PlikDomain`/`DownloadDomain`/aliases (warns if found) and computes `DownloadURL = DownloadDomain + Path`. URL helpers: `GetServerURL()`, `GetDownloadBaseURL()` (returns `DownloadDomain+Path` or `GetServerURL()`), `GetFileURL(uploadID, fileID, name, stream)`, `GetArchiveURL(uploadID, name)` — centralise all download link construction. |
| `feature_flags.go` | Feature flag types: `disabled`/`enabled`/`default`/`forced` |
| `settings.go` | `Setting` struct — server-level key/value (e.g., auth signing key) |
| `authentication.go` | `SessionAuthenticator` — JWT session cookie management |
| `paging.go` | `PagingQuery` — pagination parameters |
| `stats.go` | Counter-backed user/server/token stats, download rollups, and trending response types |
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
| `stats.go` | Read APIs for counter-backed user/server statistics |
| `stats_usage.go` | Atomic user/server/token usage counters, feature counters, TTL buckets, file-size buckets, and backfill helpers |
| `stats_download.go` | Download counter mutation, daily rollups, and import/export iteration |
| `stats_trending.go` | Lifetime and windowed trending queries for uploads/files, with an upload sort dimension (downloads/downloadedBytes) and an optional per-user scope |
| `exporter.go` | gob + Snappy export of all metadata |
| `importer.go` | gob + Snappy import |

### Stats Architecture

Stats are part of the metadata backend, not process memory. Multiple Plik instances may serve traffic against the same SQL database, so every counter mutation is done inside the same database transaction as the metadata state change that makes the event real — this document and the code call these **fused** metadata+counter transactions (`CreateUpload`, file status transitions, upload/user/token removal): the state change and its counters commit or roll back together, so they can never drift apart, and a failure surfaces to the caller because the operation itself failed. They contrast with the counter-only **best-effort** recorders (`RecordFileDownload`, `RecordArchiveDownload`, `RecordUploadedBytes`), where a failure is logged and swallowed without affecting the request. Counters are updated with SQL increments (`column = column + delta`). There is no shared global "server" row and no transaction-retry wrapper: server totals are summed on read over the per-scope rows, and the canonical lock order (below) makes the remaining writers deadlock-free, so their transactions run once.

#### Goals and Scope

- **Current usage** answers "what is retained right now": active uploads, uploaded files, retained size, and current feature/TTL/file-size distribution.
- **Lifetime usage** answers "what has happened since stats tracking started": lifetime uploads, lifetime files, lifetime size, and lifetime feature/TTL/file-size distribution.
- **Download stats** count logical download events (intent). Separately, **bytes served (egress)** are tracked to answer "how much bandwidth did downloads cost", which download events cannot measure.
- **Trending** is admin-only and derived from download counters.
- **Exact from migration forward**: migration backfill reconstructs stats from retained metadata; already purged uploads/files cannot be recovered.

#### Tables

| Table | Cardinality | Purpose |
|-------|-------------|---------|
| `usage_stats` | One row per `(user_id, token)` scope | Current/lifetime counters for anonymous usage `("__anonymous__", "")`, users `(userID, "")`, tokens `(ownerUserID, token)`, and the deleted-user tombstone `("__deleted__", "")`. **There is no server row**: server totals are the sum on read of every `token=''` row (users + anonymous + tombstone). Every row stores the same feature, TTL, file-size, download event (`downloads`), bytes-served (`downloaded_bytes`), wire bytes received (`uploaded_bytes`), lifetime user (`lifetime_users`), last-upload, and started-at fields. |
| `download_stats_daily` | One row per `(day, entity_type, entity_id)` | Daily upload/file rollups of download events (`downloads`) and bytes served (`bytes`) used for 1d/7d/30d trending. Cleanup keeps today plus 30 previous UTC-day buckets. |
| `upload_stats_daily` | One row per `(day, user_id, token)` | Daily rollups of upload creations (`uploads`) and wire bytes received (`bytes`) per attribution pair. There is no per-entity dimension (one row per day per `(user_id, token)`). Same today+30 UTC-day retention as `download_stats_daily`. |

Each `download_stats_daily` row also carries `user_id`/`token`, copied verbatim from the upload at record time (an anonymous upload's `""` User is stored as-is — unlike `usage_stats`, there is no `__anonymous__` sentinel here). Attribution is written once on insert and is immutable on conflict, so it survives upload deletion the same way the bounded rollup windows already do. `upload_stats_daily` follows the same rule, except its attribution `(user_id, token)` is part of its primary key (there is no entity dimension), so it is immutable by construction. One merged query set (`getActivityStatsDailySeries`, exposed as `GetUserActivityStatsDaily`/`GetServerActivityStatsDaily`) feeds all four measures — downloads/downloaded bytes from `download_stats_daily`'s upload-entity rows only (file-entity rows would double-count) and uploads/uploaded bytes from `upload_stats_daily` — into a dense per-day series, which is what lets a user's chart keep showing activity from uploads they later deleted. The bounded windows on the `usage` object are summed from this same series (`applyActivityWindows`), so windows and chart never drift.

`StartedAt` means "Stats since ..." for UI/API consumers. For rows created by migration it is the migration/backfill time, because older purged metadata cannot be reconstructed. For newly created user/token rows it is the row creation time. The server-scope `StartedAt` is the MIN over the summed `token=''` rows; the `("__deleted__","")` tombstone (created at migration/init) guarantees that MIN is never NULL, including on fresh installs and empty databases. It is read with a model-aware ordered `Take` rather than a raw `MIN(started_at)` aggregate, which loses the column's datetime affinity on SQLite. The tombstone is not necessarily the earliest `token=''` row — an imported user can predate it — so when `DeleteUser` (or a tombstone import) folds a row whose `started_at` is older than the tombstone's, it pulls the tombstone's `started_at` back to that earlier value; otherwise dropping the folded row would let the server `StartedAt` jump forward whenever the oldest scope is deleted.

#### Mutation Boundaries

All usage mutations are centralized in metadata methods so handlers do not hand-edit counters:

- Upload creation increments user/server/token upload counters, feature counters, TTL buckets, and file-size buckets for completed files.
- Upload removal decrements current upload counters, feature counters, TTL buckets, and current retained file/size/file-size bucket counters for files that are still uploaded.
- File completion increments file and size counters exactly once.
- File removal decrements current file and size counters only when a retained uploaded file leaves `FileUploaded`.
- Download recording, for a counted event, increments upload/file `download_count`, `last_downloaded_at`, the user-or-anonymous and token `downloads` totals, and the `downloads` column of the daily rollup rows. Independently, it increments the bytes-served counters — `uploads.downloaded_bytes` on the upload row, `usage_stats.downloaded_bytes` on the user-or-anonymous and token scopes, and the `bytes` column of the daily rollups — for every response that streams file bytes, including mid-range GETs that are not events (bytes only, no `downloads`). A bytes-only recording still locks and updates the upload row (only its `downloaded_bytes` column) but never touches `download_count`/`last_downloaded_at`, nor the hot file row at all — there is no per-file byte column.
- User creation eagerly seeds the user's own `(userID,"")` usage row with `lifetime_users = 1` (uniform per-user counter, mirroring `CreateToken`'s eager token-row seeding). Server `lifetime_users` is the sum of that column over the `token=''` rows, so it counts every user ever created since stats tracking started. It is never decremented: `DeleteUser` folds the user's row — `lifetime_users` and all lifetime counters — into the `("__deleted__","")` tombstone before dropping it (see below), so the server sum is unchanged by deletion. Anonymous/token rows leave `lifetime_users` at 0.
- User deletion folds the user's `(userID,"")` row into the tombstone. `RemoveUserUploads` (run first) has already zeroed the user's *current* counters, so only *lifetime* counters and `lifetime_users` effectively move into the tombstone: server *current* totals are unaffected, server *lifetime* totals are preserved. The fold also pulls the tombstone's `started_at` back to the folded row's when that row is older, so deleting the oldest user does not move the server `StartedAt` forward. Token rows are dropped with the user, not folded (their history goes with the revoked token). A repeat delete is idempotent — the second call finds no row and folds nothing.

Failed or incomplete uploads do not count toward lifetime file/size stats. A
completed transfer that is later rejected by a retention policy, such as user
storage quota, may still count toward lifetime stats because the bytes were
successfully received before the retained current state was cleaned up.

The file status transition rules are especially important:

| Transition | Counter effect |
|------------|----------------|
| `old != uploaded` -> `uploaded` | Count as a completed retained file: current and lifetime file/size increment. |
| `uploaded` -> `new != uploaded` | Remove retained usage only: current file/size decrement, lifetime unchanged. |
| `uploading` -> `deleted` for stream upload | Count as a completed streamed file: lifetime file/size increment, current unchanged. |
| `uploading` -> `deleted` for non-stream upload | No lifetime increment; this is failed upload cleanup. |

This split exists because regular uploads become downloadable only after data is stored, while stream uploads are successfully consumed through the stream and then immediately deleted.

#### Concurrency and Lock Ordering

Counter mutations run inside the same transaction as the metadata change, so concurrent transactions can contend on the hot upload/file rows and on the shared `usage_stats` / `download_stats_daily` rows. To make deadlocks impossible rather than merely retryable, every multi-row stats write acquires row locks in one **canonical order**:

1. `uploads` — the parent upload row.
2. `files` — file rows, ordered by ascending file ID when several are touched.
3. rollups — `download_stats_daily` (the upload bucket first, then file buckets in ascending file ID order) and/or `upload_stats_daily`.
4. `usage_stats` — usage rows in the order user/anonymous, then token. There is no server row to contend on: a single global row would sit on every counted mutation's write path, so server totals are summed on read instead — which is exactly what makes plain, retry-free transactions safe.

Because a download locks its parent upload row first, all downloads of the same upload serialize on that row instead of racing on the rows beneath it. `RecordFileDownload` and `RecordArchiveDownload` both follow this order (see the comment at the top of `metadata/stats_download.go`); any future stats writer that touches multiple rows must follow it too. The upload-side writers obey it as well: `CreateUpload` writes the day's `upload_stats_daily` +1 (a rollup) after the upload row is created and before the usage rows; the fused file-completion transaction (`UpdateFile` → `incrementUsageForCompletedFile`) writes the day's `upload_stats_daily` wire bytes after the upload-row lock and file-row update, before the usage rows. A bytes-only download recording (a mid-range GET) locks the upload row too — to accumulate `uploads.downloaded_bytes` — but still skips the file row entirely (there is no per-file byte column), so it acquires the upload row plus a **suffix** of the remaining order (rollups, then `usage_stats`), never the full order. The best-effort partial-upload recorder `RecordUploadedBytes` is unaffected by this: `uploaded_bytes` has no per-entity column at all, so it still deliberately skips the hot upload/file rows entirely and starts at the daily rollups. Either way — a prefix-then-suffix or a pure suffix of a fixed total order — it can never invert against a writer that takes the full order, so both stay deadlock-free.

This order is **universal across every fused writer**, not just the download recorders. Upload removal (`RemoveUpload`, and `RemoveUserUploads`, which iterates it per upload), the purge path (`DeleteRemovedUploads`), and the file status transitions (`UpdateFile`, `UpdateFileStatus`) all acquire the parent upload row's write lock first via `Backend.lockUploadRow` before they read, transition, or count that upload's file rows. Locking the upload row first makes those file-row reads stable: a removal derives its decrement deltas from exactly the files it transitions out of the uploaded state under the lock, so concurrent one-shot downloads or completions can neither cause a double decrement (counters going negative) nor leak an increment. Because no writer ever takes a file-row lock before the upload-row lock, AB-BA deadlocks between these paths are impossible rather than merely retryable. `lockUploadRow` issues `SELECT ... FOR UPDATE` on PostgreSQL/MySQL/MariaDB and skips the locking clause on SQLite, which has no `FOR UPDATE` syntax and is a single writer under WAL where a plain read already serializes writers.

**There is no transaction-retry wrapper.** Every fused metadata+counter mutation (upload create/remove, file status transitions, token/user creation and removal) runs its transaction exactly once, the same as the best-effort download recorders. This is safe because the deadlock surface is gone, not merely rare:

- There is no single shared hot row: server totals are summed on read rather than maintained in a global counter row that every counted mutation would have to increment, so no two transactions serialize through one global row. The residual shared warmth is the `("__anonymous__","")` row on anonymous-heavy instances; concurrent anonymous mutations simply take that row's lock in turn (an ordered wait), they do not deadlock.
- Every multi-row writer follows the canonical order above, taking the parent upload row first via `lockUploadRow`. Because no writer ever takes a file-row lock before the upload-row lock, two of them can never acquire the same two rows in opposite orders, so an AB-BA cycle is impossible. Conflicts are therefore ordered lock **waits**, not deadlocks or serialization aborts — there is nothing transient to retry away.
- On PostgreSQL/MySQL/MariaDB `lockUploadRow` issues `SELECT ... FOR UPDATE`, so a writer blocks on the row lock (bounded by the server's lock-wait timeout) instead of racing. On SQLite the DSN is opened with `_txlock=immediate` plus a `_busy_timeout` (see `metadata.sqliteConnectionString`): each transaction takes the single WAL write lock at `BEGIN` and waits if it is held, so writers serialize as bounded waits. Without `IMMEDIATE`, a transaction that begins by *reading* the upload row (SQLite has no `FOR UPDATE`) and then writes could deadlock trying to upgrade a deferred read into a writer — an unrecoverable `SQLITE_BUSY` that no busy_timeout resolves; opening `IMMEDIATE` removes that failure mode outright instead of retrying around it.

Download recording (`RecordFileDownload`, `RecordArchiveDownload`) remains **single-shot best-effort**: on any error the counters are simply skipped — the caller logs the failure and the download still succeeds — while user-facing fused mutations propagate an error (now vanishingly unlikely) to the caller. The canonical lock order applies to both; it prevents the conflicts instead of retrying them.

#### Download Counting Policy

Two distinct quantities are recorded on download paths: **download events**
(logical intent) and **bytes served** (egress). Recording happens **after**
response streaming completes (a single best-effort call in `GetFile` after
`http.ServeContent`/`io.Copy`, and in `GetArchive` after the archive closes or
fails mid-stream), so the recorded bytes reflect what was actually written to
the client. The
event decision itself is made from the request *before* streaming, so a client
that starts a valid download and then disconnects mid-stream still counts as an
event and records the partial bytes it received (correct egress).

Download events follow the intent policy:

- Count full-file `GET` requests.
- Count `Range` `GET` requests only when the first requested byte is `0`.
- Ignore `HEAD`, `OPTIONS`, failed auth, not-found files, unavailable files, redirects, malformed ranges, and non-start (mid) ranges.
- The one-shot/stream branch ignores `Range` and streams the whole file, so every `GET` there is a complete download and always counts an event.
- Archive downloads count the upload once and each included file once, after the archive stream is closed successfully.

This prevents audio/video clients from inflating trending by issuing many non-start range requests.

Bytes served are recorded for **every response that streams file bytes**,
independent of the event policy:

- Full GETs, byte-0 ranges, **and mid-range GETs** all record the bytes actually written. A mid-range GET is *not* an event but *is* egress, so it increments bytes only (`uploads.downloaded_bytes` on the upload row, `downloaded_bytes` on every usage scope, and the `bytes` column on the upload and file daily rollups) without incrementing any `downloads` counter.
- `HEAD` serves no body, so it records nothing. Range-error responses (`416`) and other `ServeContent` error bodies write only a tiny error body, not file bytes; those bytes are counted (they are real egress) but never recorded as an event, per the pre-stream policy above.
- A partial transfer (client disconnect) records the bytes actually written, not the full file size.
- For an **archive**, the whole zip stream size is attributed to the upload daily rollup and to the usage rows; the per-file daily rollups get the download event only (0 bytes), because a single compressed zip stream cannot be split across its files. A failed (mid-stream) archive still records its already-served bytes — as a **bytes-only** recording (`RecordArchiveDownload(..., countEvent=false)`): the upload's `downloaded_bytes` and its daily rollup bytes and the usage rows' bytes are updated, but no download event is counted anywhere (`download_count`/`last_downloaded_at` on the upload and every included file, and the per-file daily rollups, all stay untouched), mirroring the single-file mid-range-GET policy above.

Per-entity byte totals are intentionally **not** stored on `files` (there is no
per-file byte column; file-level "downloaded data" is not displayed anywhere).
`uploads.downloaded_bytes` **is** stored on the upload row, symmetric with
`download_count`: it accumulates every response that streams the upload's file
bytes, event or not, and powers the uploads-list "downloaded data" display and
sort. It is deliberately **not** folded back into `usage_stats` on migration
backfill or import (see "Migration Backfill" below) — that would double-count
against the usage/rollup byte paths that already account for the same bytes
independently (e.g. `fakedb`'s separate `FixtureSeedDownloadedBytes` seeding call).

Bytes are tracked **since the upgrade** that introduced them: they cannot be
reconstructed from existing metadata, so migration-backfilled and imported rows
start at `0` bytes even though their download-event counts are rebuilt.

#### Upload Counting and Uploaded Bytes (ingress)

Uploads are the symmetric counterpart of downloads. **Upload events** are counted
as one `upload_stats_daily` +1 in the fused `CreateUpload` transaction (attributed
to the upload's `(user_id, token)` on its UTC creation day) plus the existing
lifetime `usage_stats.lifetime_uploads` counter. **Uploaded bytes (ingress)** are
the wire bytes actually received from the client, symmetric with downloaded
bytes (egress):

- `AddFile` wraps the client file stream (the multipart `file` part, which already
  strips multipart framing) in a counting `io.Reader`, so the count is exactly the
  file content bytes that flow to the data backend — **multipart framing overhead
  is excluded**.
- On **successful completion** the fused completion transaction (`UpdateFile` →
  `updateUsageForFileStatusTransition` → `incrementUsageForCompletedFile`) records
  the completed file's `Size`, which for a fully consumed stream equals the counted
  wire bytes, into `upload_stats_daily.bytes` and `usage_stats.uploaded_bytes` on
  every scope. This is the same transaction that already moves the file/size
  counters, so a completed upload's event, size, and wire bytes commit atomically.
- On a **failure before completion** (stream error, aborted transfer, or backend
  write error) `AddFile` calls `RecordUploadedBytes` with the partial counted
  bytes — **single-shot best-effort** (never retried, never fails the request; a
  recording error is logged and dropped). Because the completion transaction never
  runs on those paths there is no double count. Partial/failed transfers therefore
  still count as ingress, matching how a mid-stream download disconnect records its
  partial egress. The importer path (`CreateFile`) passes `wireBytes=0`, so replayed
  history never fabricates ingress.

Uploaded bytes are tracked **since the upgrade** exactly like downloaded bytes:
migration-backfilled and imported `usage_stats` rows start at `0` (there is no
per-upload byte source to reconstruct from). The `upload_stats_daily` rollups,
however, are exported/imported as records, so their per-day byte volumes survive
a roundtrip (see below).

Server download windows in `/stats` are summed from upload daily rollups, not
file rollups, so every counted file download or archive download contributes
one server event. The `1d` window means the current UTC day; `7d` and `30d`
include today plus the previous 6 or 29 UTC days.

The regular server/CLI cleanup path prunes `download_stats_daily` **and
`upload_stats_daily`** rows older than today plus the previous 30 UTC-day buckets
(the same policy for both tables). This bounds table growth while retaining every
bucket needed by the `30d` window with one extra day of margin.

#### Trending

Lifetime trending sorts current uploads/files by their lifetime `download_count` (or, for uploads, lifetime `downloaded_bytes` — see `sort` below). Windowed trending (`1d`, `7d`, `30d`) sums `download_stats_daily` rows for the requested window and joins back to current uploads/files so deleted uploads and unavailable files do not appear. Stable ordering uses the selected metric first and deterministic tie-breakers afterward; rows with a zero value for the selected metric are omitted.

Trending uploads support a `sort` param: `downloads` (default) or `downloadedBytes`. Both `downloadCount` and `downloadedBytes` are always populated on the response regardless of which one is selected — the "all" query selects both `uploads.download_count` and `uploads.downloaded_bytes` in one pass, and the windowed query sums both `download_stats_daily.downloads` and `.bytes` in one pass, so switching sorts never costs a second query. Trending files has no `sort` param (no lifetime per-file byte column exists — file-grain byte trending is out of scope), so `downloadedBytes` stays `0` on trending file items.

Trending uploads has both an admin (server-wide) and a self-scoped endpoint; trending files is admin-only:

- `GET /stats/trending/uploads?window=all|1d|7d|30d&sort=downloads|downloadedBytes&limit=20` (admin)
- `GET /stats/trending/files?window=all|1d|7d|30d&limit=20` (admin)
- `GET /me/stats/trending/uploads?window=all|1d|7d|30d&sort=downloads|downloadedBytes&limit=20` (authenticated user, restricted to their own uploads)

The self-scoped variant filters on `uploads.user = userID` via the same live-uploads join the admin query uses (not on the rollup's own `user_id`), so it inherits the same "currently available uploads only" guarantee.

The HTTP handlers own all trending input validation via `parseTrendingWindow`/`parseTrendingSort`/`parseTrendingLimit` (`server/handlers/misc.go`, shared by the admin and self-scoped handlers): `window` must be one of `all`/`1d`/`7d`/`30d` (else `400`), `sort` must be empty/`downloads`/`downloadedBytes` (else `400`), and `limit` defaults to `20`, must be positive (else `400`), and is capped at `100`. The metadata trending queries trust these validated inputs and no longer re-clamp the limit or reject bad windows/sorts.

#### Migration Backfill

Migration `0011-stats` creates `usage_stats`, `download_stats_daily`, and `upload_stats_daily` and backfills per-user, per-token, and anonymous counters from retained metadata. It seeds `lifetime_users = 1` on every backfilled user row (so the server sum-on-read of `lifetime_users` equals the number of users present at migration time) and creates the zero-valued `("__deleted__","")` tombstone; there is no server row to backfill. gormigrate runs with `UseTransaction: false`, so the backfill statements autocommit individually and the migration body is **not** atomic. The backfill is therefore written to be idempotent and safely re-runnable:

Because gormigrate's `InitSchema` marks all migrations as applied on a brand-new database without running their bodies, the tombstone (and the anonymous row) are also created on every startup by `ensureBaseUsageStatsRows`, which additionally deletes any stray legacy `("","")` server row so it can never double-count into the sum. This is what anchors the server `StartedAt` on fresh installs where 0011's body never executes.

- It wipes `usage_stats` (a table it owns) before recomputing every row, so a re-run after a partial failure never collides on the `(user_id, token)` primary key.
- It scopes each aggregate with an upload-id subquery (`upload_id IN (SELECT id FROM uploads WHERE ...)`) instead of materializing upload ids into `WHERE ... IN (?)` bind lists. This keeps the backfill under SQLite's 32,766 / PostgreSQL's 65,535 parameter limits on large instances.

Download-event counters (`downloads`) are backfilled by summing retained `uploads.download_count`. `usage_stats.downloaded_bytes` now technically *has* a per-upload source too — `uploads.downloaded_bytes`, introduced alongside `download_count` in this same still-open migration — but it is **deliberately not used** to backfill or rebuild `usage_stats`: `CreateUpload`'s usage delta replays `download_count` into `usage_stats.downloads` exactly once (`delta.downloads = upload.DownloadCount`) but never does the equivalent for `downloaded_bytes`, so a future reseed of the byte counter (`fakedb`'s `FixtureSeedDownloadedBytes`, or any similar backfill/import helper) cannot double-count against it. `usage_stats.uploaded_bytes` remains fully unreconstructable — there is still no per-upload or per-file wire-byte source at all. So both `usage_stats` byte counters start at `0` on migration backfill and on import; only the upload row's own `downloaded_bytes` column (like `download_count`) is backfilled/imported with real values. Import behaves the same way: daily rollup `bytes` are restored verbatim from the exported rollup records, `uploads.downloaded_bytes` rides along on the imported upload record itself, while the rebuilt usage rows still start at `0` bytes. The migration does not backfill `upload_stats_daily` from historical upload creation dates (those buckets would be immediately pruned by the today+30 retention); the live `CreateUpload` path fills it going forward.

Recovery from a failed or interrupted 0011 migration is simply **restarting the server**: the migration id is only recorded on success, so an aborted run re-executes on the next start, clears any partial rows, and rebuilds correct counters — no manual database surgery is required. The migration's `Rollback` drops both stats tables.

#### Import, Export, and Repair

Exports include daily download **and upload** rollups because they cannot be derived from current metadata and are required to preserve the `1d`/`7d`/`30d` windows and chart series after restore. User/token/anonymous usage counters are intentionally rebuilt on import from the imported users/tokens/uploads/files (server totals are then summed on read over those rows); daily rollups are restored as their own records. `download_stats_daily` records import via a plain insert; `upload_stats_daily` records import via an upsert (absolute overwrite) because `CreateUpload` replays the day's `uploads` +1 (with `bytes=0`) during import, and the imported record then restores the authoritative `{uploads, bytes}` row instead of colliding on the `(day, user_id, token)` primary key. Old exports simply lack `upload_stats_daily` records, so those tables start empty (zeros) after importing a pre-upgrade dump.

The `("__deleted__","")` tombstone is the **only** usage row exported as its own record: a deleted user's uploads are gone, so its folded lifetime counters (including `lifetime_users`) have no other rebuild source. On import it is folded into the init-created tombstone (imported users replay `CreateUser` and get their own `lifetime_users`, so nothing is double-counted). Old exports lack the tombstone record, so a deleted user's contribution is absent (zeros) after importing a pre-upgrade dump — the same documented "rebuilt from what is present at export time" approximation as the rest of the rebuild. Aggregate scans and backfill helpers should stay in migration/import/repair-style paths; request-time APIs should use counter rows.

**Imports target a fresh DB**: every other imported object enforces this itself by failing on its primary-key conflict when replayed into an already-populated destination, but `ImportDeletedUsageTombstone`'s fold (`col = col + src`) has no such guard — replaying the same tombstone-bearing export a second time into a non-fresh DB silently double-counts its lifetime counters. This is accepted, documented behavior under the fresh-DB contract, not a bug (see `TestBackend_ImportDeletedTombstoneTwiceDoubleCountsLifetimeCounters`).

If counters ever drift because of a historical bug or manual database edits, the intended repair strategy is to rebuild usage counters from retained metadata and preserved rollups. Lifetime stats remain exact only for metadata and rollups that still exist.

Import (`CreateFile`) and migration backfill (`backfillFileStats`) share one deliberate approximation for lifetime file/size counters: both lifetime-count every file whose final status is `removed` or `deleted`, because a bare status column cannot distinguish a file that was successfully uploaded and later removed (which the live path lifetime-counts once, at completion) from one that never left `uploading` before its upload was removed (which the live path never lifetime-counts, per the file status transition table above). Persisting a marker to disambiguate the two cases was judged not worth the schema weight, so both rebuild paths count the same three statuses (`uploaded`/`removed`/`deleted`) and therefore always agree with each other exactly. What they do not agree with is the live, incrementally-accumulated counters: rebuilt (imported or backfilled) lifetime file/size totals can be slightly higher than what live accumulation would have recorded for the same history, because failed-upload cleanup files get folded in alongside genuinely completed ones. This is accepted, documented behavior, not a bug.

### Import / Export

The `plikd export` and `plikd import` commands dump and restore metadata (users, tokens, uploads, files, settings, and daily download and upload stats) to/from a single binary file.

- **Format**: Go [gob](https://pkg.go.dev/encoding/gob) encoding compressed with [Snappy](https://github.com/golang/snappy). Architecture-independent (portable across `amd64`/`arm64`), streaming (constant memory), Go-specific (not human-readable).
- **Export order**: users → tokens → uploads (including soft-deleted) → files → settings → daily download stats → daily upload stats → deleted-user usage tombstone. CLI auth sessions are intentionally excluded (ephemeral). User/token/anonymous usage stats are rebuilt from imported metadata (server totals are summed on read); only the deleted-user tombstone is exported as a record.
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
| `get_file.go` | `GetFile` | Download file, handle OneShot, extend TTL, support HTTP range requests (via `http.ServeContent` for non-stream/non-oneshot). Post-stream it counts a download event for full GETs and byte-0 `Range` GETs only, and records bytes served for every response that streams file bytes (incl. mid-range GETs, which are bytes-only). E2EE uploads: redirects webapp to download page, forces `application/octet-stream` |
| `get_archive.go` | `GetArchive` | Download all files as zip and, after the archive closes successfully, record one upload download plus one download per included file and the zip stream's bytes served; on a mid-stream failure, record the bytes already served as a bytes-only event instead (no download counted), matching the single-file egress policy. Compression method controlled by `EnableArchiveCompression` (default: `zip.Deflate`). Disable to prevent CPU exhaustion DoS on public instances. |
| `remove_file.go` | `RemoveFile` | Mark file as removed. Also mapped to `DELETE /stream/...` to allow cancelling a blocked stream upload (closes the in-memory pipe). |
| `remove_upload.go` | `RemoveUpload` | Soft-delete upload |
| `misc.go` | `GetConfiguration`, `GetVersion`, `GetQrCode`, `Health` | Utility endpoints |
| `local.go` | `LocalLogin`, `Logout` | Local auth |
| `google.go` | `GoogleLogin`, `GoogleCallback` | Google OAuth with PKCE S256 (RFC 7636) |
| `ovh.go` | `OvhLogin`, `OvhCallback` | OVH OAuth |
| `oidc.go` | `OIDCLogin`, `OIDCCallback` | OpenID Connect with PKCE S256 (RFC 7636) |
| `github.go` | `GitHubLogin`, `GitHubCallback` | GitHub OAuth with PKCE S256 (RFC 7636); optional org restriction |
| `cli_auth.go` | `CLIAuthInit`, `CLIAuthApprove`, `CLIAuthPoll` | CLI device auth flow |
| `me.go` | `UserInfo`, `PatchMe`, `DeleteAccount`, `GetUserStatistics`, `GetUserActivityDaily`, `GetUserTrendingUploads`, `GetUserUploads`, `RemoveUserUploads`, `GetUserTokens` | Current user; token filter param accepts both prefixed and legacy UUID formats (no DB lookup, works for revoked tokens); `GetUserUploads` (`GET /me/uploads`) supports `date`/`size`/`downloads`/`downloadedBytes` sorting; `GetUserTokens` (`GET /me/token`) includes counter-backed stats and supports `date`/`size`/`lifetimeSize` sorting only (no `downloadedBytes`); `GetUserTrendingUploads` (`GET /me/stats/trending/uploads`) is the self-scoped counterpart of `admin.go`'s `GetTrendingUploads`, sharing its window/sort/limit parsing |
| `token.go` | `CreateToken`, `RevokeToken` | Token create/revoke |
| `user.go` | `CreateUser`, `UpdateUser` | User create/update |
| `admin.go` | `GetServerStatistics`, `GetServerActivityDaily`, `GetUploads`, `GetTrendingUploads`, `GetTrendingFiles`, `GetUsers`, `SearchUsers` | Admin endpoints; `GetUploads` (`GET /uploads`) supports `date`/`size`/`downloads`/`downloadedBytes` sorting; `GetUsers`/`SearchUsers` (`GET /users`, `GET /users/search`) include counter-backed stats and support `date`/`size`/`lifetimeSize`/`downloadedBytes` sorting (`SearchUsers` itself has no sort param — it always orders by login); `GetTrendingUploads` (`GET /stats/trending/uploads`) supports a `downloads`/`downloadedBytes` sort, `GetTrendingFiles` (`GET /stats/trending/files`) does not |

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
