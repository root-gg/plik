# AGENTS.md — Plik

> Entry point for AI agents working on this codebase. Not an exhaustive manual — follow pointers to scoped ARCHITECTURE.md files for deeper context.

> [!CAUTION]
> ## Mandatory Review Gate — No Exceptions
>
> **NEVER perform any git or GitHub write action without explicit user approval.** This includes:
> - `git commit`, `git push`, `git push --force-with-lease`
> - Creating branches, pull requests, issues, or comments on GitHub
> - Submitting PR reviews, merging PRs
>
> **Required process for EVERY commit/push (use the `/commit` workflow):**
> 1. `git add -A && git diff --cached --stat` — show the diff summary to the user
> 2. Propose a commit message — wait for approval
> 3. `git commit` — only after user approves both diff and message
> 4. Ask before pushing — the user must explicitly say "push" or "go ahead"
>
> This applies equally to trivial one-line changes and large refactors. There are zero exceptions.

## What is Plik?

Plik is a temporary file upload system (WeTransfer-like) written in Go, with a Vue 3 web UI and a cross-platform CLI client. It supports multiple storage and metadata backends, authentication providers, and features like one-shot downloads, streaming, end-to-end encryption (E2EE via age), and server-side encryption.

## Tech Stack

| Layer     | Tech |
|-----------|------|
| Server    | Go, gorilla/mux, GORM |
| Webapp    | Vue 3, Vite, Tailwind CSS, CodeMirror 6 |
| CLI       | Go, docopt-go |
| Config    | TOML (server), TOML (client `.plikrc`) |
| Data      | File, OpenStack Swift, S3, Google Cloud Storage |
| Metadata  | SQLite3, PostgreSQL, MySQL (via GORM) |
| CI        | GitHub Actions (tests, docker build/deploy on PR comment, release, Helm chart publish) |

## Repo Layout

```
plik/
├── AGENTS.md              ← you are here
├── ARCHITECTURE.md         ← system-wide architecture
├── README.md               ← project README (concise)
├── Makefile                ← build orchestration
├── Dockerfile
├── .agent/                 ← agentic workflows (/review-changes, /prepare-pr, /cut-release)
├── server/                 ← Go server (see server/ARCHITECTURE.md)
│   ├── main.go             ← entry point
│   ├── plikd.cfg           ← default config
│   ├── cmd/                ← CLI commands (cobra)
│   ├── common/             ← shared types, config, feature flags
│   ├── context/            ← custom request context (predates Go stdlib context)
│   ├── data/               ← data backend interface + implementations
│   ├── handlers/           ← HTTP handlers
│   ├── metadata/           ← metadata backend (GORM)
│   ├── middleware/          ← middleware chain (auth, logging, upload/file resolution, CORS, download domain restriction)
│   └── server/             ← HTTP server + router setup
├── client/                 ← CLI client + MCP server (see client/ARCHITECTURE.md)
├── plik/                   ← Go client library (see plik/ARCHITECTURE.md)
├── webapp/                 ← Vue 3 SPA, i18n via vue-i18n (see webapp/ARCHITECTURE.md)
├── testing/                ← backend integration tests (see testing/ARCHITECTURE.md)
├── charts/                 ← Helm chart for Kubernetes deployment
├── .github/                ← GitHub Actions workflows (see .github/ARCHITECTURE.md)
├── changelog/              ← release changelogs
├── releaser/               ← release build scripts
├── docs/                   ← VitePress documentation site
└── vendor/                 ← Go vendored dependencies
```

## Build & Run

```bash
make                        # Build everything (frontend + clients + server)
make server                 # Build server only → server/plikd
make client                 # Build CLI client only → client/plik
make frontend               # Build Vue webapp → webapp/dist
make docker                 # Build Docker image (rootgg/plik:dev)
make helm                   # Package Helm chart locally (dry-run)
make helm-install           # Package and install Helm chart locally
cd server && ./plikd        # Run server on http://127.0.0.1:8080
```

#### Pull Request Deployments (GitHub Actions)
- `docker build` (comment on PR): Builds and pushes image `rootgg/plik:pr-{PR_NUMBER}`
- `docker deploy` (comment on PR): Deploy PR image to `plik.root.gg` (requires secrets)

## Test

```bash
make test                   # Unit tests + CLI integration tests
make test-frontend           # Webapp unit tests (vitest)
make test-frontend-e2e       # Webapp e2e tests (playwright — builds + starts fresh plikd)
make test-backends           # Docker-based backend integration tests (all)
make test-backend mariadb    # Docker-based test for a single backend
make lint                   # go fmt + go vet + go fix
make gofix                  # Run go fix
make vuln                   # govulncheck (report only)
```

