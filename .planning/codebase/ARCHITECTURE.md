# Architecture

**Analysis Date:** 2026-03-09

## Pattern Overview

**Overall:** Layered CLI application with provider-based data ingestion, evolving toward a web service with AI classification and newsletter delivery.

**Key Characteristics:**
- Strict unidirectional dependency flow — outer layers depend on inner layers, never vice versa
- Providers are fully isolated from internal packages (zero internal deps)
- `cmd/upstream/` is a thin wiring layer only — no business logic
- All application code lives under `internal/` (Go toolchain enforces package privacy)
- Config injected via constructors from `main.go` — packages never read env vars directly

## Layers

**CLI Entry Point:**
- Purpose: Parse flags, wire dependencies, call internal packages
- Location: `cmd/upstream/` (not yet created — Phase 1 target)
- Contains: `main.go` only; thin orchestration, no business logic
- Depends on: `internal/*`
- Used by: OS / user invocation

**Provider Layer:**
- Purpose: Fetch raw update data from external sources (Azure RSS, future AWS, K8s)
- Location: `internal/provider/` (not yet created)
- Contains: One sub-package per source (e.g., `internal/provider/azure/`)
- Depends on: External HTTP sources only — zero internal package dependencies
- Used by: `cmd/upstream/` (Phase 1), eventually `internal/digest/`

**Store Layer:**
- Purpose: Persistence — reading and writing update records
- Location: `internal/store/` (planned, out of scope for Phase 1)
- Contains: Store interface and implementations
- Depends on: Nothing internal
- Used by: `internal/web/`, `internal/digest/`

**AI Layer:**
- Purpose: Classification and enrichment of updates
- Location: `internal/ai/` (planned, out of scope for Phase 1)
- Contains: AI client wrappers
- Depends on: Nothing internal
- Used by: `internal/digest/`, `internal/classifier/`

**Classifier:**
- Purpose: Classify updates using AI
- Location: `internal/classifier/` (planned, out of scope for Phase 1)
- Contains: Classification logic
- Depends on: `internal/ai/`
- Used by: digest pipeline

**Digest:**
- Purpose: Assemble newsletter content from stored + classified updates
- Location: `internal/digest/` (planned, out of scope for Phase 1)
- Contains: Digest generation logic
- Depends on: `internal/store/`, `internal/ai/`
- Used by: `internal/web/`

**Web:**
- Purpose: HTTP server, subscriber management, newsletter delivery
- Location: `internal/web/` (planned, out of scope for Phase 1)
- Contains: HTTP handlers, middleware
- Depends on: `internal/store/`, `internal/digest/`, `internal/auth/`
- Used by: `cmd/upstream/`

**Auth:**
- Purpose: Authentication and authorization
- Location: `internal/auth/` (planned, out of scope for Phase 1)
- Contains: Auth middleware and session logic
- Depends on: `internal/store/`
- Used by: `internal/web/`

## Data Flow

**Phase 1 — CLI Fetch:**

1. User runs `./upstream fetch [--since DATE] [--format FORMAT]`
2. `cmd/upstream/main.go` parses flags, constructs provider
3. Azure RSS provider fetches XML from Azure Updates RSS feed over HTTP
4. Provider parses XML using `encoding/xml` into typed structs
5. `main.go` formats and prints results to stdout

**Future — Newsletter Digest:**

1. Scheduler triggers fetch across all providers
2. Providers write raw updates to `internal/store/`
3. `internal/classifier/` reads unclassified updates, calls `internal/ai/` to tag them
4. `internal/digest/` assembles personalized digests per subscriber from store
5. `internal/web/` or a mailer sends digest via email

**State Management:**
- Phase 1: Stateless — no persistence, all data flows through in-process structs
- Future phases: Persistence via `internal/store/` (Cosmos DB planned)

## Key Abstractions

**Provider Interface (planned):**
- Purpose: Uniform contract for all update sources (Azure, AWS, K8s)
- Examples: `internal/provider/azure/` (Phase 1 target)
- Pattern: Small interface (1-3 methods) defined where consumed, implemented per source

**Update struct (planned):**
- Purpose: Canonical representation of a single infrastructure update item
- Examples: `internal/provider/azure/` — parsed from RSS XML
- Pattern: Value type, named fields, exported with doc comments

**Config injection:**
- Purpose: Keep packages testable and free of global state
- Pattern: `main.go` reads `UPSTREAM_*` env vars, passes values into constructors
- No package reads `os.Getenv()` directly

## Entry Points

**CLI binary:**
- Location: `cmd/upstream/main.go` (Phase 1 target — not yet created)
- Triggers: `go run ./cmd/upstream` or compiled binary `./upstream`
- Responsibilities: Flag parsing (`--since`, `--format`), dependency wiring, exit code handling

## Error Handling

**Strategy:** Explicit propagation — errors are always returned, never swallowed. Callers decide how to handle.

**Patterns:**
- Wrap with context: `fmt.Errorf("azure: fetch feed: %w", err)`
- Check with `errors.Is()` / `errors.As()` — never string comparison
- `main.go` logs error and exits with non-zero code on fatal errors

## Cross-Cutting Concerns

**Logging:** Standard library `log` package; no sensitive values logged
**Validation:** At parse boundaries (XML → struct); providers validate before returning
**Authentication:** Out of scope for Phase 1; planned in `internal/auth/`
**Config:** `UPSTREAM_` prefixed env vars, parsed in `main.go` only, injected via constructors

---

*Architecture analysis: 2026-03-09*
