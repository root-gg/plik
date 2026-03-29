# Fix Path + Domain URL Construction — Centralize & Harden

## Background

File download URL construction is duplicated across the server, Go client library, and webapp, with inconsistent handling of the `Path` config option. `DownloadDomain` never inherits `Path`, producing broken URLs in subpath deployments. Additionally, `PlikDomain` and `DownloadDomain` silently accept path components that are either ignored or conflict with `Path`.

### Current bugs

| Scenario | Expected | Actual | Where |
|---|---|---|---|
| `DownloadDomain` + `Path` (quick mode) | `https://dl.io/sub/file/...` | `https://dl.io/file/...` | server `add_file.go` |
| `DownloadDomain` + `Path` (CLI show) | `https://dl.io/sub/file/...` | `https://dl.io/file/...` | server `cmd/file.go` |
| `DownloadDomain` + `Path` (Go client) | `https://dl.io/sub/file/...` | `https://dl.io/file/...` | client `plik/file.go` |
| `DownloadDomain` + `Path` (webapp) | `https://dl.io/sub/file/...` | `https://dl.io/file/...` | webapp `api.js` |
| Domain path ≠ `Path` | Error or strip | Silently wrong | config parsing |
| Download domain → Plik redirect + `Path` | Include Path | Path missing | middleware |

### Design decision: additive `DownloadURL` field

Rather than mutating the existing `DownloadDomain` field (breaking change), we add a new `DownloadURL` field that contains the fully computed download base URL (domain + Path). This is set in both:

- **`/config` response** (`Configuration.DownloadURL`) — for the webapp
- **Upload API responses** (`Upload.DownloadURL`) — for CLI/Go client

`DownloadDomain` remains unchanged for backward compatibility. Updated clients prefer `DownloadURL` when present.

---

## Proposed Changes

### 1. Config Validation — Warn & Strip Domain Paths

#### [MODIFY] [config.go](file:///home/cam/git/plik/server/common/config.go)

In `Initialize()`, after parsing `PlikDomain` and `DownloadDomain`, warn and strip any path component. `Path` is the single source of truth for the URL path prefix.

**PlikDomain block** (around L272-278):
```go
if config.PlikDomain != "" {
    config.PlikDomain = strings.Trim(config.PlikDomain, "/ ")
    if config.plikDomainURL, err = url.Parse(config.PlikDomain); err != nil {
        return fmt.Errorf("invalid plik domain URL %s : %s", config.PlikDomain, err)
    }
    if config.plikDomainURL.Path != "" {
        log.Warningf("PlikDomain %q contains a path component %q which will be ignored — use the Path config option instead", config.PlikDomain, config.plikDomainURL.Path)
        config.plikDomainURL.Path = ""
        config.PlikDomain = config.plikDomainURL.String()
    }
}
```

**DownloadDomain block** (around L280-285) — same pattern.

**DownloadDomainAlias** — strip+warn for consistency (aliases are only used for host-matching, but keeping paths would be confusing):
```go
for _, alias := range config.DownloadDomainAlias {
    domainAlias, err := url.Parse(alias)
    if err != nil { ... }
    if domainAlias.Path != "" {
        log.Warningf("DownloadDomainAlias %q contains a path component which will be ignored", alias)
        domainAlias.Path = ""
    }
    config.downloadDomainURLAlias = append(config.downloadDomainURLAlias, domainAlias)
}
```

`Initialize()` gains a `*logger.Logger` parameter to emit warnings. Signature changes from `func (config *Configuration) Initialize() error` to `func (config *Configuration) Initialize(log *logger.Logger) error`. Update callers: `LoadConfiguration()` and tests.

---

### 2. Add `DownloadURL` Computed Field

#### [MODIFY] [config.go](file:///home/cam/git/plik/server/common/config.go)

Add `DownloadURL` to the `Configuration` struct — computed, not stored in TOML. Populated in `Initialize()`:

