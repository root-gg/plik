# Configuration

Plik is configured via a TOML file (`plikd.cfg`) and optional environment variable overrides.

## Config File Locations

The server looks for configuration in this order:
1. `--config` flag
2. `PLIKD_CONFIG` environment variable
3. `./plikd.cfg` (current directory)
4. `/etc/plikd.cfg`

## Environment Variable Override

Any config parameter can be set via environment variable using `PLIKD_` prefix with SCREAMING_SNAKE_CASE:

```bash
PLIKD_DEBUG_REQUESTS=true ./plikd
PLIKD_LISTEN_PORT=9090 ./plikd
```

Arrays and maps must be provided in JSON format. Arrays are overridden, maps are merged:

```bash
PLIKD_DATA_BACKEND_CONFIG='{"Directory":"/var/files"}' ./plikd
```

## Server Settings

| Parameter | Default          | Description                                                                                                                                                                                                               |
|-----------|------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ListenPort` | `8080`           | HTTP server port                                                                                                                                                                                                          |
| `ListenAddress` | `0.0.0.0`        | HTTP server bind address                                                                                                                                                                                                  |
| `MetricsPort` | `0`              | Prometheus metrics port (0 = disabled)                                                                                                                                                                                    |
| `Path` | `""`             | HTTP root path prefix                                                                                                                                                                                                     |
| `SslEnabled` | `false`          | Enable TLS                                                                                                                                                                                                                |
| `SslCert` / `SslKey` | —                | TLS certificate and key paths                                                                                                                                                                                             |
| `TlsVersion` | `tlsv10`         | Minimum TLS version                                                                                                                                                                                                       |
| `NoWebInterface` | `false`          | Disable web UI                                                                                                                                                                                                            |
| `PlikDomain` | `""`             | Public webapp URL (e.g., `https://plik.example.com`). **Domain only — no path.** Used for OAuth redirects and CORS. Does **not** restrict downloads on its own — set `DownloadDomain` for that                            |
| `DownloadDomain` | `""`             | Enforce download domain (e.g., `https://dl.plik.example.com`). **Domain only — no path.** UI/API blocking and CORS require `PlikDomain` too                                                                               |
| `DownloadDomainAlias` | `[]`             | Additional accepted download hosts                                                                                                                                                                                        |
| `AssumeHTTPS` | `false`          | Enable HSTS + Secure cookies (auto-enabled from `SslEnabled` or HTTPS `PlikDomain`)                                                                                                                                       |
| `SessionTimeout` | `365d`           | Authentication session duration                                                                                                                                                                                           |
| `AbuseContact` | `""`             | Abuse contact email shown in footer. `settings.json` `"footer"` takes precedence when set                                                                                                                                 |
| `WebappDirectory` | `../webapp/dist` | Web UI static files directory                                                                                                                                                                                             |
| `ClientsDirectory` | `../clients`     | CLI client binaries directory                                                                                                                                                                                             |
| `ChangelogDirectory` | `../changelog`   | Release changelog directory                                                                                                                                                                                               |
| `SourceIpHeader` | `""`             | Header for real IP behind proxy (e.g., `X-Forwarded-For`)                                                                                                                                                                 |
| `UploadWhitelist` | `[]`             | Restrict uploads to IP ranges (CIDR)                                                                                                                                                                                      |
| `EnableArchiveCompression` | `true`           | Enable zip compression for archive downloads. Set to `false` to use `zip.Store` (no compression) to prevent CPU exhaustion on public instances. See [Security — Archive Compression](/guide/security#archive-compression) |
| `UploadIDLength` | `16`             | Length of ID that will be used to uniquely identify uploads.                                                                                                                                                              |
| `UploadIDinB32` | `false`          | Whether to use lowercase Crockford Base32 for ease of manual copying Upload IDs. If false, uses Base62 instead                                                                                                            |

## Limits

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MaxFileSizeStr` | `10GB` | Maximum file size (or `"unlimited"`) |
| `MaxUserSizeStr` | `unlimited` | Default per-user storage limit |
| `MaxFilePerUpload` | `1000` | Max files per upload |
| `DefaultTTLStr` | `30d` | Default time-to-live |
| `MaxTTLStr` | `30d` | Maximum TTL (0 = no limit) |

## Feature Flags

Features can be set to one of four states:

| Value | Behavior |
|-------|----------|
| `disabled` | Feature always off |
| `enabled` | Feature available, opt-in |
| `default` | Feature available, opt-out (on by default) |
| `forced` | Feature always on |

| Flag | Default | Description |
|------|---------|-------------|
| `FeatureAuthentication` | `disabled` | User authentication (`forced` = no anonymous uploads) |
| `FeatureOneShot` | `enabled` | Files deleted after first download |
| `FeatureRemovable` | `enabled` | Anyone can delete files |
| `FeatureStream` | `enabled` | Direct uploader-to-downloader streaming |
| `StreamTimeoutStr` | `5m` | Max wait for a streaming download to start before releasing the upload goroutine (`0` = no timeout) |
| `FeaturePassword` | `enabled` | Password-protected uploads |
| `FeatureComments` | `enabled` | Markdown comments on uploads |
| `FeatureSetTTL` | `enabled` | Custom TTL setting |
| `FeatureExtendTTL` | `disabled` | Extend TTL on each download |
| `FeatureClients` | `enabled` | Show CLI download button in UI (enable/disable only) |
| `FeatureApiTokens` | `enabled` | API token creation, usage via `X-PlikToken` header, and CLI auth flow. When `disabled` + `FeatureAuthentication=forced`, `FeatureClients` is also auto-disabled (enable/disable only) |
| `FeatureGithub` | `enabled` | Show source code link in UI |
| `FeatureText` | `enabled` | Text upload dialog |
| `FeatureLocalLogin` | `enabled` | Local login form (enable/disable only) |
| `FeatureDeleteAccount` | `enabled` | Allow users to delete their own account (enable/disable only) |

## Full Example

See the [default plikd.cfg](https://github.com/root-gg/plik/blob/master/server/plikd.cfg) for a fully commented configuration file.
