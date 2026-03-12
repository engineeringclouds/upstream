---
phase: 01-foundation
plan: 03
subsystem: cli
tags: [go, flag, json, encoding/json, cmd, stdout, stderr]

# Dependency graph
requires:
  - phase: 01-foundation/01-01
    provides: Provider interface and RawUpdate struct in internal/provider
  - phase: 01-foundation/01-02
    provides: AzureProvider.Fetch implementation in internal/provider/azure

provides:
  - cmd/upstream/main.go — thin CLI wiring layer with fetch subcommand
  - --since YYYY-MM-DD flag for date-filtered fetching
  - --format text|json flag for output selection
  - text output: Title/Date/URL/Tags blocks per update
  - json output: indented JSON array of RawUpdate objects
  - error terminal: only file in project that calls os.Exit(1)

affects: [02-ingest, 03-web, future CLI extensions]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "flag.FlagSet per subcommand — scopes flags, enables future subcommand expansion without collisions"
    - "runFetch and output extracted from main() so they return errors instead of calling os.Exit — keeps logic testable"
    - "cmd/ is the sole error terminal — internal/ code always returns errors up the call stack"
    - "io.Writer injection in output() and outputText() — decouples formatting from os.Stdout for testability"

key-files:
  created:
    - cmd/upstream/main.go
  modified: []

key-decisions:
  - "flag.ContinueOnError used (not ExitOnError) so fetchCmd errors flow through main()'s error path, keeping os.Exit calls centralised"
  - "No unit test file in cmd/upstream/ — output functions designed for testability but Plan 03 verifies via compiled binary per plan spec"
  - "runFetch calls azure.New(nil) directly — Phase 1 has a single provider; multi-provider wiring deferred to later phases"

patterns-established:
  - "cmd/ entrypoint pattern: flag.FlagSet per subcommand, switch on os.Args[1], all errors returned to main()"
  - "Output function signature: output(w io.Writer, updates []provider.RawUpdate, format string) error — injectable writer for tests"
  - "Error wrapping in cmd/: fmt.Errorf(\"upstream: %w\", err) — consistent prefix for user-visible error messages"

requirements-completed: [REQ-020, REQ-021, REQ-022, REQ-023, REQ-024, REQ-025, REQ-026]

# Metrics
duration: 10min
completed: 2026-03-11
---

# Phase 1 Plan 03: CLI Entrypoint Summary

**`upstream fetch` CLI wiring layer with `--since` date filter, `--format text|json` output, and stderr error handling — verified live against Azure RSS feed**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-03-11T21:00:00Z
- **Completed:** 2026-03-11T21:10:00Z
- **Tasks:** 2 (1 auto + 1 human-verify checkpoint)
- **Files modified:** 1

## Accomplishments

- `cmd/upstream/main.go` implemented: `fetch` subcommand with `--since` and `--format` flags
- Text output prints Title/Date/URL/Tags blocks; JSON output uses `json.NewEncoder` with indentation
- Error terminal pattern established: `cmd/` is the only layer that calls `os.Exit(1)` — all internal code returns errors
- Live human verification confirmed: all 6 test cases passed against the real Azure RSS feed

## Task Commits

1. **Task 1: Implement CLI entrypoint** - `18ef7c1` (feat)
2. **Task 2: Human verify CLI end-to-end** - human approved (no commit — verification only)

**Plan metadata:** (docs commit below)

## Files Created/Modified

- `cmd/upstream/main.go` — CLI entrypoint: flag parsing, provider wiring, text/JSON output, error handling to stderr

## Decisions Made

- `flag.ContinueOnError` used instead of `flag.ExitOnError` so parse errors flow through `main()`'s error path, keeping all `os.Exit` calls in one place
- No test file in `cmd/upstream/` — plan explicitly specifies binary-level verification rather than `go test`; `runFetch` and `output` are designed to be testable if needed later
- `azure.New(nil)` called directly in `runFetch` — Phase 1 scope is single provider; multi-provider registry deferred to later phases

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 1 complete: `go build ./cmd/upstream`, `go test ./...`, `go vet ./...`, and `gofmt` all pass clean
- Binary produces correct text and JSON output against the live Azure RSS feed
- `--since` date filtering verified working
- Ready for Phase 2 (ingest pipeline, persistence, web layer)

---
*Phase: 01-foundation*
*Completed: 2026-03-11*
