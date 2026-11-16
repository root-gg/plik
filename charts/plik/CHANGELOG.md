# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security
- Secure-by-default pod and container security contexts, compliant with the Kubernetes "restricted" Pod Security Standard: `runAsNonRoot`, UID/GID `1000` (the image's `plik` user), `fsGroup: 1000`, `RuntimeDefault` seccomp profile, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, all capabilities dropped. A writable `/tmp` `emptyDir` is mounted. If you previously overrode `podSecurityContext` / `securityContext`, your keys are now merged with these defaults — set a key to `null` to remove it

### Added
- Optional `NetworkPolicy` (`networkPolicy.enabled`, default `false`) restricting ingress to the http port (and `plikd.MetricsPort` when set), with configurable `ingressFrom` peers and optional `egress` rules
- `persistence.emptyDirSizeLimit` / `dbPersistence.emptyDirSizeLimit` to cap the `emptyDir` volumes used when persistence is disabled
- New `startupProbe` (TCP, up to 5 minutes by default) so slow starts such as long metadata migrations are not killed by the liveness probe

### Changed
- Liveness and readiness probes are now distinct: liveness is a TCP check, readiness is an HTTP GET on `/health` (instead of `/` for both). Probe timings are configurable via `livenessProbe`, `readinessProbe` and `startupProbe` (each with an `enabled` toggle)

### Fixed
- Probes now honor `plikd.Path` (previously `GET /` returned 404 when a path prefix was configured, so the pod never became ready) and use HTTPS when `plikd.SslEnabled` is true

### Removed
- Unused `autoscaling.*` values — no HorizontalPodAutoscaler template existed, and running several replicas is unsafe with the default SQLite metadata backend / `file` data backend

## [1.4.3]

### Fixed
- Add missing `serviceaccount.yaml` template — `serviceAccount.create: true` now correctly creates a ServiceAccount resource

### Added
- New `serviceAccount.automount` value to control `automountServiceAccountToken` on the ServiceAccount (default: `false`)
- New `extraObjects` value to deploy arbitrary extra Kubernetes manifests alongside the chart (e.g. ExternalSecrets, NetworkPolicies, RBAC rules). Supports three forms: **dict** (recommended — Helm deep-merges dicts across multiple `-f` files, so each file can add its own key independently), **list** (single values file), or **string** (supports inline Go template expressions). All forms support `tpl`.

## [1.4.2]

### Added
- New `DefaultAdminLogin` config option (and `secrets.defaultAdminPassword` in Helm) to automatically create a local admin user on first startup — idempotent, skipped if the user already exists
- New `FeatureApiTokens` config option to globally disable API token creation and usage (`disabled`/`enabled`, default: `enabled`)

### Changed
- New `BucketLookup` S3 data backend config option (`"auto"` / `"dns"` / `"path"` — path-style required for Cloudflare R2 and some MinIO deployments)

## [1.4.1]

### Changed
- New `StreamTimeoutStr` server config option (configurable streaming download timeout)
- `DownloadDomain` behavior fix: UI/API restriction only applies when `PlikDomain` is also set
- `settings.json` footer support (custom HTML footer takes precedence over `AbuseContact`)

## [1.4.0]

### Added
- `PlikDomain` config value for OAuth redirects, CORS, and download domain redirects
- `helm-docs` annotations in `values.yaml` and generated chart `README.md`

## [1.4-RC5]

No Helm chart changes in this release.

## [1.4-RC4] — Initial Release

### Added
- Helm chart for deploying Plik on Kubernetes
- `secrets:` top-level block in `values.yaml` for all sensitive credentials
  (`googleApiSecret`, `ovhApiKey`, `ovhApiSecret`, `oidcClientSecret`, `dataBackend`, `metadataBackend`)
- `secrets.existingSecret` — bring-your-own Secret support
- `plik.secretName` Helm helper for consistent Secret name resolution
- `secret.yaml` reads credentials exclusively from `secrets.*` values
- `deployment.yaml` with `optional: true` on `envFrom.secretRef` so pods start cleanly without a Secret
- `dbPersistence` — dedicated PVC for the SQLite metadata database
- Ingress template, post-install notes, Kubernetes deployment guide
- Explicit key ordering in `configmap.yaml` for deterministic rendering
