# d3: Dump Data Depot

[![CI](https://github.com/zhulik/d3/actions/workflows/push.yaml/badge.svg?branch=main)](https://github.com/zhulik/d3/actions/workflows/push.yaml)

*Not to be confused with [d3: Data-Driven Documents](https://github.com/d3/d3)*

d3 is a lightweight S3-compatible object store aimed at small deployments—home labs, local development, and single-machine setups where you want something simple instead of running full cloud-style infrastructure.

## Goals

- **Simplicity** — easy to run, few moving parts, understandable behavior.
- **Transparency** — clear configuration, predictable storage layout, no hidden magic.

## Non-goals

- **Scalability** — not designed for massive throughput or huge object counts as a primary concern.
- **Multi-node / HA** — no clustered or replicated deployment model; think one process, one place for data.

## Requirements

- **Redis-compatible server** — d3 uses Redis (or Valkey) for coordination. Point it at your instance with `REDIS_ADDRESS`.

## Installation

### Docker

The repository includes a multi-stage `Dockerfile` that builds a static binary and runs it on a minimal image.

Build the image:

```bash
docker build -t d3 .
```

Run the container (adjust paths and ports as needed). You need a reachable Redis/Valkey instance. The example below sets `ENVIRONMENT=development` so ephemeral admin credentials are printed in the logs; for production, omit that and set `ADMIN_CREDENTIALS_PATH` to a credentials YAML file instead.

```bash
docker run --rm \
  -p 8080:8080 \
  -e ENVIRONMENT=development \
  -e REDIS_ADDRESS=host.docker.internal:6379 \
  -e FOLDER_STORAGE_BACKEND_PATH=/data \
  -e MANAGEMENT_BACKEND_YAML_PATH=/data/management.yaml \
  -e MANAGEMENT_BACKEND_TMP_PATH=/data/tmp \
  -v d3-data:/data \
  d3
```

For a full local stack (Valkey + d3), use Docker Compose from the repo root (the sample uses `ENVIRONMENT=development` for the same reason):

```bash
docker compose up --build
```

The sample compose file maps the S3 API to host port `9090` (`9090:8080`). Health and management ports are not exposed in that file by default; set `HEALTH_CHECK_PORT` / `MANAGEMENT_PORT` and publish them if you need them from the host.

### From source

Use the project’s [Task](https://taskfile.dev/) targets, for example `task build` and `task run`, or build with `go build -o d3 ./cmd/main.go`.

## Configuration

Configuration is read from environment variables (via [caarlos0/env](https://github.com/caarlos0/env)). Defaults suit local folder storage under `./d3_data`.

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `production` | Runtime environment label. In `development` or `test`, temporary admin credentials may be created automatically when no admin file is configured. |
| `STORAGE_BACKEND` | `folder` | Storage backend type. Currently only `folder` is supported. |
| `FOLDER_STORAGE_BACKEND_PATH` | `./d3_data` | Root directory for object data (folder backend). |
| `MANAGEMENT_BACKEND` | `YAML` | Management backend type. `YAML` is supported. |
| `MANAGEMENT_BACKEND_YAML_PATH` | `./d3_data/management.yaml` | Path to the YAML management state file. |
| `MANAGEMENT_BACKEND_TMP_PATH` | `./d3_data/tmp` | Temp directory for management operations. Should live on the same filesystem as main storage for atomic renames (YAML backend). |
| `ADMIN_CREDENTIALS_PATH` | *(empty)* | Path to a YAML file with admin credentials. If unset, `development` and `test` environments get ephemeral credentials (logged at startup); in `production` (default), admin credentials must be provided or startup fails. See [admin-credentials.dev.yaml](./admin-credentials.dev.yaml) for reference. |
| `REDIS_ADDRESS` | `localhost:6379` | Address of the Redis or Valkey server. |
| `PORT` | `8080` | HTTP port for the S3-compatible API. |
| `HEALTH_CHECK_PORT` | `8081` | Port for the health check HTTP server. |
| `MANAGEMENT_PORT` | `8082` | Port for the management HTTP API. |

## Kubernetes

A Helm chart and a Kubernetes operator are **not** available yet (TBD).