## Key Files

| File | Purpose |
|------|---------|
| `server/plikd.cfg` | Server configuration (TOML) — all options with comments |
| `client/.plikrc` | CLI client configuration template |
| `Makefile` | Build targets for server, client, frontend, docker, release |
| `server/common/config.go` | Config struct + parsing + env var override logic |
| `server/common/file.go` | File model + status constants |
| `server/common/upload.go` | Upload model |
| `server/common/token.go` | Token model + prefixed opaque token generation (`plik_` + Base62 + CRC32 checksum) |
| `server/metadata/upload.go` | Upload queries, `UploadFilters` struct for filtering by user/token/badge settings |
| `server/handlers/misc.go` | Shared helpers: `parseBoolFilter`, `parseBadgeFilters` (badge filter struct from request) |
| `webapp/src/components/UploadControls.vue` | Shared sort/order/badge-filter control bar used by AdminView and HomeView |
| `server/common/feature_flags.go` | Feature flag types (`disabled`/`enabled`/`default`/`forced`) |
| `server/server/server.go` | `ensureDefaultAdmin()` — idempotent bootstrap of `DefaultAdminLogin`/`DefaultAdminPassword` on startup |

## Conventions

- **Configuration**: TOML file + env var override using SCREAMING_SNAKE_CASE (e.g., `PLIKD_DEBUG_REQUESTS=true`)
- **Feature flags**: Four states — `disabled`, `enabled` (opt-in), `default` (opt-out), `forced`
- **Special values**: `0` = use server default, `-1` = unlimited (for file size, TTL, etc.)
- **Error handling**: Handlers return HTTP errors; middleware chain panics on missing required context values
- **ID generation**: Random hex strings (16 chars for files, 16 chars for uploads); CLI tokens use prefixed opaque format (`plik_` + 30 Base62 + 6 CRC32, 41 chars total)
- **Backend interface**: `data.Backend` is the storage abstraction; implementations are swappable via config
- **Archive compression**: Enabled by default (`EnableArchiveCompression = true`). Can be disabled to `zip.Store` (no compression) on public instances to prevent CPU exhaustion DoS.
- **Default admin provisioning**: `DefaultAdminLogin` / `DefaultAdminPassword` config options (+ `PLIKD_DEFAULT_ADMIN_LOGIN` / `PLIKD_DEFAULT_ADMIN_PASSWORD` env vars) create a local admin user on first startup via `ensureDefaultAdmin()`. Idempotent — skipped if the user already exists. Intended for bootstrap only; remove from config once a real admin account exists.

## Best Practices

- **Always update docs**: When changing code, update the relevant `ARCHITECTURE.md` and VitePress docs
- **Keep Helm chart in sync with plikd config**: When adding, removing, or renaming configuration fields in `server/common/config.go` or `server/plikd.cfg`, you **must** also update the Helm chart:
  - `charts/plik/values.yaml` — add/update the field under `plikd:` (non-sensitive) or `secrets:` (sensitive)
  - `charts/plik/templates/configmap.yaml` — add/update the explicit key in the template (non-sensitive config only; never put secrets here)
  - `charts/plik/templates/secret.yaml` — if the field is a credential, add it under `secrets:` in `values.yaml` and a corresponding key in `secret.yaml`
- **Helm secrets pattern**: All sensitive credentials must live in the `secrets:` top-level block of `values.yaml`. They are rendered into a `Secret` resource by `secret.yaml`, and injected into the pod via `envFrom.secretRef` (`optional: true`). Never put secrets in the ConfigMap.
- **BYO Secret (existingSecret)**: Set `secrets.existingSecret: "my-secret-name"` to skip Secret creation and reference an external secret (e.g., Vault, Sealed Secrets, ESO). Use the `plik.secretName` helper in templates to resolve the correct name.
- **Helm persistence**: the chart has two independent PVCs — `persistence` for uploaded files (`/home/plik/server/files`) and `dbPersistence` for the SQLite database (`/home/plik/server/db`). Both default to `emptyDir` when disabled. The default `MetadataBackendConfig.ConnectionString` is `/home/plik/server/db/plik.db`.
- **Run tests before committing**: `make lint && make test`
- **Keep ARCHITECTURE.md files in sync**: Each root folder has its own — update the one closest to your change
- **Release process**: Before creating a GitHub release, update the version in `README.md` and move `charts/plik/CHANGELOG.md` entries from `[Unreleased]` to the new version heading

## Documentation

The documentation lives in two places:

1. **For agents**: Scoped `ARCHITECTURE.md` files in each root folder
2. **For humans**: VitePress site in `docs/` — preview locally with `cd docs && npm run dev`

