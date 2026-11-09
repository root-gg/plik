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
| `POST` | `/resumable/{uploadID}/{filename}` | Add file as a resumable upload |
| `HEAD` | `/resumable/{uploadID}/{fileID}/{filename}` | Resumable upload offset |
| `PATCH` | `/resumable/{uploadID}/{fileID}/{filename}` | Append to a resumable upload |
| `DELETE` | `/resumable/{uploadID}/{fileID}/{filename}` | Cancel a resumable upload |
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

### Resumable Upload (/resumable/{uploadID}/{fileID}/{filename})

Uploads a file in pieces, so an interrupted upload resumes instead of starting over. The protocol follows the IETF [Resumable Uploads for HTTP](https://datatracker.ietf.org/doc/draft-ietf-httpbis-resumable-upload/) draft (`draft-ietf-httpbis-resumable-upload-15`). A file declared at upload creation is the upload resource; `POST /resumable/{uploadID}/{filename}` adds a file to an upload and creates its upload resource.

| Request | Response |
|---------|----------|
| `POST /resumable/{uploadID}/{filename}` with `Upload-Complete`, optional `Upload-Length` and `?reference=` | `201` with `Location` (the upload resource) and `Upload-Offset`. The content is the beginning of the file, or the whole file with `Upload-Complete: ?1` (`200` with the file JSON). A creation with the reference of a file of the upload returns that file instead of adding another one |
| `HEAD` | `204` with `Upload-Offset` (bytes received), `Upload-Complete` (`?1` once the file is uploaded) and `Upload-Limit: max-size=<MaxFileSize>` |
| `PATCH` with `Content-Type: application/partial-upload`, `Upload-Offset` and `Upload-Complete: ?0` | `204` with the new `Upload-Offset` |
| `PATCH` with `Upload-Complete: ?1` (the content ends the file, it can be empty) | `200` with the file JSON, like `POST /file`, and `Upload-Complete: ?1` |
| `PATCH` at another offset than the server's | `409` with `Upload-Offset` and a `mismatching-upload-offset` problem (`expected-offset`, `provided-offset`) |
| `DELETE` | Cancels the upload, like removing the file |

```bash
curl -X PATCH -H "X-UploadToken: $TOKEN" -H "Content-Type: application/partial-upload" \
  -H "Upload-Offset: 0" -H "Upload-Complete: ?0" \
  --data-binary @part0 "$PLIK/resumable/$UPLOAD/$FILE/video.mp4"
```

Error responses carry `Upload-Complete: ?1` when the server processed the whole file and resuming cannot help, for example when the user quota is exceeded. Appended data is staged in the server's `ChunkDirectory` until the upload completes, then the file is saved to the data backend. Staged data is deleted after a day without appends. Replicas receiving requests for the same resumable upload share `ChunkDirectory` on storage supporting file locks across hosts. NFS mounts require close-to-open consistency and `lookupcache=positive`. Not available for stream uploads.

### Download File

The upload token is not required for public uploads. For password-protected uploads, provide HTTP Basic auth with the upload's login/password.

HTTP Range requests (`Range` header) are supported on file downloads, allowing partial content retrieval (206 responses).

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
