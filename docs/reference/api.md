# HTTP API

Full REST API reference. All endpoints accept/return JSON unless noted.

## Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/config` | Server configuration (feature flags, limits, `downloadURL`) |
| `GET` | `/version` | Build info (version, git commit) |
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
| `GET` | `/stream/{uploadID}/{fileID}/{filename}` | Stream download |
| `GET` | `/archive/{uploadID}/{filename}` | Download all files as zip |

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
`downloadURL` — fully-qualified base URL for download links: `DownloadDomain + Path`. Use this to construct file/archive links. Falls back to the server's own URL when `DownloadDomain` is not set.

### Add File (POST /file/{uploadID})

Send as `multipart/form-data` with `file` field. The `X-UploadToken` header is required (returned from upload creation).

### Download File

The upload token is not required for public uploads. For password-protected uploads, provide HTTP Basic auth with the upload's login/password.

HTTP Range requests (`Range` header) are supported on file downloads, allowing partial content retrieval (206 responses).

### GET /config — Selected Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `downloadDomain` | `string` | Raw configured `DownloadDomain` (backward compat) |
| `downloadURL` | `string` | `DownloadDomain + Path` — use this as the base for file/archive links |
| `plikDomain` | `string` | Configured `PlikDomain` (public server URL, no path) |
| `maxFileSize` | `int` | Max file size in bytes (`-1` = unlimited) |
| `feature_*` | `string` | Feature flag values: `disabled`, `enabled`, `default`, `forced` |

> **Note on path components:** `PlikDomain` and `DownloadDomain` must be domain-only (no path). If a path is inadvertently included, Plik will strip it with a warning on startup. Use the `Path` server config option to configure a URL path prefix — it is automatically incorporated into `downloadURL`.

## Authentication Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/auth/google/login` | Get Google consent URL |
| `GET` | `/auth/google/callback` | Google OAuth callback |
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
| `DELETE` | `/me` | Delete own account |
| `GET` | `/me/token` | List tokens (paginated) |
| `POST` | `/me/token` | Create upload token `{ "comment": "..." }` |
| `DELETE` | `/me/token/{token}` | Revoke token |
| `GET` | `/me/uploads` | List uploads (paginated, filterable) |
| `DELETE` | `/me/uploads` | Remove all uploads |
| `GET` | `/me/stats` | User statistics |

## Admin Endpoints

Requires admin session cookie.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/user` | Create user |
| `GET` | `/user/{userID}` | Get user info |
| `POST` | `/user/{userID}` | Update user |
| `DELETE` | `/user/{userID}` | Delete user |
| `GET` | `/stats` | Server statistics |
| `GET` | `/users` | List all users (paginated, filterable) |
| `GET` | `/users/search?q=...` | Search users (optional: `provider`, `admin`, `limit`) |
| `GET` | `/uploads` | List all uploads (paginated, filterable) |

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
| `sort` | `string` | `size` to sort by total upload size (default: `createdAt`) |
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
| `provider` | `string` | Filter by auth provider (e.g. `google`, `ovh`, `oidc`, `local`) |
| `admin` | `bool` | Filter admin/non-admin users |
