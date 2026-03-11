---
phase: 01-foundation
plan: 01
subsystem: testing
tags: [go, provider, interface, rss, tdd, xml]

# Dependency graph
requires: []
provides:
  - Provider interface (Name, Fetch) in internal/provider/provider.go
  - RawUpdate struct with all fields for normalised provider output
  - Real Azure RSS fixture at internal/provider/azure/testdata/feed.xml (296 lines, captured 2026-03-10)
  - Failing test stubs in internal/provider/azure/azure_test.go (RED phase)
  - go.mod at go 1.26.1
affects:
  - 01-02 (implements against Provider interface and azure_test.go stubs)
  - 01-03 (CLI wires against Provider interface)
  - all subsequent phases using provider.RawUpdate

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED phase: test stubs created before implementation"
    - "Interface-at-consumption-site: Provider interface lives in provider pkg, not azure pkg"
    - "Fixture-based tests: real RSS capture in testdata/, not synthetic mocks"
    - "Table-driven tests: struct slice with named fields for easy case addition"
    - "Compile-time interface assertions: var _ provider.Provider = &AzureProvider{}"

key-files:
  created:
    - internal/provider/provider.go
    - internal/provider/azure/testdata/feed.xml
    - internal/provider/azure/azure_test.go
  modified:
    - go.mod

key-decisions:
  - "Provider interface placed in provider package (consumption site), azure package has zero internal deps"
  - "RawMeta map[string]string captures provider-specific extras (a10:updated) without polluting the contract"
  - "ContentFormat field is 'html' or 'text' string, not an enum — keeps the struct simple for Phase 1"
  - "testdata/feed.xml is a real live capture, not synthetic — catches real format quirks (space-before-Z pubDate)"

patterns-established:
  - "Provider interface: 2 methods only (Name, Fetch) — resist adding methods, behaviour lives in pipeline"
  - "context.Context first param on all I/O functions — passed through, never stored in structs"
  - "Error wrapping: fmt.Errorf(\"context: %w\", err) — enforced in test stubs via parseDate error path"
  - "Table-driven tests: []struct with name/input/want fields — all azure tests follow this pattern"

requirements-completed:
  - REQ-004
  - REQ-005
  - REQ-110
  - REQ-116
  - REQ-117

# Metrics
duration: 2min
completed: 2026-03-11
---

# Phase 1 Plan 1: TDD Foundation — Provider Interface and Test Stubs

**Provider interface + RawUpdate struct + real Azure RSS fixture + five failing test stubs establishing the RED phase for Plan 02's implementation**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-11T01:52:20Z
- **Completed:** 2026-03-11T01:54:20Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Fixed go.mod version mismatch (1.25.6 -> 1.26.1) before any code was written
- Defined the Provider interface and RawUpdate struct that all downstream phases implement against
- Captured a real 296-line Azure RSS feed as testdata/feed.xml — including the non-standard `"Wed, 04 Mar 2026 21:15:02 Z"` pubDate format that synthetic fixtures would miss
- Wrote five table-driven test stubs in azure_test.go covering parseDate (6 cases), parse, since-filtering, field mapping, and the Provider interface compile-time check

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix go.mod and create project directory structure** - `cd6bf2d` (chore)
2. **Task 2: Define Provider interface and RawUpdate struct** - `7fb1f99` (feat)
3. **Task 3: Capture real Azure RSS fixture and write failing test stubs** - `e081d99` (test)

## Files Created/Modified
- `go.mod` - Updated Go version directive from 1.25.6 to 1.26.1
- `internal/provider/provider.go` - Provider interface (Name, Fetch) and RawUpdate struct with full doc comments explaining Go patterns
- `internal/provider/azure/testdata/feed.xml` - Real Azure RSS feed capture (296 lines, 2026-03-10)
- `internal/provider/azure/azure_test.go` - Five table-driven test stubs: TestParseDate, TestFetchFromFixture, TestSinceFilter, TestRawUpdateFields, TestProviderInterface

## Decisions Made
- Interface placed in `provider` package (consumption site), not `azure` package — enforces the architecture rule that providers have zero internal deps. The azure package returns `[]provider.RawUpdate` but never imports the provider package itself; callers wire them together.
- `RawMeta map[string]string` for provider-specific extras (a10:updated) — avoids polluting the contract with Azure-specific fields while still making the data available.
- Used a real live RSS capture for testdata, not synthetic XML — the live feed revealed that Azure uses a non-standard pubDate format (`"Wed, 04 Mar 2026 21:15:02 Z"` with a space before Z) that standard `time.RFC1123` and `time.RFC1123Z` layouts cannot parse. This is a key risk that Plan 02's multi-format parseDate must handle explicitly.

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered
- `go build ./internal/provider/` triggered a toolchain download for Go 1.26.1 on first run (expected — new version in go.mod). Build succeeded after download.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Plan 02 has everything it needs: the Provider interface to implement, the RawUpdate struct to populate, and five failing tests that define "done"
- The non-standard Azure pubDate format (`"... Z"` with space) is documented in TestParseDate — Plan 02 must handle this as the first case in parseDate's format list
- `go vet ./internal/provider/azure/...` currently fails on `undefined: parseDate` — this is correct RED state; Plan 02 must turn it GREEN

---
*Phase: 01-foundation*
*Completed: 2026-03-11*