```go
type Configuration struct {
    // ... existing fields ...
    DownloadURL string `json:"downloadURL,omitempty" toml:"-"` // Computed: DownloadDomain + Path
}
```

Set in `Initialize()` after domain parsing + path stripping:
```go
if config.downloadDomainURL != nil {
    u := *config.downloadDomainURL
    u.Path = config.Path
    config.DownloadURL = u.String()
}
```

#### [MODIFY] [upload.go](file:///home/cam/git/plik/server/common/upload.go)

Add `DownloadURL` to the `Upload` struct alongside existing `DownloadDomain`:

```go
type Upload struct {
    // ... existing fields ...
    DownloadDomain string `json:"downloadDomain,omitempty" gorm:"-"`  // existing, kept for compat
    DownloadURL    string `json:"downloadURL,omitempty" gorm:"-"`     // new: domain + path
}
```

Update `Sanitize()` at L110:
```go
upload.DownloadDomain = config.DownloadDomain  // kept for backward compat
upload.DownloadURL = config.DownloadURL        // new: includes Path
```

---

### 3. URL Construction Helpers

#### [MODIFY] [config.go](file:///home/cam/git/plik/server/common/config.go)

Add two new methods:

```go
// GetDownloadBaseURL returns the base URL for file download links.
// Uses DownloadDomain + Path when configured, otherwise falls back to GetServerURL().
func (config *Configuration) GetDownloadBaseURL() *url.URL {
    if config.downloadDomainURL != nil {
        u := *config.downloadDomainURL
        u.Path = config.Path
        return &u
    }
    return config.GetServerURL()
}

// GetFileURL returns the full download URL for a file.
// When stream is true, uses the /stream/ endpoint instead of /file/.
func (config *Configuration) GetFileURL(uploadID, fileID, fileName string, stream bool) string {
    mode := "file"
    if stream {
        mode = "stream"
    }
    u := config.GetDownloadBaseURL()
    u.Path += fmt.Sprintf("/%s/%s/%s/%s", mode, uploadID, fileID, url.PathEscape(fileName))
    return u.String()
}

// GetArchiveURL returns the full download URL for an upload archive.
func (config *Configuration) GetArchiveURL(uploadID, archiveName string) string {
    u := config.GetDownloadBaseURL()
    u.Path += fmt.Sprintf("/archive/%s/%s", uploadID, url.PathEscape(archiveName))
    return u.String()
}
```

---

### 4. Fix Download Domain Redirect

#### [MODIFY] [download_domain.go](file:///home/cam/git/plik/server/middleware/download_domain.go)

The redirect at L61 uses `config.PlikDomain` raw but needs the `Path` prefix (after `StripPrefix` runs, `req.URL.RequestURI()` no longer contains it):

```diff
-    redirectURL := fmt.Sprintf("%s%s", config.PlikDomain, req.URL.RequestURI())
+    redirectURL := fmt.Sprintf("%s%s%s", config.PlikDomain, config.Path, req.URL.RequestURI())
```

---

### 5. Server-side Consumers

#### [MODIFY] [add_file.go](file:///home/cam/git/plik/server/handlers/add_file.go)

Replace L216–227 (quick-mode URL construction):

```diff
 if ctx.IsQuick() {
-    var fileURL string
-    if ctx.GetConfig().GetDownloadDomain() != nil {
-        fileURL = ctx.GetConfig().GetDownloadDomain().String()
-    } else {
-        fileURL = ctx.GetConfig().GetServerURL().String()
-    }
-    fileURL += fmt.Sprintf("/file/%s/%s/%s", upload.ID, file.ID, url.PathEscape(file.Name))
+    fileURL := ctx.GetConfig().GetFileURL(upload.ID, file.ID, file.Name)
     _, _ = resp.Write([]byte(fileURL + "\n"))
```

#### [MODIFY] [file.go](file:///home/cam/git/plik/server/cmd/file.go)

Replace L145 (also fixes missing `PathEscape` on filename):

