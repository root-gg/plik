# Security

Plik allows users to upload and serve any content as-is. Hosting untrusted content raises security concerns that Plik addresses with several mechanisms.

## Content-Type Override

For security reasons, Plik doesn't trust user-provided MIME types and relies solely on server-side detection. This means some files may not render properly in browsers or embedded viewers that require the correct MIME type.

### Dangerous Content-Type Neutralization

Plik automatically neutralizes content types that could execute code in the browser:

| Original type | Served as | Reason |
|---|---|---|
| `text/html`, `*html*` | `application/octet-stream` | Prevents inline script execution |
| `image/svg+xml`, `*svg*` | `application/octet-stream` | SVG can contain `onload` JavaScript handlers |
| `text/xml`, `*xml*` | `application/octet-stream` | XML can be parsed and rendered by browsers |
| `application/javascript`, `*javascript*` | `application/octet-stream` | Prevents script execution |
| `application/x-shockwave-flash` | `application/octet-stream` | Flash content |
| `application/pdf` | `application/octet-stream` | PDF can contain JavaScript |

::: tip
This protection is always active regardless of `EnhancedWebSecurity`. Use the `?dl=true` query parameter to force a download with `Content-Disposition: attachment`.
:::

::: warning Office format detection limitation
Office formats like `.pptx`, `.docx`, and `.xlsx` are ZIP archives internally, so Go's built-in MIME detector (`http.DetectContentType`) identifies them as `application/zip` instead of their proper types (e.g., `application/vnd.openxmlformats-officedocument.presentationml.presentation`).
:::

## Enhanced Web Security

When `EnhancedWebSecurity` is enabled in `plikd.cfg`, Plik sets additional HTTP headers:

- **X-Content-Type-Options**: `nosniff`
- **X-XSS-Protection**: enabled
- **X-Frame-Options**: deny
- **Content-Security-Policy**: restrictive policy disabling resource loading, XHR, iframes
- **Strict-Transport-Security**: `max-age=31536000` (1 year) — also set when `SslEnabled` is true
- **Secure Cookies**: session cookies only transmitted over HTTPS — also set when `SslEnabled` is true

::: warning
Enhanced security will break audio/video playback, PDF rendering, and other rich content features. Disable it if you need those capabilities.
:::

::: danger Authentication requires HTTPS with Secure Cookies
When `EnhancedWebSecurity` or `SslEnabled` is enabled, session cookies have the `Secure` flag set and can only be transmitted over HTTPS connections. Authentication will not work over plain HTTP.
:::

## Upload Password Protection

When `FeaturePassword` is enabled, uploads can be protected with a login/password pair. Credentials are transmitted via HTTP Basic Authentication.

Passwords are hashed using **bcrypt(sha256(credentials))** before storage; the plaintext is never persisted. The SHA-256 pre-hash ensures credentials of any length are securely handled within bcrypt's input constraints.

| Parameter | Limit |
|-----------|-------|
| Login     | 128 characters max |
| Password  | 128 characters max |

::: tip
Legacy uploads (created before version 1.4) use MD5 hashing and continue to work until they expire.
:::

## Archive Compression

By default, archive downloads use `zip.Deflate` compression (`EnableArchiveCompression = true`). On public instances, consider disabling compression to reduce CPU load from archive generation.

::: tip
With compression disabled, archives use `zip.Store` (raw copy) — minimal CPU at the cost of larger downloads.
:::

## Removable Uploads

When `FeatureRemovable` is enabled and an upload is created with `removable: true`, **anyone with the upload URL can delete the upload and its files** — no upload token or authentication is required. This is by design: the `removable` flag is intended for ephemeral, public uploads where ease of cleanup is prioritized over access control.

::: warning
If you need to control who can delete an upload, do **not** set `removable: true`. Only the upload owner (via upload token, API token, or session) can delete non-removable uploads.
:::

## Download Domain

It is recommended to serve uploaded files on a separate (sub-)domain to:

- Protect against phishing links using your main domain
- Protect Plik's session cookie from being exposed to uploaded content
- Prevent uploaded JavaScript from making API calls as the authenticated user

### Configuration

When using a download domain, three configuration options work together:

```toml
PlikDomain     = "https://plik.root.gg"       # Main webapp URL (recommended when DownloadDomain is set)
DownloadDomain = "https://dl.plik.root.gg"    # Separate domain for serving files
DownloadDomainAlias = []                       # Additional accepted download hosts
```

**`PlikDomain`** — The public URL where the webapp is served. **Domain only — no path component.** If a path is included (e.g., `https://plik.example.com/app`), Plik will strip it with a startup warning and use the `Path` config option instead. When set:
- OAuth redirect URLs use this domain instead of the `Referer` header
- `GetServerURL()` returns this domain for CLI quick upload URLs
- CORS headers are configured when `DownloadDomain` is also set

::: tip PlikDomain does not restrict downloads
Setting `PlikDomain` alone does **not** enforce any domain check on file downloads — files remain accessible from any host. To restrict downloads to a specific domain, you must also set `DownloadDomain`.
:::

**`DownloadDomain`** — The domain that serves uploaded files. **Domain only — no path component.** If a path is included, it will be stripped with a startup warning. When set:
- File/archive download requests are rejected unless the `Host` header matches the download domain (or an alias)
- Non-file requests (webapp UI, API) on the download domain are **blocked** when `PlikDomain` is also set (see below)
- The computed `downloadURL` field (returned in `/config` and upload API responses) equals `DownloadDomain + Path`

