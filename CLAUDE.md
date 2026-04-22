# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Start infrastructure (MySQL + Redis)
docker-compose -f docker/docker-compose.yml up -d

# Run components
go run cmd/api/main.go         # HTTP API on :8080
go run cmd/worker/main.go      # Background job processor
go run cmd/devsecops/main.go scan [--fail-on-high]  # Local CLI scan

# Test
go test ./...
go test ./internal/scanner/...  # single package
```

## Architecture

Three binaries share the same `internal/` and `pkg/` packages:

```
cmd/api/        — Gin HTTP server: accepts scan job requests, stores to MySQL, returns results
cmd/worker/     — Polls MySQL for pending jobs, dispatches to scanners, writes ScanResult rows
cmd/devsecops/  — CLI that runs scanners locally and writes JSON reports to disk
```

**Data flow (distributed mode):** REST client → API → MySQL (`scan_jobs`) → Worker polls → Scanner runs → MySQL (`scan_results`) → API serves results.

**Data flow (CLI mode):** `devsecops scan` → Scanner → `internal/report` → JSON/Markdown file in `reports/`.

### Key packages

| Path | Role |
|------|------|
| `pkg/common` | `Config` loaded from env vars; `Finding` struct shared across all packages |
| `internal/store` | GORM models (`ScanJob`, `ScanResult`) and repository layer |
| `internal/job` | Job lifecycle: creation, worker poll loop, result persistence |
| `internal/scanner/sast` | Mock SAST scanner using regex; template for adding real scanners |
| `internal/report` | Finding aggregator (deduplication + risk scoring) and JSON/Markdown writers |

### Configuration (environment variables)

All env vars have defaults matching the docker-compose credentials. Key ones:

| Var | Default |
|-----|---------|
| `HTTP_ADDR` | `:8080` |
| `APP_ENV` | `development` |
| `DB_HOST / DB_USER / DB_PASSWORD / DB_NAME` | `127.0.0.1 / dev / dev123 / devsecops` |
| `WORKER_POLL_INTERVAL_SEC` | `5` |
| `REPORT_DIR` | `<project_root>/reports` |

### Adding a new scanner

1. Create `internal/scanner/<type>/` with a struct implementing the scanner interface used by `internal/job`.
2. Register it in the worker dispatch logic in `internal/job`.
3. Use `pkg/common.Finding` as the output type and follow the hash-based deduplication pattern in the existing mock SAST scanner.

## Notes

- `AGENTS.md` references `/ai/context.md` and `/ai/rules.md` — these files do not yet exist in the repo.
- Redis is defined in docker-compose but not yet wired into application logic.
- The SAST scanner (`internal/scanner/sast/mock.go`) is a regex-based mock; it is the reference implementation for real scanners.