```diff
-    fmt.Printf("File URL : %s/file/%s/%s/%s\n", config.GetServerURL(), file.UploadID, file.ID, file.Name)
+    fmt.Printf("File URL : %s\n", config.GetFileURL(file.UploadID, file.ID, file.Name))
```

---

### 6. Go Client Library

#### [MODIFY] [file.go](file:///home/cam/git/plik/plik/file.go)

Update `GetURL()` to prefer `DownloadURL` over `DownloadDomain`:

```diff
     var domain string
-    if uploadMetadata.DownloadDomain != "" {
-        domain = uploadMetadata.DownloadDomain
+    if uploadMetadata.DownloadURL != "" {
+        domain = uploadMetadata.DownloadURL
+    } else if uploadMetadata.DownloadDomain != "" {
+        domain = uploadMetadata.DownloadDomain
     } else {
         domain = file.upload.client.URL
     }
```

---

### 7. Webapp

#### [MODIFY] [api.js](file:///home/cam/git/plik/webapp/src/api.js)

Rename `setDownloadDomain` → `setDownloadURL` (or keep both for clarity):

```diff
-let _downloadDomain = ''
+let _downloadURL = ''

-export function setDownloadDomain(domain) {
-    _downloadDomain = domain || ''
+export function setDownloadURL(url) {
+    _downloadURL = url || ''
 }

 function downloadBase() {
-    return _downloadDomain || base
+    return _downloadURL || base
 }
```

#### [MODIFY] [config.js](file:///home/cam/git/plik/webapp/src/config.js)

Update the initialization to use `downloadURL`:

```diff
-import { getConfig, getVersion, setDownloadDomain } from './api.js'
+import { getConfig, getVersion, setDownloadURL } from './api.js'

-    setDownloadDomain(config.downloadDomain)
+    setDownloadURL(config.downloadURL || config.downloadDomain || '')
```

The `|| config.downloadDomain` provides backward compatibility in case the webapp talks to an older server that doesn't have `downloadURL` yet.

---

### 8. Unit Tests

#### [MODIFY] [config_test.go](file:///home/cam/git/plik/server/common/config_test.go)

- `TestInitializeConfigPlikDomainWithPath` — warn, strip path, keep scheme+host
- `TestInitializeConfigDownloadDomainWithPath` — warn, strip, set `DownloadURL` correctly
- `TestInitializeConfigDownloadDomainAliasWithPath` — warn, strip alias paths
- `TestGetDownloadBaseURL_NoDownloadDomain` — falls back to `GetServerURL()`
- `TestGetDownloadBaseURL_WithDownloadDomain` — uses download domain
- `TestGetDownloadBaseURL_WithDownloadDomainAndPath` — includes Path
- `TestGetFileURL` / `TestGetFileURL_WithPath` / `TestGetFileURL_WithDownloadDomainAndPath`
- `TestGetFileURL_Stream` — stream mode produces `/stream/` prefix
- `TestGetArchiveURL` / `TestGetArchiveURL_WithPath`

#### [MODIFY] [upload_test.go](file:///home/cam/git/plik/server/common/upload_test.go)

- Test `Sanitize()` populates both `DownloadDomain` and `DownloadURL`

#### [MODIFY] [add_file_test.go](file:///home/cam/git/plik/server/handlers/add_file_test.go)

- `TestAddFileQuickWithPath` — quick mode + Path
- `TestAddFileQuickDownloadDomainWithPath` — regression test for original bug

#### [MODIFY] [download_domain_test.go](file:///home/cam/git/plik/server/middleware/download_domain_test.go)

- `TestRestrictDownloadDomainRedirectWithPath` — redirect includes Path

---

### 9. Backend E2E Tests

#### [MODIFY] [z2_e2e_config_test.go](file:///home/cam/git/plik/plik/z2_e2e_config_test.go)

New tests exercising Path + DownloadDomain in full server round-trips:

