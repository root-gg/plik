# Architecture — Go Library (`plik/`)

> Public Go client library for the Plik API. For system-wide overview, see the root [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Structure

```
plik/
├── README.md          ← library usage documentation
├── client.go          ← Client type — server connection and config
├── upload.go          ← Upload type — create/manage uploads
├── file.go            ← File type — upload/download files
├── internal.go        ← internal HTTP request helpers
├── setup_test.go      ← test server setup
├── *_test.go          ← unit tests
└── z*_e2e_*_test.go   ← end-to-end tests (ordered by z-prefix)
```

---

## Public API

### `Client` (`client.go`)

Factory for creating uploads. Holds server URL, login/password, and upload token.

```go
client := plik.NewClient("http://localhost:8080")
```

### `Upload` (`upload.go`)

Represents an upload with options. Created via `client.NewUpload()`.

```go
upload := client.NewUpload()
upload.OneShot = true
upload.TTL = 3600
upload.Token = "xxxx-xxxx-xxxx"   // Optional: CLI token to link upload to a user

file := upload.AddFileFromReader("myfile.txt", reader)
err := upload.Upload()            // POST /upload + upload files
```

`Login`/`Password` on `UploadParams` are for downloading **password-protected** uploads, not user authentication.

### `File` (`file.go`)

Represents a file within an upload. Supports upload, download, and removal.

---

## E2E Test Suite

The e2e tests are the most comprehensive test layer in Plik. They spin up a **real plikd server** in-process and exercise the full HTTP API through the Go client library. The `z` prefix on filenames ensures Go runs them after unit tests (Go sorts test functions by file name within a package).

### Test Infrastructure (`setup_test.go`)

#### `TestMain` — Global Bootstrap

`TestMain` runs once before all tests in the package. It sets up the **data backend** and **metadata backend** that all tests share:

1. **Config loading**: If `PLIKD_CONFIG` env var is set, loads a full `plikd.cfg` from that path (used by the Docker-based backend tests in `testing/`). Otherwise, creates a default config.
2. **Data backend selection**: Based on `testConfig.DataBackend`:
   - `testing` (default) — in-memory backend, no external deps
   - `file` — temp directory, cleaned up after all tests
   - `s3` — real S3/MinIO, configured via `data_backend_config` env var
   - `swift` — real OpenStack Swift
3. **Metadata backend**: Defaults to SQLite3 at `/tmp/plik.test.db` with `EraseFirst: true` (wipes DB on startup). Overridden by `PLIKD_CONFIG` for PostgreSQL/MySQL testing.

> ⚠️ **Important**: Backends are NOT automatically cleared between individual tests. Each test creates uploads that persist across the test run. Tests must be written to not depend on a clean state.

#### `newPlikServerAndClient` — Per-Test Server Setup

Each test function calls `newPlikServerAndClient()` to get a fresh server + client pair:

```go
ps, pc := newPlikServerAndClient()
defer shutdown(ps)
err := startWithClient(ps, pc)
```

This creates:
- A new `PlikServer` with a fresh config listening on `127.0.0.1` with ephemeral port allocation (port 0 → OS assigns a free port)
- Injects the shared data + metadata backends
- A new `Client` whose URL is set by `startWithClient` to the actual listen address after boot
- Auto-clean is disabled (`config.AutoClean(false)`) so TTL-based cleanup doesn't interfere with tests

The `start()` helper starts the server and waits until the HTTP port is accepting connections. `startWithClient()` additionally updates the client URL with the actual port. `shutdown()` calls `ShutdownNow()` (immediate, no graceful drain).

Each test gets its own server instance with its own router/middleware stack, but they all share the same underlying data and metadata backends. This means tests can configure the server differently (e.g., enable/disable features, set limits) without affecting other tests' server config, while still sharing the same storage layer.

#### Test Utilities

- **`LockedReader`** — wraps an `io.Reader` with a channel-based lock. `Read()` blocks until `Unleash()` is called. Used to test concurrent upload/download scenarios and verify file status transitions (`missing` → `uploading` → `uploaded`).
- **`NewSlowReader` / `NewSlowReaderRandom`** — variant that auto-unleashes after a random delay (0–1s). Used in concurrency stress tests to simulate realistic upload timing.

### Test Files

| File | Focus | Key Scenarios |
|------|-------|---------------|
| `z1_e2e_test.go` | Core upload/download flows | Upload twice (error), download during upload (status transitions), OneShot (delete after download), Removable, Stream mode, TTL expiration, Quick upload (curl-style multipart), forbidden options sanitization |
| `z2_e2e_config_test.go` | Server config enforcement | Path prefix, MaxFileSize, MaxFilePerUpload, AnonymousUpload disabled, DefaultTTL, TTL limits, Password/OneShot/Removable disabled, DownloadDomain + aliases, UploadWhitelist, SourceIpHeader, `DownloadURL` metadata (= `DownloadDomain + Path`), path-aware file URL construction, domain path stripping |
| `z3_e2e_authentication_test.go` | Auth flows & token isolation | CLI token auth, multi-token isolation (user can't control uploads from different token), admin token isolation (admin tokens don't grant extra power — by design, to limit impact of token leak), password-protected uploads |
| `z4_e2e_client_concurrent_test.go` | Concurrency & race conditions | Multiple uploads in parallel, multiple files per upload in parallel, busy-loop upload (continuous `Upload()` calls with concurrent file adds), parallel download of same file, parallel create + get |
| `z5_e2e_browser_auth_test.go` | Browser-like authentication | Local login (cookie jar + XSRF flow), invalid password, login disabled, OIDC login via Keycloak (full redirect flow with form parsing), OIDC redirect URL validation. Uses `insecureCookieJar` to strip `Secure` flag for HTTP-only test env. Requires Keycloak for OIDC tests (skipped if unavailable). |
| `z6_e2e_cli_login_test.go` | CLI device auth flow | CLI login init, poll, and token generation via `/auth/cli/*` endpoints |

### Running

```bash
# Default (in-memory backends, fast)
cd plik && go test ./...

# With a specific data backend
data_backend=file go test ./...

# With full backend config (used by testing/ Docker scripts)
PLIKD_CONFIG=/path/to/test.cfg go test ./...
```

See also [testing/ARCHITECTURE.md](../testing/ARCHITECTURE.md) for Docker-based backend integration tests that drive these e2e tests against real database and storage backends.

