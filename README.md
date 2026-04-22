# DevSecOps Platform

A lightweight DevSecOps scanning platform built in Go. It exposes an HTTP API for asynchronous scan jobs, a worker for background execution, a shared scanner framework, and a reporting pipeline that produces persisted findings plus CI-friendly artifacts.

The current implementation is database-driven: the API writes jobs to MySQL, the worker polls MySQL for pending work, scanners run inside the worker process, and reports are written to disk and indexed in the database. Redis is provisioned in `docker/docker-compose.yml`, but it is not yet wired into the runtime queue path.

## Project Overview

This repository contains three binaries:

- `cmd/api`: Gin-based HTTP API for job submission and retrieval
- `cmd/worker`: background worker that claims and executes jobs
- `cmd/devsecops`: local CLI scanner with JSON and SARIF-like output

Shared platform logic lives under:

- `internal/job`: job lifecycle, validation, retries, timeouts, status transitions
- `internal/scanner`: scanner orchestration and scanner modules
- `internal/report`: aggregation and report generation
- `internal/store`: GORM models and database bootstrap
- `pkg/common`: configuration, logging, HTTP helpers, shared finding model

## Architecture Diagram

```text
Client / CI
   |
   v
API Service (Gin)
   |
   v
MySQL
  - scan_jobs
  - scan_results
  - scan_reports
   |
   v
Worker Service (polling loop)
   |
   +--> retry / timeout / panic recovery
   |
   v
Scanner Runner
   +--> sast
   +--> sca
   +--> secret
   +--> dast
   |
   v
Aggregation
  - deduplication
  - severity counts
  - risk score
   |
   +--> persist findings to MySQL
   +--> write report to disk
   +--> persist report metadata to MySQL
   |
   v
Final status
  - success
  - failed
  - blocked

Redis (provisioned, not actively used in current runtime flow)
```

## Core Features

- Asynchronous scan jobs with API submission and worker execution
- Shared scanner orchestration for `sast`, `sca`, `secret`, and `dast`
- Job timeout control and retry handling with max attempts of `3`
- Panic recovery and status transition enforcement in the worker path
- Blocking policy for `high` and `critical` findings
- Markdown report persistence for API retrieval
- JSON and SARIF-like report generation for CLI and CI workflows
- Structured logging with `job_id`, scanner durations, risk summaries, and total scan duration
- Simple in-process metrics counters for `jobs_total`, `jobs_failed`, and `jobs_blocked`
- Input validation and path-hardening around repository and scanner input

## Current Architecture Notes

- The effective queue is MySQL row state (`pending -> running -> success|failed|blocked`)
- Redis is provisioned but not used as a live queue or broker
- The `sast` module contains the active mock implementation
- `sca`, `secret`, and `dast` are currently stubs behind the same scanner contract
- The worker does not clone repositories today
  - local repository paths are scanned directly
  - non-local repository references are accepted and sanitized, but the mock SAST scanner falls back to an informational finding instead of performing a checkout

## How to Run

### 1. Start infrastructure

```bash
docker-compose -f docker/docker-compose.yml up -d
```

This starts:

- MySQL on `127.0.0.1:3306`
- Redis on `127.0.0.1:6379`

### 2. Run the API

```bash
go run cmd/api/main.go
```

Default API address:

- `:8080`

### 3. Run the worker

```bash
go run cmd/worker/main.go
```

### 4. Run the local CLI scanner

```bash
go run cmd/devsecops/main.go scan
```

Fail on blocking findings:

```bash
go run cmd/devsecops/main.go scan --fail-on-high
```

Simulate CI behavior:

```bash
go run cmd/devsecops/main.go run --ci-mode
```

### 5. Run tests

```bash
go test ./...
```

## Configuration

Key environment variables:

| Variable | Default |
|---|---|
| `HTTP_ADDR` | `:8080` |
| `APP_ENV` | `development` |
| `GIN_MODE` | `release` |
| `LOG_LEVEL` | `info` |
| `WORKER_POLL_INTERVAL_SEC` | `5` |
| `REPORT_DIR` | `<project_root>/reports` |
| `DB_HOST` | `127.0.0.1` |
| `DB_PORT` | `3306` |
| `DB_USER` | `dev` |
| `DB_PASSWORD` | `dev123` |
| `DB_NAME` | `devsecops` |

## Example API Usage

### Create a job

```bash
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "repo_url": "C:/path/to/local/repo",
    "branch": "main",
    "scan_type": ["sast", "sca", "secret", "dast"],
    "block_on_high": true,
    "max_execution_time_sec": 300
  }'
```

Example response shape:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "job_id": 1,
    "status": "pending",
    "repo_url": "C:/path/to/local/repo",
    "branch": "main",
    "scan_type": ["sast", "sca", "secret", "dast"],
    "block_on_high": true,
    "attempt_count": 0,
    "max_execution_time_sec": 300
  }
}
```

### Get job status

```bash
curl http://localhost:8080/api/v1/jobs/1
```

### Get findings

```bash
curl "http://localhost:8080/api/v1/jobs/1/results?page=1&page_size=20"
```

### Get Markdown report

```bash
curl http://localhost:8080/api/v1/jobs/1/report
```

## Data Flow

1. A client submits a job to `POST /api/v1/jobs`.
2. The API validates and sanitizes repository input, branch, scan types, and timeout settings.
3. The API persists a `scan_jobs` row with status `pending`.
4. The worker polls MySQL, claims the oldest pending job, and transitions it to `running`.
5. The worker executes the selected scanner modules through the shared scanner runner.
6. Raw findings are aggregated into deduplicated findings, severity counts, and total risk score.
7. The worker persists normalized findings into `scan_results`.
8. The worker writes a Markdown report to disk and persists summary metadata into `scan_reports`.
9. The worker evaluates blocking policy and finalizes the job as `success`, `failed`, or `blocked`.
10. The API serves job status, findings, and report content from persisted state.

## Blocking Logic

Blocking is evaluated after a scan completes successfully.

- If `block_on_high = false`, the job finishes as `success`
- If `block_on_high = true` and at least one `high` or `critical` finding exists, the job finishes as `blocked`
- If execution fails operationally, the job finishes as `failed`

This keeps policy failure (`blocked`) distinct from runtime failure (`failed`).

## Security Design Highlights

- Request validation at the API boundary for repository reference, branch, scan types, and execution timeout
- Repository URL sanitization and redaction before logging
- Scanner selection restricted to the fixed registry (`sast`, `sca`, `secret`, `dast`)
- Local repository path hardening in the mock SAST scanner
  - traversal-style input is rejected
  - symlinks are skipped
  - walked files must remain under the repository root
- No shell-based scanner dispatch in the shared runner
- Structured logs avoid leaking repository credentials and include job-scoped fields instead
- Worker execution is bounded by timeout, retries, and panic recovery to avoid stuck jobs

## Observability

The worker emits structured JSON logs including:

- `job_id`
- per-scanner execution duration
- total scan duration
- severity summary and risk score
- terminal status

It also maintains in-process counters for:

- `jobs_total`
- `jobs_failed`
- `jobs_blocked`

## Reports

Worker path:

- Markdown report written to `REPORT_DIR`
- report metadata persisted to `scan_reports`

CLI path:

- `cli-scan.json`
- `cli-scan.sarif.json`

These outputs are intended for local review and CI-style integration.
