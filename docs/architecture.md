# DevSecOps Platform Architecture

## Overview
This repository implements a database-driven DevSecOps scanning platform with three active runtime roles:

- API service for job submission and result retrieval
- Worker service for asynchronous scan execution
- Shared scanner and reporting packages used by both services

The current execution model is MySQL-centered. Jobs are persisted in the database, the worker polls for pending work, scanner output is aggregated into findings and risk summaries, and reports are written to disk plus indexed in MySQL. Redis is provisioned in infrastructure but is not part of the active control path in the current codebase.

## Components

### API Layer
The API layer is a Gin HTTP service under `cmd/api/`. It exposes:

- `POST /api/v1/jobs`
- `GET /api/v1/jobs/:id`
- `GET /api/v1/jobs/:id/results`
- `GET /api/v1/jobs/:id/report`

Its responsibilities are:

- validate and sanitize job input
- create `scan_jobs` records
- return job lifecycle state
- return persisted findings from `scan_results`
- return report content from the report system

The API does not execute scanners directly.

### Worker Layer
The worker under `cmd/worker/` is the asynchronous execution engine. It continuously polls MySQL for the oldest `pending` job, atomically claims it, and transitions it to `running`.

The worker is responsible for:

- timeout control per job
- retry handling with up to 3 attempts
- panic recovery during execution
- scanner orchestration
- aggregation of findings and risk scores
- writing reports
- persisting final job state as `success`, `failed`, or `blocked`

### Scanner Modules
Scanner orchestration is implemented in `internal/scanner/`. The runner dispatches to scanner modules based on `scan_type`:

- `sast`
- `sca`
- `secret`
- `dast`

`sast` currently contains the concrete mock implementation. The other modules are stubs but already participate in the orchestration contract. The worker invokes scanners through a shared runner, which also records per-scanner execution time for observability.

### Redis Queue
Redis is defined in `docker/docker-compose.yml`, but it is not currently used as a live queue or broker.

Current behavior:

- Redis is provisioned as optional infrastructure
- job dispatch is not Redis-backed
- queue semantics are currently implemented by MySQL row state plus worker polling

So Redis is part of the intended platform shape, but not the active runtime path today.

### Database Layer
The database layer uses GORM with MySQL. It persists three main record types:

- `scan_jobs`: job metadata, status, timeout, attempts, timestamps
- `scan_results`: normalized finding rows
- `scan_reports`: report path, severity summary, risk score

MySQL serves two roles:

- system of record for all job lifecycle and scan outputs
- effective work queue through `pending`/`running` state transitions

### Report System
The report system under `internal/report/` handles:

- finding aggregation and deduplication
- severity counting
- total risk scoring
- Markdown report generation for API retrieval
- JSON and SARIF-like report generation for CLI output

In the worker path, the report system writes a Markdown file to `REPORT_DIR` and stores summary metadata plus the report path in `scan_reports`.

## End-to-End Data Flow

### 1. Job Creation
A client submits a scan request to `POST /api/v1/jobs` with:

- repository reference
- branch
- scan types
- `block_on_high`
- optional max execution time

The API validates and sanitizes the request, then creates a `scan_jobs` record with status `pending`.

### 2. Persistence
The new job is stored in MySQL. At this point MySQL acts as the authoritative queue.

### 3. Worker Polling
The worker loop polls for the next `pending` job. When it claims one successfully, it updates the row to:

- `status = running`
- `started_at = now`

The worker also enforces retry and timeout policy around execution.

### 4. Scan Execution
The worker invokes the shared scanner runner, which dispatches to the requested scanner modules:

- `sast`
- `sca`
- `secret`
- `dast`

Each scanner returns findings in a common format. Execution is bounded by per-job timeout and wrapped in panic recovery.

### 5. Aggregation
The worker aggregates raw findings into a single result set:

- deduplicated findings
- severity counts
- total risk score

This aggregation result is the basis for persistence, reporting, and blocking decisions.

### 6. Persistence of Results and Reports
After aggregation, the worker persists:

- finding rows into `scan_results`
- a Markdown report file on disk
- a `scan_reports` record containing report path, summary JSON, and risk score

### 7. Final Status Decision
The worker transitions the job out of `running` into one terminal state:

- `success`
- `failed`
- `blocked`

### 8. Retrieval
Clients retrieve outputs through the API:

- `/jobs/:id` for job status
- `/jobs/:id/results` for findings
- `/jobs/:id/report` for report content

## Blocking Logic
Blocking is decided after scan execution and aggregation complete.

- If `block_on_high = false`, a completed scan becomes `success`
- If `block_on_high = true` and any finding is `high` or `critical`, the final status becomes `blocked`
- If execution fails due to timeout, retry exhaustion, or runtime failure, the final status becomes `failed`

This means blocking is a policy decision applied to a successfully completed scan, while `failed` represents operational failure.

## Queueing Model Summary
The intended platform shape includes Redis, but the current implementation is:

- API writes jobs to MySQL
- worker polls MySQL
- worker executes scanners
- worker writes findings and reports
- API reads results back from MySQL and report storage

So the active architecture is a polling worker over MySQL, with Redis present but inactive.
