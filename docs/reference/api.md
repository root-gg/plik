# HTTP API

Full REST API reference. All endpoints accept/return JSON unless noted.

## Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/config` | Server configuration (feature flags, limits, `downloadURL`) |
| `GET` | `/version` | Build info — public, richer for admin sessions (see below) |
| `GET` | `/qrcode?url=...&size=...` | Generate QR code PNG |
| `GET` | `/health` | Health check |

## Upload & File Endpoints

Authentication: session cookie or `X-PlikToken` header.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/` | Quick upload: create upload + add file |
| `POST` | `/upload` | Create upload with options |
| `GET` | `/upload/{uploadID}` | Get upload metadata |
| `DELETE` | `/upload/{uploadID}` | Delete upload |
| `POST` | `/file/{uploadID}` | Add file (multipart) |
| `POST` | `/file/{uploadID}/{fileID}/{filename}` | Add file with known ID (stream mode) |
| `DELETE` | `/file/{uploadID}/{fileID}/{filename}` | Remove file |
| `GET` | `/file/{uploadID}/{fileID}/{filename}` | Download file |
| `HEAD` | `/file/{uploadID}/{fileID}/{filename}` | File metadata |
| `POST` | `/stream/{uploadID}/{fileID}/{filename}` | Stream upload |
| `DELETE` | `/stream/{uploadID}/{fileID}/{filename}` | Cancel stream upload |
| `GET`, `HEAD` | `/stream/{uploadID}/{fileID}/{filename}` | Stream download |
| `GET`, `HEAD` | `/archive/{uploadID}/{filename}` | Download all files as zip |

### GET /version

Public endpoint. All callers receive `version`, `clients`, and `releases`. Admin sessions receive the full build details:

| Field | Public | Admin only |
|-------|--------|------------|
| `version` | ✅ | |
| `clients` | ✅ | |
| `releases` | ✅ | |
| `date` | | ✅ |
| `user` | | ✅ |
| `host` | | ✅ |
| `gitShortRevision` | | ✅ |
| `gitFullRevision` | | ✅ |
| `goVersion` | | ✅ |
| `isRelease` | | ✅ |
| `isMint` | | ✅ |

---

### Create Upload (POST /upload)

```json
{
    "ttl": 86400,
    "extend_ttl": false,
    "oneShot": false,
    "removable": true,
    "stream": false,
    "login": "foo",
    "password": "bar",
    "comments": "optional markdown",
    "e2ee": "age"
}
```

Response:

```json
{
    "id": "TczL35OTIb3InNr6",
    "uploadToken": "50lGHbLEIrpJOl4uECddTI7pga...",
    "downloadDomain": "https://dl.example.com",
    "downloadURL": "https://dl.example.com/sub",
    "files": []
}
```

`downloadDomain` — raw domain configured as `DownloadDomain`, kept for backward compatibility.
`downloadURL` — fully-qualified base URL for file/archive links. Present when `PlikDomain` or `DownloadDomain` is configured. Uses `DownloadDomain + Path` when set, otherwise `PlikDomain + Path`. Absent when neither domain is configured — clients should fall back to the URL they used to reach the server.

### Add File (POST /file/{uploadID})

Send as `multipart/form-data` with `file` field. The `X-UploadToken` header is required (returned from upload creation).

### Download File

The upload token is not required for public uploads. For password-protected uploads, provide HTTP Basic auth with the upload's login/password.

HTTP Range requests (`Range` header) are supported on file downloads, allowing partial content retrieval (206 responses).

### Upload & File Objects

Upload and file objects both expose `downloadCount` (int) and `lastDownloadedAt` (timestamp, omitted if never downloaded). The upload object additionally exposes `downloadedBytes` (int): the lifetime downloaded data (bytes served) for the upload's files, accumulated on every response that streams file bytes — including mid-range GETs that are not download events — exactly like `downloadCount` mirrors download events (there is no per-file equivalent). All three fields are visible only to an upload **admin**, as reflected by the upload's `admin` field. A requester is an upload admin when it presents the upload's per-upload `uploadToken` (`X-UploadToken` header), when it is authenticated with the API token the upload was created with (`X-PlikToken`), or when it holds a session cookie as either the upload's owner or a server administrator. API-token-authenticated requests are matched on the exact creating token only — they never receive owner or server-administrator visibility, even if the token belongs to the owner or to an admin user. Non-admin callers receive `downloadCount: 0`, `downloadedBytes: 0`, and no `lastDownloadedAt` on the upload; files (which have no `downloadedBytes` field) receive `downloadCount: 0` and no `lastDownloadedAt`.

### GET /config — Selected Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `downloadDomain` | `string` | Raw configured `DownloadDomain` (backward compat) |
| `downloadURL` | `string` | Base URL for file/archive links. Present when `PlikDomain` or `DownloadDomain` is configured (`DownloadDomain + Path`, or `PlikDomain + Path`). Absent otherwise |
| `plikDomain` | `string` | Configured `PlikDomain` (public server URL, no path) |
| `maxFileSize` | `int` | Max file size in bytes (`-1` = unlimited) |
| `feature_*` | `string` | Feature flag values: `disabled`, `enabled`, `default`, `forced` |

## Authentication Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/auth/google/login` | Get Google consent URL |
| `GET` | `/auth/google/callback` | Google OAuth callback |
| `GET` | `/auth/github/login` | Get GitHub consent URL |
| `GET` | `/auth/github/callback` | GitHub OAuth callback |
| `GET` | `/auth/ovh/login` | Get OVH consent URL |
| `GET` | `/auth/ovh/callback` | OVH OAuth callback |
| `GET` | `/auth/oidc/login` | Get OIDC consent URL |
| `GET` | `/auth/oidc/callback` | OIDC callback |
| `POST` | `/auth/local/login` | Login `{ "login": "...", "password": "..." }` |
| `POST` | `/auth/cli/init` | Start CLI auth session `{ "hostname": "..." }` |
| `POST` | `/auth/cli/approve` | Approve CLI session `{ "code": "...", "comment": "..." }` |
| `POST` | `/auth/cli/poll` | Poll CLI session `{ "code": "...", "secret": "..." }` |
| `GET` | `/auth/logout` | Logout |

