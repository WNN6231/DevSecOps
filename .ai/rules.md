很好，这一版已经**结构完全正确 + 可被 Codex 稳定解析**。
现在不需要再改 context 了，下一步该做的是你提到的关键补强：

> 👉 `.ai/rules.md`（编码风格与工程约束）

我直接给你一份**企业级版本（专门针对 Go + 安全开发 + 你这个项目）**，你可以直接落地。

---

# ✅ 创建文件

```
/.ai/rules.md
```

---

# 📄 内容（完整可用版）

````markdown
# AI Coding Rules: DevSecOps Security Gate Platform

## 1. General Principles

- Code must be simple, explicit, and readable
- Avoid unnecessary abstraction
- Prefer clarity over cleverness
- Follow Go idioms strictly
- Minimize dependencies

---

## 2. Project Awareness (必须遵守)

The AI MUST:

- Respect /ai/context.md
- Follow current development phase constraints
- Only implement what is explicitly requested

The AI MUST NOT:

- Expand scope
- Implement future phase features
- Modify architecture

---

## 3. Code Style (Go)

### Naming

- Use camelCase for variables
- Use PascalCase for exported types/functions
- Use short but meaningful names

Examples:

- jobID (good)
- j (bad)
- jobIdentifierThatIsTooLong (bad)

---

### Functions

- One function = one responsibility
- Keep functions under ~50 lines
- Avoid deep nesting (>3 levels)

---

### Error Handling

- Always handle errors explicitly
- Do not ignore errors

Example:

```go
if err != nil {
    return err
}
````

---

### Struct Design

* Use simple structs
* Avoid embedding unless necessary
* Do not introduce interfaces prematurely

---

## 4. API Design Rules

* Use RESTful conventions
* Return JSON only
* Do not introduce GraphQL or RPC

### Response format

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

---

## 5. Logging Rules

* Use simple logging (fmt or log)
* Include context (job_id, status)
* Do NOT introduce logging frameworks

---

## 6. Storage Rules (Phase 1)

* Use in-memory storage ONLY
* No database
* No Redis

Example:

```go
var jobStore = make(map[int]*Job)
```

---

## 7. Concurrency Rules

* Phase 1: NO concurrency required
* Do NOT use goroutines unless explicitly requested

---

## 8. Dependency Rules

Allowed:

* Gin (web framework)
* Standard library

Not allowed:

* Large frameworks
* ORMs
* Dependency injection frameworks

---

## 9. Scanner Implementation Rules

* Must output []Finding
* Must be deterministic
* Must not modify source files

---

## 10. Security Coding Rules (核心)

* Never execute untrusted input directly
* Validate all external input
* Avoid shell execution when possible
* Do not hardcode secrets

---

## 11. File Structure Rules

* One file = one responsibility
* Avoid large files (>300 lines)
* Keep directory structure unchanged

---

## 12. Code Generation Constraints

The AI MUST:

* Output full, runnable code
* Include imports
* Ensure compilation passes

The AI MUST NOT:

* Output pseudo code
* Skip critical parts
* Leave TODO without implementation

---

## 13. Modification Rules

When modifying existing code:

* Only change necessary parts
* Do not rewrite entire files
* Do not refactor unrelated logic

---

## 14. Output Format Rules

When generating code:

1. Provide complete file content
2. Specify file path
3. Do not mix multiple files unless asked

Example:

File: internal/job/job.go

```go
// full code here
```

---

## 15. Failure Handling Rules

If requirements are unclear:

* STOP
* ASK for clarification

DO NOT guess implementation.

---

## 16. Engineering Mindset

The AI should behave like:

* A mid-to-senior Go backend engineer
* Focused on correctness and simplicity
* Not a "creative coder"

Goal:

* Produce production-style code
* Maintain consistency across modules

---

## 17. API Handler Pattern (必须遵守)

All HTTP handlers MUST follow this pattern:

1. Parse request
2. Validate input
3. Call service logic
4. Return JSON response

Example:

```go
func CreateJobHandler(c *gin.Context) {
    var req CreateJobRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"code": 1, "message": "invalid request"})
        return
    }

    job, err := jobService.CreateJob(req)
    if err != nil {
        c.JSON(500, gin.H{"code": 1, "message": "internal error"})
        return
    }

    c.JSON(200, gin.H{
        "code": 0,
        "message": "ok",
        "data": job,
    })
}

---

## 18. Service Layer Rules

- Business logic MUST be placed in service layer
- Handlers must NOT contain business logic
- Service functions must be reusable and testable

Example:

internal/job/service.go

```go
func CreateJob(req CreateJobRequest) (*Job, error) {
    // business logic here
}

---

## 19. Data Flow Rules

- Use dedicated request/response structs
- Do NOT reuse internal models as API response
- Keep API layer and internal model separated

Example:

```go
type CreateJobRequest struct {
    RepoURL string   `json:"repo_url"`
    Branch  string   `json:"branch"`
    ScanType []string `json:"scan_type"`
}

type JobResponse struct {
    JobID  int    `json:"job_id"`
    Status string `json:"status"`
}