- `TestDownloadDomainWithPath` — upload + download with `DownloadDomain` + `Path` set, verify the file download URL contains the Path prefix
- `TestDownloadDomainRedirectWithPath` — verify the redirect from download domain to PlikDomain includes the Path prefix in the Location header
- `TestDownloadURLMetadata` — upload with `DownloadDomain` + `Path`, verify `upload.Metadata().DownloadURL` is set to `DownloadDomain + Path` and `upload.Metadata().DownloadDomain` remains the raw domain (backward compat)
- `TestDomainWithPathStripping` — set `DownloadDomain = "https://dl.plik.io/badpath"`, verify after `Initialize()` the path is stripped and `DownloadURL` uses `config.Path` instead

Update existing test at L277/L313: `Initialize()` calls need the new logger parameter.

---

### 10. Frontend E2E Tests

#### [MODIFY] [subpath.spec.js](file:///home/cam/git/plik/webapp/e2e/subpath.spec.js)

New test in the "Subpath — upload and download" describe block:

- `download link uses subpath-prefixed URL` — upload a file in the subpath deployment (`Path="/sub"`), then verify the download link (`<a>` href on the file row) includes `/sub/file/` in its URL, not just `/file/`. This catches the webapp `downloadBase()` bug.

#### [MODIFY] [start-server-subpath.sh](file:///home/cam/git/plik/webapp/e2e/start-server-subpath.sh)

If `DownloadDomain` is not currently set in the subpath test config, add it to exercise the full Path+DownloadDomain interaction in the frontend e2e suite.

---

### 11. Documentation

#### [MODIFY] [configuration.md](file:///home/cam/git/plik/docs/guide/configuration.md)

Update the domain configuration table:
- Clarify that `PlikDomain` and `DownloadDomain` should be **domain-only** (no path)
- Mention that path components are stripped with a warning
- Document the new `DownloadURL` computed field in the API response

#### [MODIFY] [security.md](file:///home/cam/git/plik/docs/guide/security.md)

- Add a note in the domain separation section about Path interaction
- Document that Path is the sole source of truth for URL prefix
- Update examples showing `DownloadDomain` with Path

#### [MODIFY] [library.md](file:///home/cam/git/plik/docs/architecture/library.md)

- Update the test table to mention new DownloadDomain+Path e2e tests

#### [MODIFY] [webapp.md](file:///home/cam/git/plik/docs/architecture/webapp.md)

- Update the config table: `downloadDomain` → still present, add `downloadURL` row
- Note the `setDownloadURL` migration from `setDownloadDomain`

#### [MODIFY] [system.md](file:///home/cam/git/plik/docs/architecture/system.md)

- Update the `RestrictDownloadDomain` row to mention Path-aware redirects

#### [MODIFY] [server.md](file:///home/cam/git/plik/docs/architecture/server.md)

- Same: update download_domain middleware description

#### [MODIFY] ARCHITECTURE.md files