## User Endpoints

Requires authenticated session cookie.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/me` | Current user info |
| `PATCH` | `/me` | Update self-editable profile fields (`name`, `email`, `theme`, `language`) |
| `DELETE` | `/me` | Delete own account |
| `GET` | `/me/token` | List tokens (paginated, sortable by `date`, `size`, `lifetimeSize`) |
| `POST` | `/me/token` | Create upload token `{ "comment": "..." }` |
| `DELETE` | `/me/token/{token}` | Revoke token |
| `GET` | `/me/uploads` | List uploads (paginated, filterable, sortable by `date`, `size`, `downloads`, `downloadedBytes`) |
| `DELETE` | `/me/uploads` | Remove all uploads |
| `GET` | `/me/stats` | User current/lifetime statistics (nested `usage`) |
| `GET` | `/me/stats?token=...` | Statistics for one API token (token-scoped `usage`) |
| `GET` | `/me/stats/activity/daily?days=N` | Authenticated user's daily activity series (downloads + uploads, counts + bytes) |
| `GET` | `/me/stats/trending/uploads?window=7d&sort=downloads&limit=20` | Authenticated user's own trending uploads (`window`: `all`, `1d`, `7d`, `30d`; `sort`: `downloads`, `downloadedBytes`) |

## Admin Endpoints

Requires admin session cookie.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/user` | Create user |
| `GET` | `/user/{userID}` | Get user info |
| `POST` | `/user/{userID}` | Update user |
| `DELETE` | `/user/{userID}` | Delete user |
| `GET` | `/stats` | Server current/lifetime statistics (nested `usage` + `anonymousUsage`) |
| `GET` | `/stats/activity/daily?days=N` | Server-wide daily activity series (downloads + uploads, counts + bytes) |
| `GET` | `/stats/trending/uploads?window=7d&sort=downloads&limit=20` | Trending uploads (`window`: `all`, `1d`, `7d`, `30d`; `sort`: `downloads`, `downloadedBytes`) |
| `GET` | `/stats/trending/files?window=7d&limit=20` | Trending files (`window`: `all`, `1d`, `7d`, `30d`) |
| `GET` | `/users` | List all users (paginated, filterable, sortable by `date`, `size`, `lifetimeSize`, `downloadedBytes`) |
| `GET` | `/users/search?q=...` | Search users (optional: `provider`, `admin`, `limit`) |
| `GET` | `/uploads` | List all uploads (paginated, filterable, sortable by `date`, `size`, `downloads`, `downloadedBytes`) |

