---
phase: 01-foundation
plan: 02
subsystem: provider
tags: [go, rss, xml, encoding/xml, azure, http]

# Dependency graph
requires:
  - phase: 01-foundation/01-01
    provides: provider.Provider interface, provider.RawUpdate struct, azure_test.go, testdata/feed.xml

provides:
  - AzureProvider struct implementing provider.Provider
  - New(*http.Client) constructor with injectable http.Client
  - parseDate multi-format parser handling Azure's non-standard "space + Z" timezone
  - parse(io.Reader, time.Time) internal XML decoder with since-time filter
  - All five tests passing: TestParseDate, TestFetchFromFixture, TestSinceFilter, TestRawUpdateFields, TestProviderInterface

affects:
  - 01-03 (CLI wiring — uses AzureProvider.Fetch via provider.Provider interface)
  - All future provider implementations (azure pattern is the reference)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Multi-format date parsing: try formats in order, return error only when all fail"
    - "Unexported RSS struct types: wire format is implementation detail, callers see []RawUpdate"
    - "Atom namespace XML: full URI required in struct tag (xml:\"http://www.w3.org/2005/Atom updated\"), not namespace prefix"
    - "Injectable http.Client: accept *http.Client in constructor, default to 30s timeout, enables fixture testing"
    - "Error wrapping: fmt.Errorf(\"azure: operation: %w\", err) pattern for all errors"
    - "nil-to-empty-slice: always return []provider.RawUpdate{} not nil when no results"

key-files:
  created:
    - internal/provider/azure/azure.go
  modified: []

key-decisions:
  - "parseDate tries four formats in order: live Azure space+Z format first (most common), then RFC1123Z, RFC1123, RFC3339"
  - "Items with unparseable dates get Published=zero and a slog.Warn, not dropped — silent data loss is worse than zero time"
  - "Atom a10:updated used as fallback date source when pubDate unparseable"
  - "Pointer receiver on AzureProvider because it holds http.Client (meaningful state)"

patterns-established:
  - "Provider zero-dep rule: internal/provider/azure imports only internal/provider (for RawUpdate) and stdlib"
  - "TDD fixture testing: real captured feed.xml, not synthetic data, catches live format quirks"

requirements-completed: [REQ-001, REQ-002, REQ-003, REQ-006, REQ-113, REQ-114, REQ-115]

# Metrics
duration: ~4min (plus ~2min wait for 01-01 files)
completed: 2026-03-10
---

# Phase 1 Plan 02: AzureProvider Implementation Summary

**Azure RSS provider in Go stdlib only: multi-format pubDate parsing, XML decode with encoding/xml, since-time filtering, all five tests green**

## Performance

- **Duration:** ~4 min implementation (waited ~2 min for 01-01 dependency files)
- **Started:** 2026-03-10T21:52:00Z
- **Completed:** 2026-03-10T21:55:33Z
- **Tasks:** 1 (GREEN + REFACTOR combined — single implementation file)
- **Files modified:** 1

## Accomplishments

- Implemented `AzureProvider` satisfying `provider.Provider` with `Name()` and `Fetch()` methods
- `parseDate` correctly handles Azure's non-standard `"Wed, 04 Mar 2026 21:15:02 Z"` space+Z timezone format that `time.RFC1123` and `time.RFC1123Z` cannot parse
- `parse()` decodes RSS XML via `encoding/xml`, applies since-time filter, falls back to `a10:updated` when `pubDate` unparseable
- All five tests pass: `TestParseDate` (6 subtests), `TestFetchFromFixture`, `TestSinceFilter`, `TestRawUpdateFields`, `TestProviderInterface`
- Code is `gofmt` clean, `go vet` clean, no `log.Fatal` or `os.Getenv` in the package

## Task Commits

1. **AzureProvider implementation (GREEN + REFACTOR)** - `77b3678` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `internal/provider/azure/azure.go` - AzureProvider, New, Name, Fetch, parse, parseDate, rssItem/rssChannel/rssFeed unexported types

## Decisions Made

- `parseDate` tries formats in order (space+Z first as most common on live feed), returns error only when all four fail
- Items with unparseable dates get `Published` zero-value and `slog.Warn` rather than being dropped — silent data loss harder to debug than a zero timestamp
- `a10:updated` serves as fallback date source (RFC3339 format, always parseable)
- Atom namespace struct tag uses full URI `"http://www.w3.org/2005/Atom updated"` not the `a10:` prefix — Go's encoding/xml silently ignores prefix-based tags

## Deviations from Plan

None - plan executed exactly as written. gofmt aligned comments in the formats slice (cosmetic only, no logic change).

## Issues Encountered

None. Tests went RED -> GREEN -> REFACTOR in one pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Azure provider complete and tested; ready for CLI wiring in 01-03
- `AzureProvider` exported via `internal/provider/azure` package; `New(nil)` returns production-ready instance
- Pattern established: future AWS/Kubernetes providers should follow same structure (unexported feed types, injectable http.Client, since-filter in parse)

---
*Phase: 01-foundation*
*Completed: 2026-03-10*
