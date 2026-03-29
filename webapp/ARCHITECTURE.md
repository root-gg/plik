# Plik Webapp — Architecture & Gotchas

> Non-obvious details, design decisions, and pitfalls that agents should know before iterating on this codebase. For system-wide overview, see the root [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Tech Stack

| Layer       | Tech                          |
|-------------|-------------------------------|
| Framework   | Vue 3 (Composition API, `<script setup>`) |
| Router      | Vue Router 4, hash history (`#/`) |
| i18n        | vue-i18n v11 (Composition API mode, `useI18n()` / global `$t()`) |
| Styling     | Tailwind CSS v4 (via `@import "tailwindcss"`) with custom `@utility` and `@theme` blocks |
| Code Editor | CodeMirror 6 (`@codemirror/language-data` for syntax, `@codemirror/theme-one-dark`) |
| Build       | Vite                          |
| HTTP        | `fetch()` for JSON APIs, `XMLHttpRequest` for file uploads (progress tracking) |
| Backend     | Go (Plik server, serves the SPA from `webapp/dist/` via `http.FileServer`) |

---

## Routing & URL Format

All routes use hash-history (`#/`):

| Route          | View            | Purpose                                   |
|----------------|-----------------|-------------------------------------------|
| `/#/`          | `RootView`      | Upload (no query) or Download (`?id=...`) |
| `/#/login`     | `LoginView`     | Local + OAuth login                       |
| `/#/home`      | `HomeView`      | User dashboard (uploads, tokens, account) |
| `/#/admin`     | `AdminView`     | Admin panel (stats, users, all uploads)   |
| `/#/clients`   | `ClientsView`   | CLI client downloads                      |
| `/#/cli-auth`  | `CLIAuthView`   | Approve CLI device auth login             |
| `/#/upload/:id`| (redirect)      | Legacy URL → `/?id=:id`                   |

Admin link (upload-level): `/#/?id=<uploadId>&uploadToken=<token>`
Deep link to a specific file: `/#/?id=<uploadId>&file=<fileId>`
Deep link to a media timestamp: `/#/?id=<uploadId>&file=<fileId>&t=<seconds>`

`RootView.vue` checks `route.query.id` — if present, renders `DownloadView`; otherwise `UploadView`.

### Tab Routes & Filter Query Parameters

HomeView and AdminView use path-based tab segments for the active tab and query parameters for filter state, enabling bookmarking, sharing, and browser back/forward navigation.

**HomeView** — `/#/home/:tab`:

| Path | Tab |
|------|-----|
| `/#/home/stats` | Stats (default — `/#/home` redirects here) |
| `/#/home/uploads` | Uploads |
| `/#/home/tokens` | Tokens |

> **Security**: Token filter values (raw UUIDs) are intentionally NOT included in the URL. They remain in-memory only.

**AdminView** — `/#/admin/:tab`:

| Path | Tab |
|------|-----|
| `/#/admin/stats` | Stats (default — `/#/admin` redirects here) |
| `/#/admin/users` | Users |
| `/#/admin/uploads` | Uploads |

Filter/sort state is appended as query parameters (e.g. `/#/admin/users?provider=local&admin=true`):

| Param | Values | Default | Tab | Notes |
|-------|--------|---------|-----|-------|
| `user` | user ID | — | uploads | Filter uploads by user |
| `sort` | `date`, `size` | `date` | uploads/users | Sort field |
| `order` | `desc`, `asc` | `desc` | uploads/users | Sort direction |
| `provider` | `local`, `google`, `github`, `ovh`, `oidc` | — | users | Filter by auth provider |
| `admin` | `true`, `false` | — | users | Filter by admin role |

> **Security**: Token filter values are NOT included in admin upload URLs — they contain full API tokens that would leak in browser history, Referer headers, and shared links.

**Sync strategy**: Tab changes use `router.push()` (creates history entries — back/forward works between tabs). Filter changes use `router.replace()` (avoids cluttering history with each filter tweak). Router constraints (`/:tab(stats|users|uploads)`) reject invalid tab segments.

> **Gotcha**: The router uses `createWebHashHistory()`, so all URLs include `#/`. The `base` in `api.js` is computed from `window.location.origin + pathname` (without hash), so API calls go to the correct backend path.

### Auth Navigation Guard

The router's `beforeEach` guard enforces authentication in three layers (checked in order):

1. **`requiresAuth` routes** (`/#/home`, `/#/admin`): Unauthenticated users are redirected to `/#/login` with the intended destination saved in `sessionStorage` (survives OAuth round-trips).
2. **`requiresAdmin` routes** (`/#/admin`): Authenticated non-admin users are redirected to `/`.
3. **Forced authentication** (`config.feature_authentication === "forced"`): All other routes redirect unauthenticated users to `/#/login`, except:
   - The login page itself (`to.name === 'login'`)
   - CLI client downloads (`to.name === 'clients'`) — so users can get the CLI without logging in
   - Download pages (`to.name === 'root' && to.query.id`) — so shared links still work

CLI auth approval (`to.name === 'cli-auth'`) always requires authentication regardless of auth mode.

> **Gotcha**: In `main.js`, `app.use(router)` is called inside the `Promise.all([loadConfig(), loadSettings(), checkSession()]).then(...)` callback, NOT before it. This is critical because the router's navigation guards rely on `config.feature_authentication` being loaded, and the UI needs settings (name, background, custom CSS/JS) resolved before rendering. Installing the router before these load would cause the forced-auth guard to see default values and the UI to flash with empty branding.

**Redirect preservation**: When the guard redirects to login, it saves the intended destination to `sessionStorage` (`plik-auth-redirect` key) instead of a URL query parameter. This is necessary because OAuth flows do a full-page round-trip through an external provider (Google, GitHub, OIDC, OVH), and the server callback redirects back to `/#/login` — any hash-fragment query params would be lost during this round-trip. Using sessionStorage solves this uniformly for all auth methods (local login and OAuth).

---

## Upload Token (Admin Auth)

### How it works

The Plik server generates an `uploadToken` when an upload is created. This token grants admin access (add/remove files, delete upload). It is returned as part of the upload JSON response.

### Token lifecycle

```
UploadView creates upload → server returns { id, uploadToken, ... }
  ↓
Token stored in memory (tokenStore.js) via setToken(id, token)
  ↓
Router navigates to /?id=<id>  (NO token in URL)
  ↓
DownloadView reads token from getToken(id) → sends as X-UploadToken header
```

### Admin URL sharing

The **Admin URL** (shown in DownloadView sidebar) is the only way to share admin access:
```
http://host/#/?id=<id>&uploadToken=<token>
```

When someone opens an admin URL:
1. `onMounted` in DownloadView reads `uploadToken` from `route.query`
2. Stores it in memory via `setToken(id, token)`
3. **Immediately strips it from the URL** via `router.replace()` to prevent accidental sharing

### Key rules

- **Never persist tokens** to `localStorage`, `sessionStorage`, or cookies
- **Never leave tokens in the URL** after initial load — strip immediately
- Tokens are sent as `X-UploadToken` header, never in the request body
- Token is **per-tab, per-session** — refreshing the page loses admin access (by design)
- `getUpload()` and other API calls pass the token so the server returns `admin: true` in the response

---

## File Status Values

Files in the Plik API have a `status` field with 5 possible values:

| Status      | Meaning                                           | Displayed? |
|-------------|---------------------------------------------------|------------|
| `missing`   | File entry created, waiting to be uploaded         | ✅ Yes (uploading UI) |
| `uploading` | File is currently being uploaded                   | ✅ Yes (progress bar) |
| `uploaded`  | File has been uploaded and is ready for download   | ✅ Yes      |
| `removed`   | File has been removed by user, not yet cleaned up  | ✅ Yes (greyed-out, "Removed" badge) |
| `deleted`   | File has been deleted from the data backend        | ✅ Yes (greyed-out, "Removed" badge) |

### activeFiles computed property

During active uploads (`isAddingFiles`), the top panel only shows files the user can interact with:
- **Non-streaming**: only `uploaded` files (must be complete on server)
- **Streaming**: `uploading` + `uploaded` (download works via live stream)

When not uploading (e.g. friend viewing the download page), all files are shown — including removed/deleted files, which render greyed-out with a "Removed" badge (no download/view/remove actions):

```js
const activeFiles = computed(() => {
  if (!upload.value?.files) return []
  return upload.value.files.filter(f => {
    if (f.status === 'removed' || f.status === 'deleted') return true
    if (isAddingFiles.value) {
      if (upload.value.stream) {
        return f.status === 'uploading' || f.status === 'uploaded'
      } else {
        return f.status === 'uploaded'
      }
    }
    return true
  })
})
```

> **Key design**: Files "move" from the bottom pending panel to the top active list as they become ready — non-streaming files appear when uploaded, streaming files appear when they start uploading (since their download link is immediately valid).

> **Deleted file visibility**: Removed/deleted files stay visible in the file list with `opacity-50`, a red "Removed" pill badge, a `line-through` filename (plain text, no link), and all action buttons hidden. The `totalFiles` computed (which excludes removed/deleted) drives the heading count and the auto-view watcher, so "2 files" only counts downloadable files.

> **Gotcha**: After deleting a file via the API, the server returns `"ok"` (plain text, not JSON). The file's status changes to `removed` server-side but the API **does not return the updated file object**. You must call `fetchUpload()` again to refresh the list.

> **Note**: If all files are deleted, the file list still shows the greyed-out deleted rows. The "No files in this upload" empty state only appears when `activeFiles.length === 0` (i.e. no files at all, including deleted ones).

---

## API Error Handling

### The two-pass text→JSON pattern

The Plik server returns errors as **either JSON or plain text** depending on the endpoint. The `apiCall` function handles this with a two-pass approach:

```js
// 1. Read as text first (always works)
const text = await resp.text()
// 2. Try to parse as JSON
try {
    const body = JSON.parse(text)
    message = body.message || body || message
} catch {
    // 3. Fall back to raw text (e.g., "upload abc123 not found")
    message = text || message
}
```

> **Why**: Calling `resp.json()` first consumes the body stream. If it fails (plain text response), `resp.text()` would then also fail with "body stream already read". The text-first approach avoids this.

### Network error wrapping

`apiCall` wraps `fetch()` in a try/catch to convert the browser's generic `TypeError: Failed to fetch` into a user-friendly `"Network error — server may be unreachable"`. Without this, network failures (offline, DNS, server down) surface as cryptic browser errors.

### XHR upload errors

`uploadFile` uses XHR (not fetch) for progress tracking. The server returns **plain text** errors, not JSON, so the XHR error handler uses the same two-pass pattern: try `JSON.parse`, fall back to `xhr.responseText`. The `error` event (network failure) produces `"Upload connection lost — check your network"` instead of the generic browser error.

### Error display format

Error messages include the HTTP status code when available: `"message (HTTP 404)"`. File upload errors in the banner include the filename: `"photo.jpg: file too big"`. This gives users enough context to understand what went wrong and report issues.

### Success responses

Some endpoints return **plain text** on success:
- `DELETE /upload/:id` → `"ok"`
- `DELETE /file/:uploadId/:fileId/:fileName` → `"ok"`

The `apiCall` function handles this too:
```js
const text = await resp.text()
if (!text) return null
try { return JSON.parse(text) } catch { return text }
```

### Error display

All views use the same reusable error components for consistent look and feel:

| Component | Purpose | Display |
|-----------|---------|--------|
| `ErrorState` | Full-page error when content can't be loaded (e.g. upload not found) | Centered glass-card with danger icon, message, and retry button — replaces content area |
| `ErrorBanner` | Inline error for API failures while content remains visible | Horizontal glass-card with danger icon, message, and dismiss ✕ button — sits atop content |

### Separated error states in DownloadView

DownloadView uses **two separate error refs** to avoid upload errors from hiding the upload content:

| Ref | Purpose | Display |
|-----|---------|---------|
| `error` | Page-level failures (e.g., `fetchUpload` fails, upload not found) | Full-page error state via `ErrorState` component (`v-else-if="error"`) — replaces entire content |
| `uploadError` | Non-file operational errors (reserved for future use) | Dismissible inline `ErrorBanner` within the upload content area |

> **Why two refs**: The template uses `v-if="loading"` / `v-else-if="error"` / `v-else-if="upload"` branching. If file upload errors set `error`, the `v-else-if="error"` branch takes over and hides the sidebar + file list. The `uploadError` ref keeps errors in the `v-else-if="upload"` block so the user retains context.

### Per-file error handling with retry

File upload errors are shown **per-file in the pending panel**, not in a top banner. Failed files:
- Stay in the pending panel with `status: 'error'` and a red error message
- Have a **Retry** button (per-file) and a **Retry Failed** button (bulk)
- Have a dismiss (X) button to remove them from the list
- Keep `isAddingFiles = true` so they don't appear as "Waiting for upload" in the top panel
- When retried, transition back to `status: 'toUpload'` and re-enter the upload pool

### Upload pool architecture

All upload logic is DRY across three entry points:

| Function | Purpose |
|---|---|
| `uploadFileEntry(file)` | Shared helper: XHR upload, progress, success/error handling |
| `uploadPendingFiles()` | Pool manager: concurrency-limited batch with re-check loop |
| `retryFile(file)` / `retryAllFailed()` | Reset file(s) to `toUpload`, delegate to pool |

Key design decisions:
- **`isUploading`** (non-reactive) guards pool re-entry. Separate from `isAddingFiles` (reactive, UI display).
- **`activeBasicAuth`** stored at component level so retries preserve password-protected upload credentials.
- **Re-check loop**: after each batch completes, the pool re-scans for `toUpload` files. This lets retries queue into the existing pool without bypassing `MAX_CONCURRENT`.
- **`cancelAllUploads`** calls `fetchUpload()` after a 200ms delay so the server has time to update metadata.

---

## File Upload Mechanics

### Two URL patterns for uploading files

| Scenario                   | URL pattern                                  |
|----------------------------|----------------------------------------------|
| Initial upload (has fileId from `createUpload`) | `POST /file/:uploadId/:fileId/:fileName` |
| Adding files to existing upload (no fileId)     | `POST /file/:uploadId`                   |

The `api.js` `uploadFile` function picks the right pattern:
```js
if (fileEntry.id) {
    url = `${base}/${mode}/${upload.id}/${fileEntry.id}/${fileEntry.fileName}`
} else {
    url = `${base}/${mode}/${upload.id}`
}
```

### Stream vs File mode

The URL prefix changes based on whether the upload uses streaming:
- Normal: `/file/...`
- Streaming: `/stream/...`

### Upload flow (UploadView → DownloadView)

1. `buildUploadParams()` pre-populates files (with `reference` fields) so the server assigns IDs upfront
2. `createUpload(params)` → server returns upload with `id`, `uploadToken`, and pre-created file entries (with IDs)
3. `setPendingFiles(id, files, basicAuth)` stashes files in the in-memory `pendingUploadStore` — file IDs are matched via `reference` (not array index)
4. `setToken(id, token)`, then `router.push({ path: '/', query: { id } })` — **navigates immediately**
5. DownloadView mounts, calls `consumePendingFiles(id)` to retrieve the stashed files
6. Auto-starts `uploadPendingFiles()` — uploads files concurrently (max 5 at a time) with a worker pool
7. Status updates are **local** (reactive mutations on `upload.value.files[i].status`) — no `fetchUpload()` during uploads to avoid UI flash
8. For streaming uploads: `onStart` callback marks server files as `'uploading'` → they appear in the top panel immediately
9. For all uploads: `.then()` marks server files as `'uploaded'` → they appear in the top panel
10. One final `fetchUpload()` after all uploads complete to sync with server truth

> **Key design**: UploadView does NO file uploading — it only stages files and creates the upload. All upload logic lives in DownloadView, reusing the same `uploadPendingFiles()` used when adding files to an existing upload.

### Pending Upload Store (`pendingUploadStore.js`)

In-memory store (same pattern as `tokenStore.js`) to pass files from UploadView → DownloadView across navigation:
- `setPendingFiles(uploadId, files, basicAuth, passphrase)` — stash after `createUpload()` (includes E2EE passphrase if enabled)
- `consumePendingFiles(uploadId)` — retrieve and clear (one-shot)

### Staged upload flow (DownloadView)

When adding files to an existing upload:
1. `onFilesSelected` stages files in `pendingFiles` ref (NOT uploaded yet)
2. User sees staged files with remove buttons, can review before uploading
3. Clicking "Upload" runs `uploadPendingFiles()` which uploads files concurrently (max 5) with local status updates
4. Files transition from bottom panel → top panel as they become ready
5. Files added to existing uploads have **no pre-created fileId** — server assigns one

### Upload Cancellation

The `uploadFile()` function in `api.js` returns `{ promise, abort }`:
- `promise` — resolves to file metadata on success
- `abort()` — calls `xhr.abort()`, rejecting with `{ cancelled: true }`

Cancel buttons in `FileRow.vue` emit a `cancel` event for individual files.
A "Cancel All" button in the pending files header aborts all in-progress uploads.

> **Gotcha**: When a file upload is aborted via `xhr.abort()`, the server needs time to detect the broken connection and clean up the file status (`uploading` → `removed` → `deleted`). The `cancelFileUpload()` function waits 200ms before calling `fetchUpload()` to avoid showing stale `uploading` status in `activeFiles`.

---

## Staged File Object Shape

Files stored locally before upload use this shape (NOT the server shape):

```js
{
  reference: 'ref-1707123456-1',  // Local unique ID (from generateRef())
  fileName: 'photo.jpg',
  size: 1048576,
  file: File,                      // The browser File object
  status: 'toUpload',             // 'toUpload' | 'uploading' | 'uploaded' | 'error'
  progress: 0,                    // 0-100 upload progress
  abort: null,                    // Set during upload — calls xhr.abort()
}
```

> **Gotcha**: Local files use `reference` as a key, not `id`. The `id` is only assigned by the server after upload. The `size` field is `size` locally but `fileSize` in server responses.

---

## Filename Length Limit

Filenames are capped at **1024 characters** — enforced client-side at multiple points:

| Location | Enforcement |
|----------|-------------|
| `UploadView.addFiles()` | Truncates `file.name` to 1024 chars when files are added to the staging list |
| `FileRow.onNameInput()` | Truncates on blur (when editing finishes) |
| `FileRow.onNameKeydown()` | Blocks character input at limit (allows Backspace/Delete/ctrl keys) |
| `FileRow.onNamePaste()` | Intercepts paste, calculates available space, clamps inserted text |

> **Note**: The server also validates filename length and returns a 400 if exceeded. The client-side enforcement prevents this from happening under normal use.

---

## Feature Flags (Config)

The server exposes feature flags via `GET /config`:

| Value       | Meaning                                     |
|-------------|---------------------------------------------|
| `enabled`   | Feature is available, default off            |
| `disabled`  | Feature is hidden entirely                   |
| `forced`    | Feature is on, user cannot toggle it off     |
| `default`   | Feature is available, default on             |

| `feature_e2ee` | `"enabled"` or `"disabled"` — controls E2EE toggle in upload sidebar |

The config object keys use the pattern `feature_<name>` (e.g., `feature_one_shot`, `feature_stream`).

Helper functions (in `config.js`):
- `isFeatureEnabled(name)` → returns `true` unless value is `"disabled"`
- `isFeatureForced(name)` → returns `true` only if value is `"forced"`
- `isFeatureDefaultOn(name)` → returns `true` if value is `"default"` or `"forced"` (controls initial toggle state)

### Other Config Keys

The `GET /config` response also includes:

| Key | Purpose |
|-----|---------||
| `maxFileSize` | Max file size in bytes (shown in upload drop zone) |
| `maxUserSize` | Max total size per user |
| `maxTTL` | Max TTL in seconds |
| `googleAuthentication` | `true` if Google OAuth is configured → shows Google login button |
| `githubAuthentication` | `true` if GitHub OAuth is configured → shows GitHub login button |
| `ovhAuthentication` | `true` if OVH OAuth is configured → shows OVH login button |
| `feature_local_login` | `"enabled"` or `"disabled"` — controls local login form visibility (replaces old `localAuthentication` boolean) |
| `oidcAuthentication` | `true` if OIDC is configured → shows OIDC login button |
| `oidcProviderName` | Display name for OIDC button (e.g. `"Keycloak"`, defaults to `"OpenID"`) |
| `downloadDomain` | Raw configured `DownloadDomain` — kept for backward compatibility |
| `downloadURL` | Fully-qualified base URL for download links: `DownloadDomain + Path` (set in `api.js` via `setDownloadURL`). Falls back to `downloadDomain` when not present (pre-1.4 servers). Use this to construct file/archive links |
| `abuseContact` | Abuse contact email → displayed in global footer (`App.vue`) |

---

## Webapp Settings (`settings.js`)

The webapp loads instance-level settings from `/settings.json` at startup (JSONC — `//` comments are stripped before parsing). This is separate from the server `/config` endpoint and lives in `webapp/public/settings.json`.

| Field | Type | Default | Purpose |
|-------|------|---------|--------|
| `name` | string | `"Plik"` | Logo text and page title |
| `logo` | string | `""` | Logo image path (replaces text when set) |
| `theme` | string | `"auto"` | `"dark"`, `"light"`, `"auto"` (OS preference), or any custom theme name matching a CSS file in `themes/` |
| `backgroundImage` | string | `""` | Background image path |
| `backgroundColor` | string | `""` | Fallback background color |
| `overlayOpacity` | number | `0.2` | Dark overlay over background |
| `customCSS` | string | `""` | Path to custom CSS (injected if non-empty) |
| `customJS` | string | `""` | Path to custom JS (injected if non-empty) |
| `themes` | array | `["*"]` | Available themes in the picker (`["*"]` = all built-ins, `[]` = no picker). Entries can be strings (`"nord"`), objects (`{ "name": "custom", "label": "My Theme" }`), or `"*"` to expand all built-ins (e.g. `["*", { "name": "acme", "label": "Acme" }]`) |
| `defaultDarkTheme` | string | `"dark"` | Theme used by "auto" when OS prefers dark mode |
| `defaultLightTheme` | string | `"light"` | Theme used by "auto" when OS prefers light mode |
| `language` | string | `"auto"` | `"auto"` (detect from browser), `"en"`, `"fr"`, or any language code matching a registered locale |
| `languages` | array | `["*"]` | Available languages in the picker (`["*"]` = all built-ins, `[]` = no picker). Entries can be strings (`"fr"`), objects (`{ "name": "de", "label": "Deutsch" }`), or `"*"` to expand all built-ins |
| `footer` | string | `""` | Custom footer HTML (e.g. `"Powered by <a href='…'>Plik</a>"`). Takes precedence over `AbuseContact` in `plikd.cfg`. |

**Footer priority**: `settings.footer` > `config.abuseContact` (`plikd.cfg`) > none. When only `AbuseContact` is set, the footer renders a default "For abuse contact &lt;mailto&gt;" template.

**Streaming upload UX**: The download view shows a "Streaming Upload" info banner (`v-if="upload.stream"`) with an optional timeout notice derived from `config.streamTimeout` (seconds). Cancel for streaming uploads explicitly calls `apiRemoveFile()` after aborting the XHR, because the server goroutine stays blocked in `io.Copy` waiting for a downloader and won't clean up on its own. On any error (timeout, network drop), the server resets the file to `missing` so the existing Retry button works — the `uploadFileEntry` catch block calls `fetchUpload()` and removes non-retryable files from the pending list. The file-uploaded counter (`X/Y files uploaded`) is hidden for streaming uploads since files aren't truly "uploaded" in the traditional sense.

**White-label safety**: The JS defaults are all empty (name = `''`). Only the shipped `settings.json` provides `"Plik"`. If the file is missing or fails to load, no branding leaks.

**Custom asset injection**: `loadSettings()` conditionally injects `<link>` and `<script>` tags if `customCSS`/`customJS` paths are set. Injection happens before Vue mounts (inside the `Promise.all` in `main.js`), so there's no flash of unstyled content.

**Theme system**: Themes are standalone CSS files in `webapp/public/themes/` that override the design tokens defined in `style.css`'s `@theme` block. The built-in `dark` theme is compiled into `style.css` (zero HTTP cost). All other themes (including `light`) are lazy-loaded from `/themes/{name}.css` before `data-theme` is set. A `loadedThemes` Set prevents duplicate `<link>` injection on OS theme toggle. Flash prevention uses inline `<style>` in `index.html` to hide the page with `visibility: hidden` + `background: transparent !important` until the theme is resolved. The "auto" theme resolves to `settings.defaultDarkTheme` / `settings.defaultLightTheme` (defaulting to `dark`/`light`), allowing deployments to customize which themes "auto" uses (e.g. Solarized pair).

Built-in themes: `solarized-dark`, `solarized-light`, `nord`, `nord-light`, `catppuccin-mocha`, `catppuccin-latte`, `matrix`, `hexless`. Dark themes may use outlined buttons (transparent bg + colored border with 40% opacity, brightening to 60% on hover) — see `TEMPLATE.css` for the pattern. Custom themes can be created by copying `themes/TEMPLATE.css`.

**Dropdown pickers** (`DropdownPicker.vue`): Generic shared dropdown component used by both `ThemePicker.vue` and `LanguagePicker.vue`. Handles open/close state, click-outside dismissal, scrollable option list (`max-h-80 overflow-y-auto`), checkmark for active item, optional flag images, and dropdown transition animation. Accepts props: `id`, `items`, `current`, `itemIdPrefix`, `buttonClass`, `title`, `dropdownWidth`. Provides `#icon` and default slots for each thin wrapper to supply its own icon and label text.

**Theme picker** (`ThemePicker.vue`): Thin wrapper over `DropdownPicker`. Palette icon dropdown in the header nav bar, with "Theme" text label and dedicated separators. Lists themes from `getAvailableThemes()` (reads `settings.themes` — `["*"]` = all built-ins, `[]` = no picker). The picker is hidden when `themes.length ≤ 1`. Selection writes to localStorage (`plik-theme` key) and calls `applyTheme()` for instant switching. On boot, `loadSettings()` reads localStorage first, falling back to `settings.theme` default. The `autoListener` variable tracks the OS `prefers-color-scheme` listener and properly removes it when switching away from "auto" mode. **Server-side persistence**: For authenticated users, the theme is also stored in the `User.Theme` DB field. On login/session restore, `syncThemeFromUser()` applies the server value (server wins over localStorage). On theme change, `setUserTheme()` fires a background `patchMe()` call to persist the choice. Anonymous users use localStorage only.

**Dark theme refinements**: The default dark theme uses semi-transparent button fills (`color-mix` at 85% for primary, 75% for danger) to reduce visual harshness. Body text defaults to `surface-200` (not `surface-100`) for reduced eye strain; `surface-100`/`surface-50` are reserved for headings and hover highlights. CodeEditor uses a single unified theme with CSS custom properties (`--color-surface-*`, `--color-accent-*`) so all themes get correct editor styling automatically — no per-theme CodeMirror overrides needed.

**CSS hook**: Logo `<span>` elements in `AppHeader.vue` have the class `plik-logo-text` for targeting via custom CSS.

---

## Size & TTL Limit Precedence

The server enforces layered limits: **user-specific → server config**. The special values `0` (use default) and `-1` (unlimited) are key.

### Value Semantics

| Value | Meaning |
|-------|---------|
| `> 0` | Explicit limit (bytes for size, seconds for TTL) |
| `0`   | Use server default |
| `-1`  | Unlimited (no limit enforced) |

### Precedence Rules (from `server/context/upload.go`)

**MaxFileSize** (`GetMaxFileSize()`):
```
if user != nil && user.MaxFileSize != 0 → user.MaxFileSize
else → config.MaxFileSize
```

**MaxUserSize** (`GetUserMaxSize()`):
```
if user == nil → unlimited (-1)                    // anonymous = no user quota
if user.MaxUserSize > 0 → user.MaxUserSize          // explicit user limit
if user.MaxUserSize < 0 → unlimited (-1)            // user explicitly unlimited
if user.MaxUserSize == 0 → config.MaxUserSize        // fall back to server default
```

**MaxTTL** (inside `setTTL()`):
```
maxTTL = config.MaxTTL
if user != nil && user.MaxTTL != 0 → maxTTL = user.MaxTTL
if maxTTL > 0 → enforce (reject infinite or over-limit TTL)
if maxTTL <= 0 → no limit enforced
```

### Effective Limit Calculation (Client-Side)

`UploadView.vue` computes effective limits via `auth.user` with fallback to `config`:

```js
const effectiveMaxFileSize = computed(() => {
  const user = auth.user
  if (user && user.maxFileSize !== 0 && user.maxFileSize !== undefined) return user.maxFileSize
  return config.maxFileSize
})
```

The same pattern applies for `effectiveMaxTTL`, which is passed as a prop to `UploadSidebar`.

### Size Unit Convention (SI / 1000-based)

> [!IMPORTANT]
> All size formatting uses **SI units** (1 GB = 1,000,000,000 bytes), matching the server's `go-humanize` library (`humanize.ParseBytes` / `humanize.Bytes`). The `GB` constant in edit modals is `1000³`, not `1024³`.

This means:
- Config `MaxFileSizeStr = "10GB"` → 10,000,000,000 bytes → displays "10.00 GB" everywhere
- Admin enters "1" GB in edit modal → stores 1,000,000,000 bytes → shows "1.00 GB"

> [!CAUTION]
> Never use 1024-based division with "GB" labels. If you need binary units, use "GiB" labels with 1024-based math.

---

## Component Architecture

```
App.vue
├── AppHeader.vue          — top nav bar (Upload, CLI, Source, user/admin links)
├── RootView.vue           — switches between Upload/Download based on query.id
│   ├── UploadView.vue     — file staging, settings, upload execution
│   │   ├── UploadSidebar  — upload settings (one-shot, stream, TTL, E2EE, etc.) with (?) help tooltips
│   │   ├── FileRow        — individual file display
│   │   ├── ErrorBanner    — inline dismissible error banner
│   │   └── CodeEditor     — text paste mode with syntax highlighting
│   └── DownloadView.vue   — file list, admin actions
│       ├── DownloadSidebar — upload info (E2EE badge), share (passphrase + toggle), admin URL, actions
│       ├── FileRow         — file link (preview), caret (details), download/QR/copy/view/remove
│       ├── ErrorState      — full-page error state (not found, network error)
│       ├── ErrorBanner     — inline dismissible error banner
│       ├── CodeEditor      — inline file viewer (read-only)
│       ├── QrCodeDialog    — QR code modal
│       ├── CopyButton      — clipboard copy with feedback
│       └── ConfirmDialog   — confirmation modal
├── LoginView.vue          — local login form + OAuth/OIDC buttons
├── HomeView.vue           — user dashboard (uploads/tokens/account)
│   ├── ErrorBanner        — inline dismissible error banner
│   ├── CopyButton         — clipboard copy for tokens
│   ├── EditUserModal      — shared edit-user modal (quotas, name, email, password)
│   ├── UploadControls     — sort/order/badge filters with active-filters slot
│   └── UploadCard         — shared upload card (files with status badges, tokens, actions)
├── AdminView.vue          — admin panel (stats/users/uploads)
│   ├── ErrorBanner        — inline dismissible error banner
│   ├── EditUserModal      — shared edit-user modal (quotas always shown)
│   ├── UploadControls     — sort/order/badge filters with active-filters slot
│   └── UploadCard         — shared upload card (with user column)
├── ClientsView.vue        — CLI client downloads (from embedded build info)
└── CLIAuthView.vue        — CLI device auth approval (displays code, approves session)
```

---

## Authenticated Pages

### Auth State (`authStore.js`)

Reactive singleton holding `auth.user` (set on login, cleared on logout). Checked by `main.js` on app load via `GET /me`. The header shows user/admin links when `auth.user` is set.

### LoginView (`/#/login`)

- Local login form (username + password → `POST /auth/local/login`) — **hidden** when `isFeatureEnabled('local_login')` returns `false` (i.e. `FeatureLocalLogin = "disabled"` on the server)
- Conditional OAuth buttons (Google, GitHub, OVH) based on `config.googleAuthentication` / `config.githubAuthentication` / `config.ovhAuthentication`
- OIDC button (label from `config.oidcProviderName`) → calls `GET /auth/oidc/login` to get the authorization URL, then `window.location.href` redirects to the OIDC provider
- "or continue with" divider only shown when both local login and at least one OAuth/OIDC provider are enabled
- Redirects to the stored `sessionStorage` destination on success via `consumeRedirect()`, or `/` if none


### HomeView (`/#/home`)

Sidebar + main content layout (same pattern as download view).

**Sidebar**: user avatar, login/provider, name, email, admin badge, stats (uploads/files/size). Buttons: Upload files, Uploads, Tokens, Sign out, Edit account, Delete uploads, Delete account.

**Uploads tab**: paginated user uploads via `GET /me/uploads`. Supports token-based filtering. Each upload shows files, date, size, with clickable token labels.

**Tokens tab**: list/create/revoke tokens via `GET|POST|DELETE /me/token`. Token comment displayed above UUID. Click token to filter uploads by it.

**Edit Account modal**: name, email, password (local only). Admin users additionally see maxFileSize, maxUserSize, maxTTL, admin toggle. Saves via `POST /user/{id}`.

> **Gotcha**: Non-admin users cannot change quota fields or admin status — the server enforces this; the UI hides those fields.

### AdminView (`/#/admin`)

Admin-only page. Redirects non-admins to `/` on mount.

**Sidebar**: server version/build info (release + mint badges), nav buttons (Stats, Uploads, Users), Create User button.

**Stats tab**: server config (maxFileSize, maxUserSize, defaultTTL, maxTTL) + server statistics (users, uploads, files, totalSize, anonymous counts).

**Users tab**: paginated user list via `GET /users`. Each row shows login, provider, name, email, quotas, admin badge. Actions: Impersonate (👤), Edit (opens modal with full quota controls), Delete (with confirmation). Delete disabled for self. Impersonate disabled for self.

**Uploads tab**: paginated all-uploads via `GET /uploads`. Sort by date/size, order asc/desc. Filter by user/token (clickable links in each row). Each row shows upload ID (link), dates, user, token, files with sizes, Remove button.

**Create User modal**: provider (select), login, password (local only), name, email, quotas (maxFileSize, maxUserSize, maxTTL), admin toggle. Creates via `POST /user`.

**Edit User modal**: same as HomeView edit but with full admin quota controls always visible.

### Impersonation

Allows an admin to "become" another user to browse their uploads, test their quotas, or manage their account. The feature spans four files:

**Flow:**
1. Admin clicks 👤 on a user row in AdminView
2. `authStore.impersonate(user)` stores the target user and calls `api.setImpersonateUser(userId)`
3. `api.js` injects `X-Plik-Impersonate: <userId>` header on **every** subsequent API request
4. Server middleware (`server/middleware/impersonate.go`) detects the header, verifies the caller is an admin, and switches the request context to the impersonated user
5. `GET /me` now returns the impersonated user — `authStore.user` updates accordingly
6. A yellow banner in `AppHeader.vue` shows "⚠️ Impersonating **username**" with a **Stop** button

**State management (`authStore.js`):**
- `auth.originalUser` — preserved real admin identity (never changes during impersonation)
- `auth.impersonatedUser` — the user object being impersonated (null when not impersonating)
- `auth.user` — switches to the impersonated user during impersonation
- `clearImpersonate()` — resets header, restores `auth.user` to `auth.originalUser`

**API layer (`api.js`):**
- `setImpersonateUser(userId)` — sets/clears a module-level `_impersonateUserId`
- `apiCall()` — if `_impersonateUserId` is set, adds `X-Plik-Impersonate` header


### API Endpoints (Auth/Admin)

| Endpoint              | Method | Purpose                        | Auth       |
|-----------------------|--------|--------------------------------|------------|
| `/auth/local/login`   | POST   | Local login                    | —          |
| `/auth/oidc/login`    | GET    | Get OIDC authorization URL     | —          |
| `/auth/oidc/callback` | GET    | OIDC callback (sets session)   | —          |
| `/auth/google/login`  | GET    | Get Google authorization URL   | —          |
| `/auth/github/login`  | GET    | Get GitHub authorization URL   | —          |
| `/auth/ovh/login`     | GET    | Get OVH authorization URL      | —          |
| `/auth/logout`        | GET    | Logout                         | Session    |
| `/me`                 | GET    | Get current user               | Session    |
| `/me`                 | DELETE | Delete account                 | Session    |
| `/me/uploads`         | GET    | User uploads (paginated)       | Session    |
| `/me/uploads`         | DELETE | Delete all user uploads        | Session    |
| `/me/token`           | GET    | List tokens                    | Session    |
| `/me/token`           | POST   | Create token                   | Session    |
| `/me/token/{token}`   | DELETE | Revoke token                   | Session    |
| `/user/{id}`          | POST   | Update user                    | Session    |
| `/stats`              | GET    | Server statistics              | Admin only |
| `/users`              | GET    | List all users (paginated)     | Admin only |
| `/user`               | POST   | Create user                    | Admin only |
| `/user/{id}`          | DELETE | Delete user                    | Admin only |
| `/uploads`            | GET    | All uploads (paginated, filterable) | Admin only |

> **Gotcha**: XSRF token (from `plik-xsrf` cookie) must be sent as `X-XSRFToken` header on all mutating requests (POST, DELETE). This is handled automatically in `apiCall()`.

---

## Responsive Layout

The layout uses a **mobile-first stacking pattern**:

```
Mobile (<768px):     [Sidebar]     (full width, stacked on top)
                     [Main Content] (full width, below)

Desktop (≥768px):   [Sidebar | Main Content]  (side by side)
```

Key classes:
- Containers: `flex flex-col md:flex-row`
- Sidebars: `w-full md:w-72 md:shrink-0`
- Outer wrapper: `overflow-x-hidden` (prevents long URLs from causing horizontal scroll)
- FileRow: inner container uses `flex-wrap`, "Download" text hidden on mobile (`hidden md:inline`)

---

## CSS Custom Utilities

The `style.css` file defines custom utility classes via `@utility` (Tailwind v4 syntax):

| Utility           | Description                                      |
|--------------------|--------------------------------------------------|
| `glass-card`       | Semi-transparent card with backdrop blur          |
| `btn`              | Base button styles                                |
| `btn-primary`      | Accent-colored button (cyan)                      |
| `btn-success`      | Green button                                      |
| `btn-danger`       | Red button                                        |
| `btn-ghost`        | Transparent hover button                          |
| `toggle-switch`    | Toggle switch base                                |
| `toggle-dot`       | Toggle switch dot (animated)                      |
| `input-field`      | Styled text input                                 |
| `sidebar-section`  | Glass-card styled sidebar section (`overflow: visible` for tooltips) |
| `file-row`         | Glass-card styled file row with hover effect      |
| `setting-help`     | Small muted (?) circle icon for setting help      |
| `setting-tooltip`  | Absolute-positioned tooltip bubble (shown on hover/focus) |

> **Note**: The `.setting-help-wrap` CSS class (not a `@utility`) controls tooltip visibility via `:hover` and `:focus-within`.

> **Gotcha**: These are `@utility` blocks, NOT traditional CSS classes or Tailwind `@apply`. They follow Tailwind v4's custom utility syntax and generate single utility classes.


---

## Internationalization (i18n)

### Setup

- **Library**: `vue-i18n` v11, Composition API mode (`legacy: false`, `globalInjection: true`)
- **Config**: `src/i18n.js` — creates the i18n instance, exports `setLocale()` and `getLocale()` helpers
- **Language management**: `src/settings.js` — `BUILTIN_LANGUAGES`, `getAvailableLanguages()`, `getUserLanguage()`, `setUserLanguage()`, `syncLanguageFromUser()`, `resolveAutoLanguage()`, `currentLanguage` ref (mirrors theme pattern)
- **Integration**: i18n registered as a Vue plugin in `main.js` (`app.use(i18n)`). Language resolved in `loadSettings()` before mount (zero flash).
- **Persistence**: `localStorage('plik-locale')` for all users; `User.Language` DB field for authenticated users (synced on login/session restore via `syncLanguageFromUser()` in `authStore.js`)

### Architecture

Language management follows the **exact same pattern as themes**:

1. `settings.json` declares `language` (default) and `languages` (picker list, `["*"]` = all built-in)
2. `settings.js` owns the `BUILTIN_LANGUAGES` registry and all language logic
3. `loadSettings()` reads localStorage → settings.json fallback, calls `applyLanguage()` before mount (resolves "auto" to browser locale, sets `currentLanguage` ref, calls `setLocale()`)
4. `setUserLanguage()` writes localStorage, delegates to `applyLanguage()`, and fire-and-forget PATCHes `/me` for logged-in users
5. `syncLanguageFromUser()` is called by `authStore.js` on login/session restore (server wins over localStorage)
6. `LanguagePicker.vue` (thin wrapper over shared `DropdownPicker.vue`) uses `getAvailableLanguages()` and `currentLanguage` from `settings.js`

### File Structure

| File | Purpose |
|------|---------|
| `src/i18n.js` | vue-i18n instance, `setLocale()`, `getLocale()` |
| `src/settings.js` | `BUILTIN_LANGUAGES`, `getAvailableLanguages()`, `getUserLanguage()`, `setUserLanguage()`, `syncLanguageFromUser()`, `resolveAutoLanguage()`, `currentLanguage` ref |
| `src/locales/en.json` | English translations (source of truth) |
| `src/locales/*.json` | Translations for de, es, fr, hi, it, nl, pl, pt, ru, sv, zh (must be key-synced with `en.json`) |
| `src/__tests__/locales.test.js` | Automated key sync test — validates keys, empty values, and placeholder tokens |
| `src/components/DropdownPicker.vue` | Generic shared dropdown (scrollbar, click-outside, transitions, flags) |
| `src/components/LanguagePicker.vue` | Thin wrapper over `DropdownPicker`, supplies globe icon + language data |
| `e2e/language-picker.spec.js` | Language picker e2e tests (visibility, dropdown, localStorage, wildcards, flags) |

### Translation Conventions

1. **Template strings**: Use `$t('namespace.key')` or <code v-pre>{{ $t('namespace.key') }}</code> in templates
2. **Script strings**: Destructure `const { t: $t } = useI18n()` and call `$t('...')`
3. **Parameterized**: `$t('key', { name: value })` with `{name}` placeholders in JSON
4. **Component interpolation**: Use `<i18n-t keypath="..." tag="p">` for strings with embedded HTML/components
5. **Utility functions**: Functions in `utils.js` (`quotaLabel`, `ttlLabel`, `defaultSizeHint`, `defaultTTLHint`) accept an optional `t` function as second argument for translation

### Key Namespaces

Keys are grouped by component: `common.*`, `header.*`, `uploadSidebar.*`, `downloadSidebar.*`, `fileRow.*`, `badges.*`, `uploadView.*`, `downloadView.*`, `homeView.*`, `adminView.*`, `loginView.*`, `clientsView.*`, `cliAuth.*`, `errorView.*`, `editUser.*`, `uploadCard.*`, `uploadControls.*`, `api.*`, `languagePicker.*`.

### Adding a New Locale

1. Copy `src/locales/en.json` → `src/locales/<code>.json` and translate all values
2. Add the language to `BUILTIN_LANGUAGES` in `src/settings.js` (with name, label, and flag SVG)
3. Import the locale file in `src/i18n.js` and add to the `messages` object

### Gotchas

- **All locale files must have identical keys to en.json** — verified automatically by `src/__tests__/locales.test.js` (run with `npm test`). The test also checks for empty values and placeholder token mismatches.
- **Flag emojis don't render on Linux** — flags are SVG files in `webapp/public/flags/` (same pattern as themes in `themes/`)
- **TTL_UNITS** have both `label` (English fallback) and `i18nKey` (for `$t()` in templates)
- **`formatDate()`** uses `toLocaleDateString(undefined, ...)` which auto-localizes via the browser locale

### Known Limitations

- **Server-side errors are English-only**: The Go backend returns error messages in English (e.g. `"Invalid credentials"`, `"Upload not found"`). These propagate to the UI as-is. Only client-side error messages (network errors, fallback text) are translated via the `api.*` i18n keys. Server-side i18n would require a significant backend refactor and is out of scope for now.

---

## Code Editor & File Viewer

### CodeEditor Component

Reusable CodeMirror 6 wrapper (`CodeEditor.vue`) used in two contexts:

| Context | View | Mode | Purpose |
|---------|------|------|---------|
| Text paste | UploadView | Read-write | Paste/edit text before uploading as a file |
| File viewer | DownloadView | Read-only | Preview uploaded text files inline |

**Props**: `modelValue` (v-model), `filename` (drives syntax highlighting), `readonly`, `placeholder`

**Language switching**: Uses a `Compartment` to reconfigure the language extension dynamically when `filename` changes — no editor destruction/recreation needed, preserving cursor position and undo history.

**Content-based language detection**: Uses `highlight.js` (lazy-loaded via dynamic `import()` on first detection call) for accurate auto-detection of ~190 languages. Detection fires via a 1s debounce on content changes. In UploadView, auto-detection only updates the filename when it still matches the default `paste.*` pattern.

**JSON prettify / validate**: When the detected language is JSON, two action buttons appear in the editor header bar. **Validate** (`JSON.parse()` only) checks syntax and shows a brief green "Valid" flash on success or a dismissable red error banner on failure — it never changes the content. **Prettify** (`JSON.parse()` → `JSON.stringify(…, null, 2)`) validates *and* reformats the content with 2-space indentation. In read-only mode (DownloadView file viewer) prettify updates the displayed view only — it does not modify the file on the server.

**Auto-display**: In `DownloadView.vue`, if an upload contains exactly one text file, the viewer panel opens automatically on mount (or when the file finishes uploading). A watcher on `activeFiles` triggers `viewFile()` for the first file if it's the only one and it's a text file. **Exception**: auto-display is disabled for one-shot and streaming uploads — one-shot viewing would consume the single download, and streaming files may not be fully stored on the server.

### Text-File Detection

The `isTextFile()` utility in `utils.js` determines if a file can be viewed in the code editor based on:
1. **Size**: Max 5 MB (`MAX_VIEWABLE_SIZE`)
2. **MIME type**: `text/*` prefix only — the server detects MIME types via Go's `http.DetectContentType`, which returns `text/plain` for all text-like content (JS, JSON, Go, Python, etc.) and `application/octet-stream` for binary

`FileRow.vue` uses this to conditionally show a "View" button on uploaded files in download mode. The View button is also hidden for one-shot (`isOneShot` prop) and streaming (`isStream` prop) uploads.

### Markdown File Preview

When viewing or editing a Markdown file (`.md` or `.markdown` extension), **Code / Preview** tabs appear. All three usages share the `MarkdownTabs.vue` component:

| Context | View | Tab labels | Trigger |
|---------|------|-----------|---------|
| Comment editor | UploadView | Write / Preview | Always shown when comments enabled |
| Text paste editor | UploadView | Code / Preview | `isMarkdownFile({ fileName, fileType: 'text/plain' })` |
| File viewer | DownloadView | Code / Preview | `isMarkdownFile(file)` — checks filename + MIME from server |

**`MarkdownTabs.vue`** — Reusable component that renders the tab bar, the HTML preview panel (with `.prose` styling), and a default slot for the editor content. Props: `modelValue` (active tab), `leftLabel`/`leftIcon` (Code vs Write), `renderedHtml`. Named slot `left-badge` for extras like "required".

**`isMarkdownFile(file)`** — Utility in `utils.js` checking filename extension AND `text/*` MIME type.

Default tab for markdown files in the download viewer is **Preview**; in the paste editor it stays on **Code**.

### Mermaid Diagram Rendering

Fenced code blocks with language `mermaid` are rendered as interactive SVG diagrams in all Markdown preview contexts (upload comments, text-paste preview, file viewer).

**Architecture**: Mermaid rendering is a two-phase process:
1. **Parse-time** (`markdown.js`): A custom `marked` renderer detects ` ```mermaid ` blocks and outputs `<div class="mermaid">…</div>` sentinel containers instead of `<pre><code>`. These containers pass through DOMPurify unchanged.
2. **Render-time** (`initMermaidInElement()`): After Vue injects the HTML via `v-html`, the `mermaid` library is lazy-loaded via dynamic `import()` (~2 MB, loaded only on first diagram display) and `mermaid.run()` transforms the sentinel divs into SVGs.

**Integration points**:
- `MarkdownTabs.vue` — watchers on `renderedHtml` and `modelValue` (tab switch to preview) call `initMermaidInElement()` after `nextTick`
- `DownloadView.vue` — watcher on `upload.value?.comments` calls `initMermaidInElement()` on the comments container

**Theme reactivity**: Mermaid diagrams automatically re-render when the user switches themes via `ThemePicker`. On first render, `initMermaidInElement()` stashes the original diagram source in a `data-source` attribute (since `mermaid.run()` replaces the text with SVG) and installs a `MutationObserver` on `<html data-theme="…">` (same pattern as `CodeEditor.vue`). When the theme changes, `reRenderAllMermaid()` detects the new `colorScheme`, re-initializes mermaid with the appropriate theme (`'dark'` or `'default'`), restores all processed diagrams from their stashed source, and re-runs `mermaid.run()`.

> **Gotcha**: `mermaid.run()` must be called on DOM nodes, not HTML strings. The sentinel `<div class="mermaid">` must exist in the DOM before calling `run()` — hence the `nextTick()` dance after `v-html` injection.

### Image File Preview

When viewing an image file (`image/*` MIME type), the file viewer renders an `<img>` tag directly from the server URL — no content fetching or text decoding required.

- **`isImageFile(file)`** in `utils.js` checks that the MIME type starts with `image/`
- **`isViewableFile(file)`** combines `isTextFile(file) || isImageFile(file) || isVideoFile(file) || isAudioFile(file)` — used by `FileRow` for the View button and the auto-view watcher
- No file size limit for images (browsers handle large images natively)
- The viewer header shows a landscape-photo icon (instead of the code angle-brackets icon) for image files
- E2E encrypted images are not supported in the inline viewer (same limitation as text viewer)

### Video & Audio Playback

Video (`video/*`) and audio (`audio/*`) files are played inline using native HTML5 `<video>` and `<audio>` elements.

- **`isVideoFile(file)`** / **`isAudioFile(file)`** in `utils.js` check MIME type prefixes
- No file size limit — browsers handle streaming playback natively via range requests
- The `src` attribute is set directly on `<video>`/`<audio>` (NOT via `<source>` children, which causes browsers to make multiple probe requests)
- `preload="metadata"` lets the browser fetch duration/dimensions without downloading the full file
- The viewer header shows a film icon for video and a music-note icon for audio
- The Copy button is hidden for video/audio (content isn't text); a **"Copy link at current time"** button is shown instead
- The `timeupdate` event updates a `mediaCurrentTime` reactive ref used by the "Copy link at current time" button
- On load, if `t=` is in the URL, the media element seeks to that timestamp via `loadedmetadata` event and attempts autoplay (muted, then unmuted)
- E2E encrypted media is not supported in the inline player (same limitation as images)

### Viewer Navigation

When an upload contains multiple viewable files (text, image, video, or audio), the viewer shows prev/next navigation:

- **Arrow buttons** (‹ ›) with a position indicator (`2/5`) appear in the viewer header
- **Keyboard shortcuts**: `ArrowLeft` / `ArrowRight` to navigate, `Escape` to close
- `viewableFiles` computed filters `activeFiles` through `isViewableFile`, excluding one-shot and streaming uploads
- Keyboard handler ignores events when focus is in an input, textarea, or contenteditable element

### URL Deep Linking (`file=` and `t=` query params)

The viewer state is synced bidirectionally with URL query parameters for sharing:

- **`file=<fileId>`**: When a file viewer opens, `syncViewerToUrl()` adds `file=<fileId>` to the URL via `router.replace()`. When the viewer closes, the param is removed. On page load, if `file=` is in the URL, the corresponding file is auto-opened in the viewer.
- **`t=<seconds>`**: On page load, if `t=` is present for a video/audio file, the media element seeks to that time once `loadedmetadata` fires and attempts autoplay. The `t=` param is preserved in the URL while viewing media but is **not** live-updated during playback.
- **"Copy link at current time"** button appears in the viewer header for video/audio files — copies the full URL including `file=` and `t=` at the current playback position.
- Uses `router.replace()` (not `push`) to avoid cluttering browser history. No `path` is specified in `router.replace()` calls so the app works under sub-paths.

> **Gotcha**: The `shareAtTimeUrl` computed property uses a reactive `mediaCurrentTime` ref that's updated in the `timeupdate` handler, since Vue cannot observe native DOM property changes on `<video>`/`<audio>` elements directly.

---

## Testing

The webapp uses [Vitest](https://vitest.dev/) with jsdom for unit testing.

```bash
npm test                    # Run all tests (vitest run)
make test-frontend          # Same, via Makefile (npm ci + npm test)
```

Tests live in `src/__tests__/` and cover pure utility functions, config helpers, and stores:

| File | Scope |
|------|-------|
| `utils.test.js` | All pure functions in `utils.js` (formatting, conversion, round-trips) |
| `config.test.js` | Feature flag helpers (`isFeatureEnabled`, `isFeatureForced`, `isFeatureDefaultOn`) |
| `markdown.test.js` | Markdown rendering + XSS sanitization via DOMPurify |
| `pendingUploadStore.test.js` | One-shot store semantics (set, consume, double-consume) |


Vitest configuration is in `vite.config.js` under the `test` key (`globals: true`, `environment: 'jsdom'`).

### E2E Testing (Playwright)

End-to-end tests use [Playwright](https://playwright.dev/) to drive a real Chromium browser against a running `plikd` instance.

```bash
make test-frontend-e2e          # Full self-contained run (builds server+frontend, starts fresh plikd)
cd webapp && npx playwright test           # Quick run (assumes plikd is already running)
cd webapp && npx playwright test --ui      # Interactive UI mode
```

Tests live in `webapp/e2e/` and cover core flows:

| File | Scope |
|------|-------|
| `settings.spec.js` | Feature flags, TTL, toggles, abuse contact, header links |
| `upload.spec.js` | File upload via input, multi-file, text paste |
| `admin.spec.js` | Server info, config, stats, version badges |
| `download.spec.js` | Download page, text viewer, paste upload |
| `navigation.spec.js` | Routing, auth redirects, OAuth |
| `e2ee.spec.js` | End-to-end encryption flows |
| `password.spec.js` | Password protection |
| `home.spec.js` | User info, config, stats panels |
| `qrcode.spec.js` | QR code modal |
| `retry.spec.js` | Upload failure/retry, cancel |
| `streaming.spec.js` | Stream upload, URL path, hidden actions |
| `customization.spec.js` | Runtime settings.json override, custom CSS/JS injection, white-label fallback |
| `mermaid.spec.js` | Mermaid diagram rendering, source stashing, comment SVG, theme reactivity |
| `subpath.spec.js` | Subpath deployment (`Path=/sub`): asset loading, settings.json URL, theme/flag paths, upload/download, API URL scoping |
| `language-picker.spec.js` | Language picker visibility, dropdown, localStorage, wildcards, flags |

**Server lifecycle**: Playwright's `webServer` launches two `plikd` instances: `e2e/start-server.sh` (root path, port 8585) and `e2e/start-server-subpath.sh` (Path="/sub", port 8586). Each creates a fresh temp directory with clean SQLite DB + data backend, seeds an admin user, and starts `plikd`. Two Playwright projects target these: `chromium` (all specs except `subpath.spec.js`) and `chromium-subpath` (only `subpath.spec.js`, with `baseURL: 'http://localhost:8586/sub/'`). The `globalTeardown` cleans up both temp dirs.

> **Gotcha**: In the subpath project, `page.goto('./')` must be used instead of `page.goto('/')`. Playwright resolves `/` as an absolute path from the origin (`http://localhost:8586/`), ignoring the subpath in `baseURL`. The `'./'` form stays relative to the base.

**Fixtures** (`e2e/fixtures.js`): `authenticatedPage` provides a pre-logged-in admin session; `withConfig(overrides)` intercepts `/config` API to test feature flags; `withVersion(overrides)` intercepts `/version` API for badge testing; `uploadTestFile()` creates a quick upload through the UI. Note: the `authenticatedPage` fixture uses `fetch('/auth/local/login')` with an absolute path — it works for the root-path project but not for `chromium-subpath`. The subpath spec has its own `loginAs()` helper that derives the API base from `window.location.pathname`, mirroring how `api.js` works in production.

---

## Build & Release Process

### Development

```bash
cd webapp && npm install && npm run dev    # Vite dev server on :5173, proxies API to :8080
cd server && go run . --config ./plikd.cfg # Go backend on :8080
```

Vite proxy is configured in `vite.config.js` — all `/api`, `/auth`, `/file`, `/stream`, `/config`, `/me`, etc. calls are forwarded to the Go backend.

### Production Build

```bash
make frontend   # cd webapp && npm ci && npm run build → webapp/dist/
make server     # cd server && go build → server/plikd
```

The Go server serves `webapp/dist/` via `http.FileServer`. Default config: `WebappDirectory = "../webapp/dist"`.

### Makefile Targets

| Target            | Purpose                                           |
|-------------------|---------------------------------------------------|
| `all`             | `clean clean-frontend frontend clients server`    |
| `frontend`        | `npm ci && npm run build` in `webapp/`            |
| `server`          | Build Go binary `server/plikd`                    |
| `client`          | Build Go CLI client `client/plik`                 |
| `clients`         | Cross-compile clients for all architectures       |
| `docker`          | Build Docker image `rootgg/plik:dev`              |
| `release`         | Create release archives via `releaser/release.sh` |
| `test-frontend`   | `npm ci && npm test` — run vitest unit tests      |
| `clean`           | Remove server/client binaries                     |
| `clean-frontend`  | Remove `webapp/dist/`                             |
| `clean-all`       | Clean everything including `node_modules`         |

### Build Info & Client Downloads

The server binary embeds a JSON blob (via `server/gen_build_info.sh`) containing a client list discovered from the `clients/` directory. The `ClientsView` page displays download links from this embedded build info.

For full details on the Docker multi-stage build and release packaging, see [releaser/ARCHITECTURE.md](../releaser/ARCHITECTURE.md).

---

## Common Pitfalls

1. **Don't call `resp.json()` then `resp.text()`** — the body stream can only be read once. Always read as text first.

2. **File IDs are server-assigned** — when adding files to existing uploads, don't pass a `fileId` in the URL. The server creates one.

3. **`uploadToken` must be in `X-UploadToken` header** — not in the request body or URL query for API calls.

4. **During uploads, `activeFiles` filters by readiness** — non-streaming: only `uploaded`; streaming: `uploading` + `uploaded`. When not uploading (friend viewing), all files are shown including removed/deleted (greyed-out with "Removed" badge). The `totalFiles` computed excludes removed/deleted for counting purposes.

5. **Refreshing the page loses admin access** — tokens are in-memory only. The only way to regain access is to open the Admin URL again.

6. **Delete responses are plain text `"ok"`** — don't try to parse `.message` from them. Always `fetchUpload()` after mutations.

7. **One-shot files disappear after download** — their status changes server-side; re-fetching will show them as removed/missing.

8. **The Admin URL sidebar truncation uses `overflow-hidden` + `min-w-0`** — without this, long URLs push the entire mobile layout wider than the viewport.

9. **`generateRef()` is for local tracking only** — it creates monotonically increasing IDs that are never sent to the server.

10. **Vite dev server runs on port 5173/5174** — the Go backend runs on port 8080. During dev, Vite proxies API calls to the backend via `vite.config.js`.

11. **`webapp/dist/` is gitignored** — never commit build artifacts. The CI/Docker build produces them fresh.

12. **DownloadView has two error refs** — `error` (page-level, rendered via `ErrorState`) and `uploadError` (inline `ErrorBanner`). Setting file upload errors on `error` hides the entire upload content due to template branching. Always use `uploadError` for file transfer failures. HomeView and AdminView use a single `error` ref rendered via `ErrorBanner` at the top of `<main>`.

13. **Filenames are capped at 1024 characters** — enforced in `UploadView.addFiles()`, `FileRow.onNameInput/onNameKeydown/onNamePaste`. The server also validates this, so both layers must agree.

14. **E2EE passphrase is never stored server-side** — it lives only in the `pendingUploadStore` (for same-session navigation) and optionally in the URL fragment (via the share toggle). If the user loses the passphrase, decryption is impossible.

---

## Markdown Rendering

### Module: `markdown.js`

Shared utility for rendering Markdown comments to sanitized HTML:

```javascript
import { renderMarkdown } from '../markdown.js'
```

| Function | Description |
|----------|-------------|
| `renderMarkdown(text)` | Parses Markdown via `marked`, sanitizes HTML via `DOMPurify` |

Used by both `UploadView` (comment preview) and `DownloadView` (comment display) via `v-html`. DOMPurify prevents stored XSS from user-supplied Markdown comments that could contain malicious HTML/JS.

> **Rule**: Never use `marked.parse()` directly with `v-html`. Always use `renderMarkdown()` which applies DOMPurify sanitization.

---

## End-to-End Encryption (E2EE)

### Module: `crypto.js`

Provides streaming encryption/decryption using the `age-encryption` npm package:

| Function | Description |
|----------|-------------|
| `encryptFile(file, passphrase)` | Encrypts a `File` object → returns encrypted `File` |
| `fetchAndDecrypt(url, passphrase)` | Fetches encrypted bytes, decrypts → returns `Blob` |
| `generatePassphrase()` | Generates a 32-char cryptographically-secure passphrase |

### Upload Flow (E2EE)

1. User toggles E2EE in `UploadSidebar` → passphrase auto-generated (or customized)
2. **Validation**: Both `doUpload()` and `createEmptyUpload()` reject the upload with an error if E2EE is enabled but the passphrase is empty (prevents unencrypted files from being marked as encrypted). `UploadSidebar` also shows a red warning ring and "Passphrase cannot be empty" message on the input field.
3. `UploadView.doUpload()` encrypts each file via `encryptFile()` before building the upload params
4. `params.e2ee = 'age'` sent to server → server stores the E2EE scheme on the upload model
5. Passphrase passed via `setPendingFiles(id, files, basicAuth, passphrase)` to the pending store
6. Navigation to DownloadView — passphrase is **not** in the URL

### Download Flow (E2EE)

1. `DownloadView.onMounted()` reads passphrase from `pendingUploadStore` (same-session) or URL fragment `#key=` (shared link)
2. If E2EE is set on the upload but no passphrase is available → a non-dismissable passphrase modal appears (no Cancel button, overlay click blocked). The modal can only be closed by entering a valid passphrase and clicking Decrypt.
3. Passphrase is stripped from the URL after extraction (security measure)
4. `decryptAndDownload()` fetches the encrypted file and decrypts in-browser via `fetchAndDecrypt()`
5. For E2EE files, `FileRow` emits `decrypt-download` instead of using a direct download link

### Server Behavior for E2EE Uploads

- **Browser redirect**: `GetFile` handler checks `common.IsPlikWebapp(req)` (via `X-ClientApp: web_client` header) — if the request is from the webapp and the upload has `E2EE != ""`, it redirects to `/#/?id=<uploadId>` so the webapp handles passphrase input and decryption
- **Content-Type**: E2EE uploads are always served as `application/octet-stream` — content-type detection on encrypted bytes is meaningless
- **CLI downloads**: Non-webapp requests get raw encrypted bytes directly (for piping to `age --decrypt`)

### DownloadSidebar (E2EE)

- **🔐 Encrypted badge**: Shown in upload info when `upload.e2ee` is truthy — displays "End-to-End Encrypted with Age" where Age is a link to [age-encryption.org](https://age-encryption.org)
- **Passphrase display**: Read-only display in Share section with edit (pencil) button and copy button, always shown for E2EE uploads. Edit button opens the passphrase modal to change the passphrase (overlay dismiss is allowed when editing since a passphrase already exists)
- **Include passphrase in link toggle**: Off by default — appends `#key=<passphrase>` to the share URL when enabled