### The `usage` object (canonical stats shape)

Every stats endpoint and attachment (`/stats`, `/me/stats`, per-token `token.stats.usage`, per-user `user.stats.usage`) exposes the **same** nested `usage` object. It is the canonical stats contract; prefer it over the legacy flat fields. All three scopes nest it identically one level deep — `server.usage`, `user.usage`, and `token.stats.usage` all end in `…usage.current` / `…usage.lifetime` — so `token.stats` is itself just a one-field `{ "usage": {...} }` envelope, not the usage object directly.

```json
{
  "startedAt": "2026-05-04T10:00:00Z",
  "lastUploadAt": "2026-05-04T10:30:00Z",
  "downloads": {
    "total": 42,
    "bytes": 1048576,
    "today": 3,
    "last7Days": 10,
    "last30Days": 40
  },
  "uploads": {
    "total": 30,
    "bytes": 2097152,
    "today": 2,
    "last7Days": 6,
    "last30Days": 20
  },
  "current": {
    "uploads": 1,
    "files": 2,
    "totalSize": 42,
    "features": { "passwordUploads": 0, "removableUploads": 0, "oneShotUploads": 0, "streamUploads": 0, "extendTTLUploads": 0, "e2eeUploads": 0, "commentUploads": 0 },
    "ttl": { "noneUploads": 0, "lessThan1HourUploads": 0, "oneHourToOneDayUploads": 0, "oneDayToSevenDaysUploads": 0, "sevenDaysTo30DaysUploads": 0, "greaterThan30DaysUploads": 0 },
    "fileSizes": { "lessThan1MBFiles": 2, "oneMBTo10MBFiles": 0, "tenMBTo100MBFiles": 0, "hundredMBTo1GBFiles": 0, "oneGBTo10GBFiles": 0, "tenGBTo100GBFiles": 0, "greaterThan100GBFiles": 0 }
  },
  "lifetime": { "…identical shape to current…": 0 }
}
```

Field notes:

- `startedAt` — the "stats since" timestamp for the `lifetime` counters (omitted if unset).
- `lastUploadAt` — timestamp of the most recent upload in this scope (omitted if none). This is the **single canonical location** for the last-upload timestamp; for a token it is `token.stats.usage.lastUploadAt` (there is no separate `token.lastUploadAt`).
- `downloads.total` — all-time download **events** (not bytes); `downloads.bytes` — all-time bytes served (downloaded data), tracked from the byte-accounting upgrade forward.
- `uploads` is the symmetric upload counterpart of `downloads`: `uploads.total` — all-time upload count (the lifetime uploads counter); `uploads.bytes` — all-time wire bytes received (uploaded data, including partial/failed transfers), tracked from the upgrade that introduced it forward (backfilled/imported rows start at 0).
- `downloads.today` / `last7Days` / `last30Days` (and the symmetric `uploads.today` / `last7Days` / `last30Days`) — bounded **count** windows over UTC calendar days: `today` is the current UTC day, `last7Days` is today plus the previous 6 UTC days, `last30Days` is today plus the previous 29 UTC days. **These window fields are present only where a daily rollup series exists — server `usage` and user `usage` — and are omitted for token scope, `anonymousUsage`, and the per-user `stats` attachment in admin user lists (which carry `total` + `bytes` only). Byte volumes have no bounded windows in the API (only lifetime `bytes`); the daily-activity series below supplies per-day byte breakdowns.**
- `current` vs `lifetime` — `current` counts retained metadata now; `lifetime` counts everything since `startedAt` and never decreases. Both use the identical `UsageStatsPeriod` shape (`uploads`, `files`, `totalSize`, `features`, `ttl`, `fileSizes`).
- `features` counters are individual upload counters, not combinations. `ttl` buckets count the TTL chosen at upload creation. `fileSizes` buckets count files (not bytes) using binary MiB/GiB thresholds.

