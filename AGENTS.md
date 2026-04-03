# AGENTS.md

## Cursor Cloud specific instructions

### Overview

d3 is a lightweight S3-compatible object storage server written in Go. It uses Redis for distributed locking and stores data on the local filesystem.

### Prerequisites (installed via snapshot/setup)

- **Go 1.25.6** (system Go)
- **Redis** on `localhost:6379` — must be running before starting the server or running integration tests
- **Task** (taskfile.dev) v3.49.1 — task runner
- **golangci-lint** v2.11.4 — linter
- **Ginkgo** v2.28.1 — test runner (installed to `~/go/bin/`, ensure `PATH` includes it)

### Starting Redis

Redis must be running before the d3 server or integration tests. Start it with:

```bash
redis-server --daemonize yes
```

Verify with `redis-cli ping` (should return `PONG`).

### Development commands

All standard commands are in `Taskfile.yml` and `tasks/*.yml`:

| Command | Purpose |
|---------|---------|
| `task build` | Build all packages |
| `task run` | Run server with hot reload (air) |
| `task lint:lint` | Lint (no auto-fix) |
| `task lint` | Lint with `--fix` |
| `task test` | Unit tests (Ginkgo, excludes conformance) |
| `task test:integration` | Integration tests (conformance + management) |
| `task test:conformance` | S3 conformance tests only |
| `task test:management` | Management API tests only |

### Running the server manually

```bash
ENVIRONMENT=development ADMIN_CREDENTIALS_PATH=admin-credentials.dev.yaml go run ./cmd/d3-server
```

This starts the S3 API on `:8080`, health check on `:8081`, and management API on `:8082`. In `development` mode with `admin-credentials.dev.yaml`, the admin credentials are:
- Access Key: `AKIAIOSFODNN7EXAMPLE`
- Secret Key: `wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`

### Gotchas

- **Ginkgo binary path**: `ginkgo` is installed to `~/go/bin/`. The `PATH` must include this directory. This is set in `~/.bashrc`.
- **Integration tests require Redis**: They connect to `localhost:6379` (hardcoded in `integration/testhelpers/app.go`). Tests will fail immediately without Redis running.
- **Lint has pre-existing issues**: `task lint:lint` (without `--fix`) reports 2 pre-existing `nlreturn`/`wsl_v5` issues in `internal/apis/s3/api_objects_test.go`. These are not caused by new changes.
- **S3 API requires SigV4 signing**: Plain `curl` calls to port 8080 will get `403 Forbidden`. Use an S3 SDK client (Go AWS SDK, MinIO client, etc.) with the admin credentials above.
