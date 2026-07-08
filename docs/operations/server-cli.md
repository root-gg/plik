# Server CLI

The `plikd` binary is both the Plik server and an admin CLI for managing users, tokens, uploads, and maintenance tasks.

::: tip
All `plikd` commands load configuration using the same search order:
`--config` flag → `PLIKD_CONFIG` env → `./plikd.cfg` → `/etc/plikd.cfg`
:::

## User Management

Manage user accounts (local, Google, OVH, OIDC providers).

### Create a user

```bash
# Local user with password
plikd user create --login admin --password s3cret123 --admin

# OAuth provider user
plikd user create --provider google --login user@gmail.com --name "John Doe"

# With size and TTL limits
plikd user create --login bob --max-file-size 100MB --max-user-size 1GB --max-ttl 7d
```

| Flag | Description |
|------|-------------|
| `--provider` | Auth provider: `local` (default), `google`, `ovh`, `oidc` |
| `--login` | User login (min 4 chars) |
| `--password` | Password for local users (min 8 chars, auto-generated if omitted) |
| `--name` | Display name |
| `--email` | Email address |
| `--admin` | Grant admin privileges |
| `--max-file-size` | Per-file size limit (e.g. `100MB`, `-1` for unlimited) |
| `--max-user-size` | Total storage limit (e.g. `1GB`, `-1` for unlimited) |
| `--max-ttl` | Maximum upload TTL (e.g. `7d`, `24h`) |

### List users

```bash
plikd user list
```

::: tip OIDC users
OIDC users are identified by their `sub` claim, not by their username. `plikd user list` prints the real ID (e.g. `oidc:8f9d056a-1dab-3192-f1ae-552e024d948e`) followed by the username — pass the full ID part after `oidc:` as `--login` to `show`, `update`, `delete` and `token` commands:

```bash
plikd user update --provider oidc --login 8f9d056a-1dab-3192-f1ae-552e024d948e --admin
```
:::

### Show user details

```bash
plikd user show --login admin
plikd user show --provider google --login user@gmail.com
```

### Update a user

```bash
plikd user update --login admin --admin
plikd user update --login bob --max-file-size 500MB --max-ttl 14d
```

Only the specified flags are changed — all other fields are preserved.

### Delete a user

```bash
plikd user delete --login admin
plikd user delete --provider google --login user@gmail.com
```

::: warning
Deleting a user also removes **all their uploads and files** from both the metadata and data backends.
:::

## Token Management

Manage API tokens for authenticated uploads.

### Create a token

```bash
plikd token create --login admin --comment "CI/CD pipeline"
```

| Flag | Description |
|------|-------------|
| `--provider` | Auth provider (default: `local`) |
| `--login` | User login to create the token for |
| `--comment` | Token description |

### List tokens

```bash
plikd token list
```

### Delete a token

```bash
plikd token delete --token xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## File & Upload Management

Manage uploads and individual files in the system.

### List files

```bash
# List all files
plikd file list

# List files in a specific upload
plikd file list --upload abc123

# Show a specific file
plikd file list --file def456

# Machine-readable sizes (bytes)
plikd file list --human=false
```

### Show file details

```bash
plikd file show --file abc123
```

Displays full file metadata, upload URL, and direct download URL.

### Delete

You must specify exactly one of `--file`, `--upload`, or `--all`:

```bash
# Delete a single file
plikd file delete --file abc123

# Delete an entire upload
plikd file delete --upload def456

# Delete ALL uploads (requires confirmation)
plikd file delete --all
```

::: danger
`--all` removes **every upload** in the system. A confirmation prompt is always shown.
:::

## Cleanup

Remove expired uploads, purge deleted files from the data backend, and prune old download statistics used for trends:

```bash
plikd clean
```

This runs the same cleanup routine that the server executes periodically when running. Use it for manual maintenance or cron jobs.