### GET /me/stats

Returns the authenticated user's usage. The response keeps the legacy flat current trio `uploads`, `files`, `totalSize` for compatibility and adds the nested `usage` object (with download and upload count windows populated from the user's daily activity series):

```json
{
  "uploads": 1,
  "files": 2,
  "totalSize": 42,
  "usage": { "…the usage object above…": 0 }
}
```

The optional `token` query parameter returns stats for an API token owned by the user; in that case `usage` is token-scoped and omits the download windows (revoked tokens do not retain historical token stats).

### GET /me/token

Each token returned by `GET /me/token` includes a `stats` field wrapping the nested `usage` object in the same envelope as the server and user scopes — `token.stats.usage` (token scope: `downloads.total` + `downloads.bytes`, no windows; `startedAt` is the token creation time; `lastUploadAt` when the token has created uploads):

```json
{
  "stats": {
    "usage": { "…the usage object above, token-scoped (no windows)…": 0 }
  }
}
```

Supported sort fields are `date` (token creation), `size` (current retained size), and `lifetimeSize` (lifetime uploaded size).

### GET /stats

Returns server-wide usage statistics. The response keeps master's legacy flat current fields for compatibility — `users`, `uploads`, `anonymousUploads`, `files`, `totalSize`, `anonymousTotalSize` — plus the top-level lifetime user counter `lifetimeUsers` (user creations since `usage.startedAt`, never decremented). All other counters live in the nested `usage` (server scope, download windows populated) and `anonymousUsage` (anonymous scope, no windows) objects:

```json
{
  "users": 12,
  "uploads": 34,
  "anonymousUploads": 5,
  "files": 78,
  "totalSize": 123456,
  "anonymousTotalSize": 6789,
  "lifetimeUsers": 15,
  "usage": { "…the usage object above, with downloads/uploads today/last7Days/last30Days windows…": 0 },
  "anonymousUsage": { "…the usage object above, downloads/uploads total+bytes only…": 0 }
}
```

### GET /users

Each user returned by `GET /users` (and `/users/search`) includes a `stats` object using the `/me/stats` shape — legacy flat trio `uploads`, `files`, `totalSize` plus the nested `usage` object. The attachment is batched across the page, so its `usage.downloads` and `usage.uploads` carry `total` + `bytes` only (no per-user windows). Supported sort fields are `date` (user creation), `size` (current retained size), `lifetimeSize` (lifetime uploaded size), and `downloadedBytes` (lifetime bytes served, `usage.downloads.bytes`). `downloadedBytes` is a `GET /users`-only sort value — it is **not** in the value list `GET /me/token` accepts (see below).

### GET /stats/activity/daily and GET /me/stats/activity/daily

Return a daily **activity** series for the chart views: `/stats/activity/daily` is admin-only and server-wide; `/me/stats/activity/daily` is the authenticated user's own series. One series feeds all four measures (downloads, downloaded data, uploads, uploaded data), so the bounded windows in the `usage` object and this chart series can never drift apart. (These endpoints supersede the former `/stats/downloads/daily` and `/me/stats/downloads/daily`, which have been removed.)

The `days` query parameter selects the number of trailing UTC days: it defaults to `30`, must be an integer, and must be between `1` and `31` inclusive — anything else (out of range or non-integer) returns `400` with a curated message.

The response is a bare JSON array (never `null`), dense and oldest-first: exactly `days` points, one per UTC day, zero-filled for days with no recorded activity. `day` is the UTC calendar day formatted `YYYY-MM-DD`; `downloads`/`uploads` are the day's event counts and `downloadedBytes`/`uploadedBytes` are the day's byte volumes.

