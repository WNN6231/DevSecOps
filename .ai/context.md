
---

````markdown
# AI Context: DevSecOps Security Gate Platform

## 1. Project Identity (不可修改)

This is a DevSecOps Security Gate platform.

### Core purpose:
- Orchestrate multiple security scanners
- Aggregate results
- Enforce blocking on high severity findings

### This project is NOT:
- A full-featured vulnerability scanner
- A UI-heavy system
- A multi-language SAST platform (current scope: Go only)

---

## 2. MVP Scope (强约束)

ONLY implement these capabilities:

1. Create scan job
2. Execute scanners
3. Aggregate results
4. Generate report + block high risk

### DO NOT:
- Add authentication system
- Add frontend UI
- Add microservices split
- Add Kubernetes support

---

## 3. System Architecture (必须遵守)

### Components:

- API Service (Gin)
- Worker (goroutine pool)
- Scanner Modules
- Result Aggregator

### Flow:

1. API creates job
2. Job enters queue (Redis) [Phase 3]
3. Worker pulls job
4. Worker runs scanners
5. Results normalized
6. Aggregator builds report
7. Apply blocking rules

---

## 4. Core Data Structures (不可擅自修改)

### Finding (统一结构)

```go
type Finding struct {
    Scanner        string
    Severity       string
    RuleID         string
    Title          string
    Description    string
    FilePath       string
    LineNumber     int
    Evidence       string
    Recommendation string
    Hash           string
}
````

**ALL scanners MUST output this format.**

---

## 5. Severity Model (强约束)

### Allowed values:

* critical
* high
* medium
* low
* info

### Blocking rule:

* If (critical OR high) AND block_on_high=true → job.status = blocked

---

## 6. Scanner Constraints (非常关键)

### SAST

* Language: Go only
* Rule-based (NOT AST-heavy initially)

Focus:

* Hardcoded secrets
* SQL injection patterns
* Command execution risks

---

### SCA

* Input: go.mod / go.sum

Detect:

* Known vulnerable dependencies
* Outdated packages

---

### Secret Scan

* Regex-based detection

Detect:

* API keys
* tokens
* passwords
* private keys

---

### DAST (minimal version)

* Only HTTP requests
* No crawler
* No headless browser

Check:

* Security headers
* Basic reflection
* Sensitive endpoints

---

## 7. Code Organization (必须遵守)

```
cmd/api
cmd/worker

internal/job
internal/scanner/sast
internal/scanner/sca
internal/scanner/secret
internal/scanner/dast
internal/report
internal/store

pkg/common
```

**DO NOT restructure this.**

---

## 8. Development Rules (防止 Codex乱写)

* Always implement minimal working version first
* No premature abstraction
* No unnecessary interfaces
* Prefer simple structs
* Avoid over-engineering
* Keep functions small and explicit
* No speculative features

---

## 9. Current Development Phase

### Phase: 1 (Single-node MVP)

#### Allowed:

* Synchronous execution
* Local repository cloning
* Single SAST rule
* Local file-based report output

#### Not allowed:

* Redis
* Distributed worker
* Message queue
* CI integration

---

## 10. API Design Constraints

### Create Job

POST `/api/v1/jobs`

#### Request:

```json
{
  "repo_url": "https://github.com/example/demo.git",
  "branch": "main",
  "scan_type": ["sast"],
  "block_on_high": true
}
```

#### Response:

```json
{
  "job_id": 1001,
  "status": "pending"
}
```

---

### Get Job Status

GET `/api/v1/jobs/:id`

---

### Get Results

GET `/api/v1/jobs/:id/results`

---

### Download Report

GET `/api/v1/jobs/:id/report`

---

## 11. Data Model Constraints

### scan_jobs

* id
* repo_url
* branch
* scan_type
* status
* block_on_high
* created_at
* started_at
* finished_at

#### Status values:

* pending
* running
* success
* failed
* blocked

---

### scan_results

* id
* job_id
* scanner_name
* severity
* rule_id
* file_path
* line_number
* title
* description
* evidence
* recommendation
* hash

---

### scan_reports

* id
* job_id
* report_path
* summary_json
* high_count
* medium_count
* low_count
* risk_score

---

## 12. Task Instruction Protocol (给 Codex 的行为规范)

When generating code:

1. Follow directory structure strictly
2. Do not invent new modules
3. Do not refactor unrelated code
4. Do not introduce new architecture
5. Keep implementation concrete (NO pseudo code)
6. Respect current phase constraints
7. Keep changes minimal and localized

---

## 13. Prompt Usage Pattern (必须遵守)

Always start prompts with:

```
Read /ai/context.md.

Strictly follow the AI context. Do not expand scope.
```

### Example:

```
Read /ai/context.md.

Strictly follow the AI context.

Implement job creation API using Gin.
Do not include worker logic.
```

---

## 14. Behavior Rules When Uncertain

* DO NOT guess
* ASK for clarification

---

## 15. Long-term Evolution (禁止提前实现)

### Phase 2:

* Add Secret Scan
* Add SCA

### Phase 3:

* Redis queue
* Async worker

### Phase 4:

* Blocking integration
* CI/CD integration

---

## 16. Engineering Goal

This project demonstrates:

* DevSecOps pipeline security design
* Multi-scanner orchestration
* Risk aggregation and enforcement
* Practical secure engineering implementation

### NOT:

* Feature completeness
* UI complexity
* Multi-language scanning

````
