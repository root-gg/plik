# Architecture — Helm Chart (`charts/plik/`)

> Kubernetes deployment chart for Plik. For system-wide overview, see the root [ARCHITECTURE.md](../../ARCHITECTURE.md).

---

## Structure

```
charts/plik/
├── Chart.yaml                  ← Chart metadata (version set at release time via __VERSION__)
├── values.yaml                 ← All user-configurable values (annotated for helm-docs)
├── README.md.gotmpl            ← helm-docs template for generating README.md
├── README.md                   ← Auto-generated values reference (do not edit manually)
├── CHANGELOG.md                ← Keep-a-Changelog (update [Unreleased] before each release)
├── ARCHITECTURE.md             ← this file
└── templates/
    ├── _helpers.tpl            ← Template helpers (plik.fullname, plik.secretName, etc.)
    ├── configmap.yaml          ← Renders plikd.cfg from non-sensitive plikd.* values
    ├── secret.yaml             ← Renders Kubernetes Secret from secrets.* values
    ├── serviceaccount.yaml     ← Optional ServiceAccount (when serviceAccount.create is true)
    ├── deployment.yaml         ← Deployment or StatefulSet (controlled by .Values.kind)
    ├── service.yaml            ← ClusterIP service on port 8080
    ├── ingress.yaml            ← Optional Ingress resource
    ├── pvc.yaml                ← PVC for file/db data (when persistence/dbPersistence enabled)
    ├── extra-objects.yaml      ← Renders arbitrary extra Kubernetes manifests (extraObjects)
    └── NOTES.txt               ← Post-install instructions
```

---

## Key Design Decisions

### Config vs. Secrets separation

| Category | Source | Rendered to | Mechanism |
|---|---|---|---|
| Non-sensitive config | `plikd.*` in `values.yaml` | ConfigMap (`plikd.cfg`) | TOML config file |
| Sensitive credentials | `secrets.*` in `values.yaml` | Kubernetes Secret | `envFrom.secretRef` → env var override |

Sensitive fields in `secrets.*` include OAuth client secrets (`googleApiSecret`, `oidcClientSecret`, `githubApiSecret`, `ovhApiKey`, `ovhApiSecret`), backend credentials (`dataBackend`, `metadataBackend`), and the default admin password (`defaultAdminPassword`).

The server loads the config file first, then applies env var overrides via `PLIKD_` prefix + screaming snake case (e.g., `GoogleAPISecret` → `PLIKD_GOOGLE_API_SECRET`). Map-type fields like `DataBackendConfig` accept JSON and **merge** into the config file map.

### BYO Secret (existingSecret)

Set `secrets.existingSecret: "my-secret-name"` to skip Secret creation and reference an externally managed secret (Vault, Sealed Secrets, ESO). The `plik.secretName` helper in `_helpers.tpl` resolves the correct name everywhere.

### Persistence

Two independent PVCs:
- `persistence` — uploaded file data at `/home/plik/server/files`
- `dbPersistence` — SQLite database at `/home/plik/server/db`

Both default to `emptyDir` when disabled. For `StatefulSet` mode, volumes use `volumeClaimTemplates`.

### Versioning

Chart `version` and `appVersion` in `Chart.yaml` use `__VERSION__` placeholders, replaced at release time by `releaser/helm_release.sh` to match the app release tag.

### extraObjects — deploying supplementary resources

`extraObjects` allows users to deploy arbitrary Kubernetes manifests alongside the chart (e.g. ExternalSecrets, NetworkPolicies, RBAC rules). It supports three forms to accommodate different values file strategies:

| Form | Value type | Multi-file merge | tpl support | When to use |
|------|-----------|-----------------|-------------|-------------|
| **Dict** | `extraObjects: {key: {...}}` | ✅ Helm deep-merges dicts | ✅ Yes | **Recommended** — each `-f` file adds its own named key independently |
| **List** | `extraObjects: [...]` | ❌ Last file wins | ✅ Yes | Single values file |
| **String** | `extraObjects: \|` | ❌ Last file wins | ✅ Yes (whole block evaluated first) | Inline Go template expressions |

The dict form is the default (`extraObjects: {}`) because Helm performs a deep merge on maps across multiple `-f` files, whereas lists and strings are always fully replaced by the last file. Each dict key is an arbitrary label used only to allow merging — only the value (the manifest) is rendered.

All forms pass manifests through `tpl` before rendering, so Go template helpers such as `{{ include "plik.fullname" . }}` and `{{ .Release.Name }}` work in all cases.