```json
[
  { "day": "2026-05-03", "downloads": 0, "downloadedBytes": 0, "uploads": 0, "uploadedBytes": 0 },
  { "day": "2026-05-04", "downloads": 3, "downloadedBytes": 1024, "uploads": 2, "uploadedBytes": 2048 }
]
```

### Trending Stats

Admin-only `/stats/trending/uploads` and `/stats/trending/files` return current uploads/files ordered by download count by default. `GET /me/stats/trending/uploads` is the self-scoped counterpart: same params, same response shape, restricted to the authenticated user's own uploads — there is no self-scoped trending files endpoint. If omitted, `window` defaults to `all`; accepted values are `all`, `1d`, `7d`, and `30d`, and any other value returns `400`. `limit` defaults to `20`, must be a positive integer, and is capped at `100` (values above 100 are clamped; `0`, negatives, and non-integers return `400`). Window/limit/sort validation happens only in the HTTP handler.

The two trending-**uploads** endpoints also accept `sort`: `downloads` (default) ranks by download count, `downloadedBytes` ranks by bytes served instead; any other value returns `400`. Trending **files** has no `sort` param — it always ranks by download count. Whichever metric is selected, uploads with a zero value for that metric are omitted from the results.

Trending items include IDs, display name/comment, user ID, size or file count, `downloadCount`, `downloadedBytes`, and `lastDownloadedAt`. Both `downloadCount` and `downloadedBytes` are always populated on trending uploads regardless of which `sort` was requested. `downloadedBytes` is tracked from the byte-accounting upgrade forward (like the other byte counters in this document) and stays `0` on trending **files** — there is no lifetime per-file byte column, so file-grain byte trending is not tracked. The response is a bare JSON array (never `null`), empty when nothing has been downloaded yet.

## Pagination

Paginated endpoints use **cursor-based** pagination. Parameters can be passed as query strings or as a JSON object in the `X-Plik-Paging` header.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `limit` | `20` | Max results per page |
| `order` | `desc` | Sort order (`asc`/`desc`) |
| `before` | | Cursor: fetch items before this ID |
| `after` | | Cursor: fetch items after this ID |

Paginated responses use this envelope:

```json
{
    "before": "cursor-id-for-previous-page",
    "after": "cursor-id-for-next-page",
    "total": 142,
    "results": [...]
}
```

Pass the `after` value as the `after` query parameter to fetch the next page. Pass `before` to go backwards. A `null` cursor means there are no more pages in that direction.

## Upload Filters

Upload listing endpoints (`/me/uploads`, `/uploads`) accept these optional query parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `sort` | `string` | Sort order: `date` (creation date, the default when omitted), `size` (total upload size), `downloads` (download count), or `downloadedBytes` (bytes served). Any other value returns `400`. |
| `user` | `string` | Filter by user ID (admin only) |
| `token` | `string` | Filter by upload token (admin only) |
| `oneShot` | `bool` | Filter one-shot uploads |
| `removable` | `bool` | Filter removable uploads |
| `stream` | `bool` | Filter stream uploads |
| `extendTTL` | `bool` | Filter extend-TTL uploads |
| `password` | `bool` | Filter password-protected uploads |
| `e2ee` | `bool` | Filter end-to-end encrypted uploads |

## User Filters

User listing endpoints (`/users`) accept:

| Parameter | Type | Description |
|-----------|------|-------------|
| `sort` | `string` | Sort order: `date` (creation date, the default when omitted), `size` (current retained size), `lifetimeSize` (lifetime uploaded size), or `downloadedBytes` (lifetime bytes served — `GET /users` only). Any other value returns `400`. `GET /me/token` shares the same rule but only the `date`/`size`/`lifetimeSize` value list — `downloadedBytes` returns `400` there. |
| `provider` | `string` | Filter by auth provider (e.g. `google`, `ovh`, `oidc`, `local`) |
| `admin` | `bool` | Filter admin/non-admin users |
