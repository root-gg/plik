# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.1] - 2026-02-20
### Added
- New top-level `secrets:` block in `values.yaml` to hold all sensitive credentials
  (`googleApiSecret`, `ovhApiKey`, `ovhApiSecret`, `oidcClientSecret`, `dataBackend`, `metadataBackend`)
- `secrets.existingSecret` — bring-your-own Secret support (replaces top-level `existingSecret`)
- `plik.secretName` Helm helper for consistent Secret name resolution

### Changed
- `secret.yaml` now reads credentials exclusively from `secrets.*` values
- `deployment.yaml` now uses explicit `env[].valueFrom.secretKeyRef` with `optional: true`
  instead of the broad `envFrom.secretRef`, for safer and more auditable env var injection
- `configmap.yaml` no longer contains any sensitive values; secret fields are removed
  from the rendered `plikd.cfg`

### Removed
- Top-level `existingSecret` value (replaced by `secrets.existingSecret`)
- Sensitive fields from under `plikd.*` in `values.yaml`:
  `GoogleApiSecret`, `OvhApiKey`, `OvhApiSecret`, `OIDCClientSecret`,
  `DataBackendConfig.SecretAccessKey`, `DataBackendConfig.ApiKey`,
  `MetadataBackendConfig.Password`

> [!IMPORTANT]
> **Migration**: If you were setting `plikd.GoogleApiSecret`, `plikd.OvhApiKey`,
> `plikd.OIDCClientSecret`, etc. or using the top-level `existingSecret`, you must update
> your `values.yaml` overrides to use the new `secrets.*` structure.

## [0.3.0] - 2026-02-20

### Added
- `dbPersistence` — dedicated PVC for the SQLite metadata database, independent of the file data PVC
  - Mounts at `/home/plik/server/db` (does not shadow the server binary or config)
  - Supported for both `Deployment` (named PVC `<release>-db`) and `StatefulSet` (volumeClaimTemplate)
  - Defaults to `emptyDir` when disabled, fully backward-compatible

### Changed
- Default `MetadataBackendConfig.ConnectionString` changed from `"plik.db"` (relative) to `"/home/plik/server/db/plik.db"` (absolute path inside the `dbPersistence` volume)

## [0.2.0] - 2026-02-20
### Added
- Ingress template (`templates/ingress.yaml`)
- Post-install notes (`templates/NOTES.txt`)
- Missing config fields: `FeatureLocalLogin`, `FeatureDeleteAccount`, `OvhApiKey`, `OIDCProviderName`, `OIDCRequireVerifiedEmail`
- Kubernetes deployment guide (`docs/guide/kubernetes.md`)

### Changed
- Rewrite `configmap.yaml` with explicit key ordering (fixes non-deterministic rendering)
- Bump `appVersion` to `1.4-RC3`
- Upgrade GitHub Actions: `peaceiris/actions-gh-pages` v4, `azure/setup-helm` v4, `helm/chart-releaser-action` v1.7.0

### Fixed
- `DisableLocalLogin` renamed to `FeatureLocalLogin` (matches plikd config naming)
- `OvhApiKey` env var injection added to `secret.yaml`

## [0.1.1] - 2024-02-13
### Fixed
- Fixed release workflow configuration

## [0.1.0] - 2024-02-13
### Added
- Initial Chart Implementation