### Updating docs

```bash
cd docs && npm install       # First time only
cd docs && npm run dev       # Preview at localhost:5173
make docs                    # Build docs (builds client, injects help+plikrc, builds VitePress)
```

**Important**: Always run `make docs` when you touch documentation files to catch build errors (dead links, etc.) before committing.

## Scoped Architecture Docs

| File | Scope |
|------|-------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System-wide: package layering, data flow, API, auth, config |
| [server/ARCHITECTURE.md](server/ARCHITECTURE.md) | Server internals: packages, middleware chain, handlers |
| [client/ARCHITECTURE.md](client/ARCHITECTURE.md) | CLI client: commands, config, archive/crypto |
| [plik/ARCHITECTURE.md](plik/ARCHITECTURE.md) | Go library: public API, types, test harness |
| [webapp/ARCHITECTURE.md](webapp/ARCHITECTURE.md) | Vue 3 SPA: components, routing, API layer, state |
| [testing/ARCHITECTURE.md](testing/ARCHITECTURE.md) | Backend integration tests: docker-based test scripts |
| [releaser/ARCHITECTURE.md](releaser/ARCHITECTURE.md) | Release tooling: build pipeline, Docker stages, client/server compilation |
| [charts/plik/ARCHITECTURE.md](charts/plik/ARCHITECTURE.md) | Helm chart: structure, config/secrets separation, persistence, versioning |
| [.github/ARCHITECTURE.md](.github/ARCHITECTURE.md) | GitHub Actions workflows, CI/CD, Helm chart release flow |

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **plik** (4386 symbols, 18388 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## When Debugging

1. `gitnexus_query({query: "<error or symptom>"})` — find execution flows related to the issue
2. `gitnexus_context({name: "<suspect function>"})` — see all callers, callees, and process participation
3. `READ gitnexus://repo/plik/process/{processName}` — trace the full execution flow step by step
4. For regressions: `gitnexus_detect_changes({scope: "compare", base_ref: "main"})` — see what your branch changed

## When Refactoring

- **Renaming**: MUST use `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` first. Review the preview — graph edits are safe, text_search edits need manual review. Then run with `dry_run: false`.
- **Extracting/Splitting**: MUST run `gitnexus_context({name: "target"})` to see all incoming/outgoing refs, then `gitnexus_impact({target: "target", direction: "upstream"})` to find all external callers before moving code.
- After any refactor: run `gitnexus_detect_changes({scope: "all"})` to verify only expected files changed.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Tools Quick Reference

| Tool | When to use | Command |
|------|-------------|---------|
| `query` | Find code by concept | `gitnexus_query({query: "auth validation"})` |
| `context` | 360-degree view of one symbol | `gitnexus_context({name: "validateUser"})` |
| `impact` | Blast radius before editing | `gitnexus_impact({target: "X", direction: "upstream"})` |
| `detect_changes` | Pre-commit scope check | `gitnexus_detect_changes({scope: "staged"})` |
| `rename` | Safe multi-file rename | `gitnexus_rename({symbol_name: "old", new_name: "new", dry_run: true})` |
| `cypher` | Custom graph queries | `gitnexus_cypher({query: "MATCH ..."})` |

## Impact Risk Levels

| Depth | Meaning | Action |
|-------|---------|--------|
| d=1 | WILL BREAK — direct callers/importers | MUST update these |
| d=2 | LIKELY AFFECTED — indirect deps | Should test |
| d=3 | MAY NEED TESTING — transitive | Test if critical path |

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/plik/context` | Codebase overview, check index freshness |
| `gitnexus://repo/plik/clusters` | All functional areas |
| `gitnexus://repo/plik/processes` | All execution flows |
| `gitnexus://repo/plik/process/{name}` | Step-by-step execution trace |

## Self-Check Before Finishing

Before completing any code modification task, verify:
1. `gitnexus_impact` was run for all modified symbols
2. No HIGH/CRITICAL risk warnings were ignored
3. `gitnexus_detect_changes()` confirms changes match expected scope
4. All d=1 (WILL BREAK) dependents were updated

## Keeping the Index Fresh

After committing code changes, the GitNexus index becomes stale. Re-run analyze to update it:

```bash
npx gitnexus analyze
```

If the index previously included embeddings, preserve them by adding `--embeddings`:

```bash
npx gitnexus analyze --embeddings
```

To check whether embeddings exist, inspect `.gitnexus/meta.json` — the `stats.embeddings` field shows the count (0 means no embeddings). **Running analyze without `--embeddings` will delete any previously generated embeddings.**

> Claude Code users: A PostToolUse hook handles this automatically after `git commit` and `git merge`.

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
