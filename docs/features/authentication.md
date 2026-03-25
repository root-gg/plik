# Authentication

Plik supports multiple authentication providers. Enable with:

```toml
FeatureAuthentication = "enabled"   # Opt-in
# or
FeatureAuthentication = "forced"    # All uploads require auth
```

## Local Accounts

Create users via the server CLI:

```bash
./plikd --config ./plikd.cfg user create --login root --name Admin --admin
# Generated password for user root is 08ybEyh2KkiMho8dzpdQaJZm78HmvWGC
```

For Docker or Kubernetes deployments, you can instead provision a local admin user automatically on first startup via config:

```toml
DefaultAdminLogin    = "admin"
DefaultAdminPassword = "s3cr3tpass"   # Optional: auto-generated and logged if omitted
```

Or equivalently via environment variables:

```bash
PLIKD_DEFAULT_ADMIN_LOGIN=admin
PLIKD_DEFAULT_ADMIN_PASSWORD=s3cr3tpass
```

This is idempotent — if the user already exists it is left untouched. In Kubernetes/Helm, set `plikd.DefaultAdminLogin` in `values.yaml` and put the password in `secrets.defaultAdminPassword` so it is stored in a Secret rather than the ConfigMap.

> [!WARNING]
> This feature is intended for **bootstrap and troubleshooting** only. For production deployments, change the admin password as soon as possible, or grant admin permissions to a real user account via the web UI or `plikd user update --admin` CLI command instead.

## Google OAuth

1. Create an application in the [Google Developer Console](https://console.developers.google.com)
2. Get your Client ID and Client Secret
3. Whitelist the redirect URL: `https://yourdomain/auth/google/callback`

```toml
GoogleApiClientID = "your-client-id"
GoogleApiSecret = "your-client-secret"
GoogleValidDomains = ["company.com"]  # Optional: restrict to email domains
```

## GitHub

1. Create an OAuth App in [GitHub Developer Settings](https://github.com/settings/developers)
2. Set the callback URL to `https://yourdomain/auth/github/callback`

```toml
GitHubApiClientID = "your-client-id"
GitHubApiSecret = "your-client-secret"
GitHubValidOrganizations = ["myorg"]  # Optional: restrict to org members
```

## OVH

1. Create an application at [OVH API](https://eu.api.ovh.com/createApp/)
2. Get your Application Key and Secret

```toml
OvhApiKey = "your-app-key"
OvhApiSecret = "your-app-secret"
OvhApiEndpoint = "https://eu.api.ovh.com/1.0"  # Optional, defaults to EU
```

## OpenID Connect (OIDC)

Works with any OIDC provider (Keycloak, Authentik, Dex, etc.).

1. Create a client application in your OIDC provider
2. Set redirect URI to `https://yourdomain/auth/oidc/callback`

```toml
OIDCClientID = "plik"
OIDCClientSecret = "your-secret"
OIDCProviderURL = "https://keycloak.example.com/realms/myrealm"
OIDCProviderName = "Keycloak"           # Optional: login button label
OIDCValidDomains = ["company.com"]      # Optional: restrict by email domain
OIDCRequireVerifiedEmail = true         # Optional: reject unverified emails (default: false)
```

## Session Settings

```toml
SessionTimeout = 3600  # Session cookie lifetime in seconds (default: 1h)
```

## Disabling Local Login

When using an external provider exclusively:

```toml
FeatureLocalLogin = "disabled"
```

This hides the login/password form and rejects local login attempts.

## CLI Tokens

Authenticated users can generate CLI tokens to authenticate the CLI client, linking uploads to the user's account for quota tracking and management.

### Browser login (recommended)

The easiest way to authenticate the CLI is via the built-in device auth flow:

```bash
plik --login
```

This opens your browser, lets you approve the login, and automatically saves the token to `~/.plikrc`. See the [CLI client documentation](/features/cli-client#cli-authentication) for details.

### Manual token

You can also create tokens manually in the web UI or via the API. Tokens are sent via the `X-PlikToken` header or configured in `.plikrc`:

```toml
Token = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```