- [ARCHITECTURE.md](file:///home/cam/git/plik/ARCHITECTURE.md) — update middleware table
- [server/ARCHITECTURE.md](file:///home/cam/git/plik/server/ARCHITECTURE.md) — update middleware table
- [plik/ARCHITECTURE.md](file:///home/cam/git/plik/plik/ARCHITECTURE.md) — mention `DownloadURL` field
- [webapp/ARCHITECTURE.md](file:///home/cam/git/plik/webapp/ARCHITECTURE.md) — document `setDownloadURL`

#### [MODIFY] [AGENTS.md](file:///home/cam/git/plik/AGENTS.md)

- No structural changes expected, but verify conventions and key files sections are still accurate after the changes

#### [MODIFY] [api.md](file:///home/cam/git/plik/docs/reference/api.md)

- **`GET /config` response**: document new `downloadURL` field (domain + Path, replaces manual concatenation)
- **`POST /upload` / `GET /upload/{uploadID}` response**: document new `downloadURL` field alongside existing `downloadDomain`
- Note backward compatibility: `downloadDomain` remains in all responses; `downloadURL` is additive

---

## Files Changed Summary

| File | Change |
|------|--------|
| **Server** | |
| `server/common/config.go` | Warn+strip domain paths, add `DownloadURL` field, `GetDownloadBaseURL()` + `GetFileURL()` + `GetArchiveURL()`, `Initialize()` logger param |
| `server/common/upload.go` | Add `DownloadURL` field to `Upload`, populate in `Sanitize()` |
| `server/common/config_test.go` | Unit tests for path stripping, URL helpers, DownloadURL |
| `server/common/upload_test.go` | Test Sanitize sets DownloadURL |
| `server/handlers/add_file.go` | Use `GetFileURL()` in quick mode |
| `server/handlers/add_file_test.go` | Path regression tests |
| `server/cmd/file.go` | Use `GetFileURL()` in CLI |
| `server/middleware/download_domain.go` | Fix redirect to include Path |
| `server/middleware/download_domain_test.go` | Test redirect with Path |
| **Client library** | |
| `plik/file.go` | Prefer `DownloadURL` over `DownloadDomain` |
| `plik/z2_e2e_config_test.go` | E2E tests for DownloadDomain+Path, redirect with Path, DownloadURL metadata |
| **Webapp** | |
| `webapp/src/api.js` | Use `downloadURL` instead of `downloadDomain` |
| `webapp/src/config.js` | Initialize with `downloadURL`, fallback to `downloadDomain` |
| `webapp/e2e/subpath.spec.js` | E2E test: download link uses subpath-prefixed URL |
| `webapp/e2e/start-server-subpath.sh` | Add DownloadDomain to subpath test config (if needed) |
| **Documentation** | |
| `docs/guide/configuration.md` | Domain-only clarification, DownloadURL |
| `docs/guide/security.md` | Path interaction notes |
| `docs/architecture/library.md` | Updated test table |
| `docs/architecture/webapp.md` | `downloadURL` config row |
| `docs/architecture/system.md` | Path-aware redirect note |
| `docs/architecture/server.md` | Middleware description update |
| `ARCHITECTURE.md` | Middleware table update |
| `server/ARCHITECTURE.md` | Middleware table update |
| `plik/ARCHITECTURE.md` | DownloadURL field |
| `webapp/ARCHITECTURE.md` | setDownloadURL |
| `docs/reference/api.md` | Document `downloadURL` in `/config` and upload API responses |
| `AGENTS.md` | Verify accuracy after changes |

## Decisions (confirmed)

- **Logger in `Initialize()`**: ✅ Accepted — add `*logger.Logger` param, update `LoadConfiguration()` and tests
- **DownloadDomainAlias path stripping**: ✅ Strip+warn for consistency

## Verification Plan

### Automated Tests
```bash
# Unit tests
cd server && go test ./common/ -run "TestInitializeConfigPlikDomainWithPath|TestInitializeConfigDownloadDomainWithPath|TestGetDownloadBaseURL|TestGetFileURL|TestGetArchiveURL" -v
cd server && go test ./common/ -run "TestSanitize" -v
cd server && go test ./handlers/ -run "TestAddFileQuick" -v
cd server && go test ./middleware/ -run "TestRestrictDownloadDomainRedirect" -v

# Backend e2e
cd plik && go test -run "TestDownloadDomainWithPath|TestDownloadDomainRedirectWithPath|TestDownloadURLMetadata|TestDomainWithPathStripping" -v

# Full suite
make lint && make test

# Frontend
make test-frontend
make test-frontend-e2e
```

### Manual Verification
- Quick-mode upload with `Path = "/sub"` + `DownloadDomain` → URL includes `/sub/file/...`
- `plikd file show` uses download domain + path when set
- Download domain redirect preserves the Path prefix
- Webapp file download links use `downloadURL` with correct path
- Documentation builds: `make docs`