::: info Path prefix and DownloadURL
The `Path` config option is the **single source of truth** for URL path prefixes. Setting it to `/sub` means:
- All API and webapp routes are served under `/sub`
- The `downloadURL` field in API responses is `DownloadDomain + /sub` (or `PlikDomain + /sub` if no `DownloadDomain`)
- File download links from quick upload and the web UI automatically include `/sub`

Do **not** put the path in `PlikDomain` or `DownloadDomain` — use `Path` instead.
:::

**`DownloadDomainAlias`** — Additional hostnames accepted for file downloads. Useful when:
- Accessing the server via `localhost` during development
- The reverse proxy uses a different host internally

### Security: UI Restriction on Download Domain

When **both** `PlikDomain` and `DownloadDomain` are configured, Plik **blocks** the webapp UI and API endpoints from being served on the download domain. This is critical because:

- An attacker could share a link like `https://dl.plik.root.gg/` — the user sees the familiar Plik UI but on the download domain
- If the user logs in on this domain, their session cookie is exposed to uploaded content
- Uploaded JavaScript could make authenticated API calls on behalf of the user

::: warning Only DownloadDomain set (no PlikDomain)
If only `DownloadDomain` is set without `PlikDomain`, the UI/API restriction is **disabled** — all requests pass through normally. This preserves backward compatibility with pre-1.4 deployments. A warning is logged at startup in this case.
:::

**How it works (when both domains are set):**

| Request type | Download domain behavior |
|---|---|
| File/stream/archive | ✅ Served normally |
| `/health` | ✅ Served normally (for load balancer probes) |
| Everything else (UI, API) | 🔄 Redirect to `PlikDomain` |

::: tip Ideal setup — use both domains
For the best security and user experience, configure **both** `PlikDomain` and `DownloadDomain`:

```toml
PlikDomain     = "https://plik.root.gg"
DownloadDomain = "https://dl.plik.root.gg"
```

This gives you:
- **Domain isolation**: uploaded files served separately from the webapp
- **Smooth redirects**: users who land on the download domain are redirected to the webapp
- **CORS support**: the file viewer and E2EE decrypt work cross-origin
- **Reliable OAuth**: redirect URLs use PlikDomain instead of the fragile Referer header

:::

::: info How CORS works here
When both domains are configured, Plik adds `Access-Control-Allow-Origin: <PlikDomain>` headers to download responses. This allows the webapp's JavaScript to fetch file content cross-origin (for the file viewer and E2EE decrypt), while still preventing uploaded JavaScript on the download domain from accessing the webapp's origin.
:::

### Troubleshooting: Redirect Loops

If you see this error:

```
Invalid download domain 127.0.0.1:8080, expected plik.root.gg
```

`DownloadDomain` checks the `Host` header of incoming HTTP requests. By default, reverse proxies like Nginx and Apache do not forward this header. Make sure to configure:

```
Apache mod_proxy: ProxyPreserveHost On
Nginx:            proxy_set_header Host $host;
```

## XSRF Protection

Plik uses a dual-cookie XSRF protection mechanism:

1. The `plik-xsrf` cookie value must be copied into the `X-XSRFToken` HTTP header for all mutating authenticated requests
2. This prevents cross-site request forgery attacks

## Upload Restrictions

### Source IP Whitelist

Restrict uploads and user creation to specific IP ranges:

```toml
UploadWhitelist = ["10.0.0.0/8", "192.168.1.0/24"]
```

### Authentication

Set `FeatureAuthentication = "forced"` to require authentication for all uploads.

### Upload Tokens

Authenticated users can generate upload tokens to link CLI uploads to their account. Tokens are sent via the `X-PlikToken` HTTP header.

## Link Preview Bot Protection

Messaging apps like Slack, Telegram, WhatsApp, Signal, Discord, and others generate link previews by fetching shared URLs. This is problematic for:

- **One-shot uploads**: the bot's preview request counts as the single allowed download, deleting the file before the intended recipient can access it
- **Streaming uploads**: the bot consumes the stream data, leaving nothing for the real downloader

Plik automatically blocks known messaging app link preview bots from downloading one-shot and streaming files, returning a **406 Not Acceptable** response. Normal (multi-download) uploads are not affected — bots can still generate previews for those.

::: tip No configuration needed
This protection is always active and requires no configuration. It uses a hardcoded list of known bot user-agent strings that is maintained with Plik releases.
:::

### Blocked Bots

Slack, Telegram, WhatsApp, Signal, Facebook/Messenger, Discord, Skype, Viber, LinkedIn, Twitter/X, Microsoft Teams, Wire, Mattermost, Rocket.Chat, and Zulip.

## Server-Side Encryption (S3)

When using the S3 data backend, Plik supports server-side encryption to protect uploaded files at rest.

### Encryption Modes

| Mode | Key Management | Description |
|------|---------------|-------------|
| `S3` | S3 backend | The S3 service manages encryption keys transparently |
| `SSE-C` | Plik | Plik generates a unique 32-byte key per file |

### SSE-C Threat Model

With `SSE-C`, encryption keys are generated by Plik and stored in the **metadata database** (in the `BackendDetails` column of each file record).

::: warning Security boundary
SSE-C protects against **S3 bucket compromise in isolation** — if an attacker gains access to the S3 storage but not the metadata database, uploaded files remain encrypted and unreadable.

However, if both the S3 bucket **and** the metadata database are compromised, the attacker can retrieve the encryption keys and decrypt all files.
:::

**Recommendations:**
- For maximum security, host the metadata database on **separate infrastructure** from the S3 storage
- If both are co-located (e.g., SQLite + local MinIO on the same host), SSE-C provides limited additional protection
- Consider `S3` mode if your S3 provider already offers robust at-rest encryption with their own key management

