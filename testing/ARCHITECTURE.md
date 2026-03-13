# Architecture — Testing (`testing/`)

> Docker-based backend integration tests. For system-wide overview, see the root [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Structure

```
testing/
├── test_backends.sh    ← orchestrator: runs all or specific backend tests
├── utils.sh            ← shared helpers (docker, server start/stop, test assertions)
├── mariadb/            ← MariaDB metadata backend test
├── mysql/              ← MySQL metadata backend test
├── postgres/           ← PostgreSQL metadata backend test
├── mssql/              ← MS SQL Server (disabled — reserved keyword issue)
├── minio/              ← S3-compatible data backend test (MinIO)
├── swift/              ← OpenStack Swift data backend test
└── keycloak/           ← OIDC authentication test (Keycloak) — see [keycloak/ARCHITECTURE.md](keycloak/ARCHITECTURE.md)
```

---

## How It Works

Each backend directory contains:
- `run.sh` — starts/stops docker container, runs tests
- `plikd.cfg` — server config pointing at the dockerized backend

### Running Tests

```bash
make test-backends                        # All backends
testing/test_backends.sh postgres         # Specific backend
testing/test_backends.sh postgres test_name  # Specific test
DOCKER_VERSION="XXX" testing/test_backends.sh minio  # Specific docker image version
```

### Test Flow

1. `run.sh start` — spin up docker container for the backend
2. Build and start `plikd` with the backend's `plikd.cfg`
3. Run the Go e2e test suite from `plik/` against the live server (via `PLIKD_CONFIG` env var)
4. `run.sh stop` — tear down docker container

The actual test code lives in `plik/z*_e2e_*_test.go`. The `testing/` scripts provide the Docker infrastructure to run those same tests against real backends instead of the default in-memory backend. See [plik/ARCHITECTURE.md — E2E Test Suite](../plik/ARCHITECTURE.md#e2e-test-suite) for details on the test infrastructure, server bootstrapping, and what each test file covers.

### Per-Backend Storage-Level Tests

Each cloud data backend also has integration tests in its own package (`server/data/s3/`, `server/data/gcs/`, `server/data/swift/`) that verify upload, download, and deletion against real storage. These tests use `t.Skip` when `PLIKD_CONFIG` is not set or doesn't point to the matching backend.

They run automatically:
- During `test_backends.sh` — S3 and Swift tests are executed alongside the e2e suite
- During `make test` — they compile but skip (no real backend available)

To run manually against a specific backend:

```bash
PLIKD_CONFIG=$PWD/testing/minio/plikd.cfg go test ./server/data/s3/... -v -race
```

### Backend Coverage

| Backend | Tests | Type |
|---------|-------|------|
| MariaDB | Metadata CRUD, migrations | Metadata |
| MySQL | Metadata CRUD, migrations | Metadata |
| PostgreSQL | Metadata CRUD, migrations | Metadata |
| MinIO | File upload/download/delete (S3 API) | Data |
| Swift | File upload/download/delete (Swift API) | Data |
| Keycloak | OIDC login/callback flow | Auth |
