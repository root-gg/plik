# plik

A Helm chart for Plik, a scalable & friendly temporary file upload system.

![Version: 1.4.1](https://img.shields.io/badge/Version-1.4.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.4.1](https://img.shields.io/badge/AppVersion-1.4.1-informational?style=flat-square)

## Installation

### From Helm repository (recommended)

```bash
helm repo add plik https://root-gg.github.io/plik
helm repo update
helm install plik plik/plik
```

### From source

```bash
git clone https://github.com/root-gg/plik.git
cd plik
make helm-install
```

## Quick Start

Minimal install with default settings (in-memory storage, no authentication):

```bash
helm repo add plik https://root-gg.github.io/plik
helm install plik plik/plik
```

With persistence and ingress:

```yaml
# custom-values.yaml
kind: StatefulSet

persistence:
  enabled: true
  size: 50Gi

dbPersistence:
  enabled: true

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: plik.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: plik-tls
      hosts:
        - plik.example.com

plikd:
  FeatureAuthentication: "forced"
  MaxFileSizeStr: "5GB"
  DefaultTTLStr: "7d"
  MaxTTLStr: "30d"
```

```bash
helm install plik plik/plik -f custom-values.yaml
```

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for design decisions around config vs. secrets separation, persistence, and workload kind.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for pod scheduling |
| autoscaling.enabled | bool | `false` | Enable Horizontal Pod Autoscaler |
| autoscaling.maxReplicas | int | `100` | Maximum number of replicas |
| autoscaling.minReplicas | int | `1` | Minimum number of replicas |
| autoscaling.targetCPUUtilizationPercentage | int | `80` | Target CPU utilization percentage |
| dbPersistence.accessModes | list | `["ReadWriteOnce"]` | PVC access modes |
| dbPersistence.enabled | bool | `false` | Enable persistent storage for the SQLite database |
| dbPersistence.path | string | `"/home/plik/server/db"` | Mount path for the database inside the container |
| dbPersistence.size | string | `"1Gi"` | PVC storage size |
| fullnameOverride | string | `""` | Override the full release name |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.repository | string | `"rootgg/plik"` | Docker image repository |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion |
| imagePullSecrets | list | `[]` | Docker registry pull secrets |
| ingress.annotations | object | `{}` | Additional ingress annotations |
| ingress.className | string | `""` | Ingress class name |
| ingress.enabled | bool | `false` | Enable Ingress resource |
| ingress.hosts | list | `[{"host":"chart-example.local","paths":[{"path":"/","pathType":"ImplementationSpecific"}]}]` | Ingress host rules |
| ingress.tls | list | `[]` | Ingress TLS configuration |
| kind | string | `"Deployment"` | Workload kind: `Deployment` or `StatefulSet`. Use StatefulSet if you are using the `file` backend with a PersistentVolume. |
| nameOverride | string | `""` | Override the chart name |
| nodeSelector | object | `{}` | Node selector for pod scheduling |
| persistence.accessModes | list | `["ReadWriteOnce"]` | PVC access modes |
| persistence.enabled | bool | `false` | Enable persistent storage for uploaded files |
| persistence.path | string | `"/home/plik/server/files"` | Mount path for file storage inside the container |
| persistence.size | string | `"10Gi"` | PVC storage size |
| plikd | object | see sub-values | Plik server configuration. Values are rendered into the `plikd.cfg` config file. See [server configuration reference](https://github.com/root-gg/plik/tree/master/server/plikd.cfg) for all options. |
| plikd.AbuseContact | string | `""` | Abuse contact email displayed in the footer |
| plikd.ChangelogDirectory | string | `"../changelog"` | Path to the changelog directory |
| plikd.ClientsDirectory | string | `"../clients"` | Path to the pre-built CLI clients directory |
| plikd.DataBackend | string | `"file"` | Data backend type (`file`, `gcs`, `s3`, `swift`) |
| plikd.DataBackendConfig | object | `{"Directory":"files"}` | Non-sensitive data backend configuration. Keys depend on the backend type. See [data backends](https://github.com/root-gg/plik#data-backends). |
| plikd.Debug | bool | `false` | Enable debug mode |
| plikd.DebugRequests | bool | `false` | Log every HTTP request |
| plikd.DefaultAdminLogin | string | `""` | Login for the default local admin user to create on first startup (empty = disabled). Idempotent: skipped if the user already exists. |
| plikd.DefaultTTLStr | string | `"30d"` | Default upload TTL (e.g. `30d`, `24h`) |
| plikd.DownloadDomain | string | `""` | Custom download domain |
| plikd.DownloadDomainAlias | list | `[]` | Additional download domain aliases |
| plikd.EnableArchiveCompression | bool | `true` | Enable zip compression for archive downloads. When true (default), archives use zip.Deflate (smaller files but higher CPU usage). Set to false to use zip.Store (no compression) to prevent CPU exhaustion DoS on public instances. |
| plikd.EnhancedWebSecurity | bool | `false` | Enable enhanced web security headers (CSP, X-Frame-Options, etc.) |
| plikd.FeatureAuthentication | string | `"disabled"` | Enable user authentication (`enabled`, `disabled`, `forced`) |
| plikd.FeatureClients | string | `"enabled"` | Enable pre-built CLI clients download page |
| plikd.FeatureComments | string | `"enabled"` | Enable upload comments |
| plikd.FeatureDeleteAccount | string | `"enabled"` | Allow users to delete their account |
| plikd.FeatureExtendTTL | string | `"disabled"` | Allow users to extend an existing TTL |
| plikd.FeatureGithub | string | `"enabled"` | Show GitHub link in the footer |
| plikd.FeatureLocalLogin | string | `"enabled"` | Enable local login |
| plikd.FeatureOneShot | string | `"enabled"` | Enable one-shot downloads |
| plikd.FeaturePassword | string | `"enabled"` | Enable password-protected uploads |
| plikd.FeatureRemovable | string | `"enabled"` | Enable removable uploads |
| plikd.FeatureSetTTL | string | `"enabled"` | Allow users to set a custom TTL |
| plikd.FeatureStream | string | `"enabled"` | Enable streaming uploads |
| plikd.FeatureText | string | `"enabled"` | Enable plain-text paste mode |
| plikd.GitHubApiClientID | string | `""` | GitHub OAuth app client ID |
| plikd.GitHubValidOrganizations | list | `[]` | Restrict to GitHub organization members (empty = allow all) |
| plikd.GoogleApiClientID | string | `""` | Google OAuth2 client ID |
| plikd.GoogleValidDomains | list | `[]` | Allowed Google domains (empty = allow all) |
| plikd.ListenAddress | string | `"0.0.0.0"` | HTTP listen address |
| plikd.ListenPort | int | `8080` | HTTP listen port |
| plikd.LogLevel | string | `"INFO"` | Log level (`DEBUG`, `INFO`, `WARNING`, `CRITICAL`) |
| plikd.MaxFilePerUpload | int | `1000` | Maximum number of files per upload |
| plikd.MaxFileSizeStr | string | `"10GB"` | Maximum file size per upload (e.g. `10GB`, `unlimited`) |
| plikd.MaxTTLStr | string | `"30d"` | Maximum upload TTL |
| plikd.MaxUserSizeStr | string | `"unlimited"` | Maximum total size per user (e.g. `10GB`, `unlimited`) |
| plikd.MetadataBackendConfig | object | `{"ConnectionString":"/home/plik/server/db/plik.db","Debug":false,"Driver":"sqlite3"}` | Metadata backend configuration (database driver, connection string) |
| plikd.MetricsAddress | string | `"0.0.0.0"` | Prometheus metrics listen address |
| plikd.MetricsPort | int | `0` | Prometheus metrics port (0 = disabled) |
| plikd.NoWebInterface | bool | `false` | Disable the web interface |
| plikd.OIDCClientID | string | `""` | OpenID Connect client ID |
| plikd.OIDCProviderName | string | `"OpenID"` | Display name for the OIDC provider |
| plikd.OIDCProviderURL | string | `""` | OpenID Connect provider discovery URL |
| plikd.OIDCRequireVerifiedEmail | bool | `false` | Require verified email from OIDC provider |
| plikd.OIDCValidDomains | list | `[]` | Allowed OIDC email domains (empty = allow all) |
| plikd.OvhApiEndpoint | string | `""` | OVH API endpoint (e.g. `ovh-eu`, `ovh-ca`) |
| plikd.Path | string | `""` | URL path prefix (e.g. `/plik`) |
| plikd.PlikDomain | string | `""` | Public webapp URL for OAuth redirects, CORS, and download domain redirects (e.g., `https://plik.example.com`) |
| plikd.SessionTimeout | string | `"365d"` | User session timeout (e.g. `365d`, `24h`) |
| plikd.SourceIpHeader | string | `""` | HTTP header to use for source IP (e.g. `X-Forwarded-For`) |
| plikd.SslCert | string | `"plik.crt"` | Path to TLS certificate |
| plikd.SslEnabled | bool | `false` | Enable HTTPS |
| plikd.SslKey | string | `"plik.key"` | Path to TLS private key |
| plikd.TlsVersion | string | `"tlsv10"` | Minimum TLS version (`tlsv10`, `tlsv11`, `tlsv12`, `tlsv13`) |
| plikd.UploadWhitelist | list | `[]` | IP whitelist for uploads (empty = allow all) |
| plikd.WebappDirectory | string | `"../webapp/dist"` | Path to the webapp distribution directory |
| podAnnotations | object | `{}` | Additional pod annotations |
| podSecurityContext | object | `{}` | Pod-level security context |
| replicaCount | int | `1` | Number of pod replicas |
| resources | object | `{}` | CPU/memory resource requests and limits |
| secrets.dataBackend | object | `{}` | Sensitive data backend config (merged with `plikd.DataBackendConfig` at runtime). Injected as `PLIKD_DATA_BACKEND_CONFIG` JSON. |
| secrets.defaultAdminPassword | string | `""` | Default admin password (injected as `PLIKD_DEFAULT_ADMIN_PASSWORD`). If empty, a random password is generated and logged on first startup. |
| secrets.existingSecret | string | `""` | Use an existing Kubernetes Secret instead of creating one. The Secret must contain the relevant env var keys. |
| secrets.githubApiSecret | string | `""` | GitHub OAuth app client secret (injected as `PLIKD_GIT_HUB_API_SECRET`) |
| secrets.googleApiSecret | string | `""` | Google OAuth2 client secret (injected as `PLIKD_GOOGLE_API_SECRET`) |
| secrets.metadataBackend | object | `{}` | Sensitive metadata backend config (e.g. database password). Injected as `PLIKD_METADATA_BACKEND_CONFIG` JSON. |
| secrets.oidcClientSecret | string | `""` | OIDC client secret (injected as `PLIKD_OIDC_CLIENT_SECRET`) |
| secrets.ovhApiKey | string | `""` | OVH API key (injected as `PLIKD_OVH_API_KEY`) |
| secrets.ovhApiSecret | string | `""` | OVH API secret (injected as `PLIKD_OVH_API_SECRET`) |
| securityContext | object | `{}` | Container-level security context |
| service.annotations | object | `{}` | Additional service annotations |
| service.port | int | `8080` | Service port |
| service.type | string | `"ClusterIP"` | Kubernetes Service type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.create | bool | `false` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template |
| tolerations | list | `[]` | Tolerations for pod scheduling |
