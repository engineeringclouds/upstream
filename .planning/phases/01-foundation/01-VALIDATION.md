---
phase: 1
slug: foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-10
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `testing` (stdlib) |
| **Config file** | none — `go test` convention |
| **Quick run command** | `go test ./internal/provider/azure/...` |
| **Full suite command** | `go test ./... && go vet ./... && gofmt -l .` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/provider/azure/... && go vet ./...`
- **After every plan wave:** Run `go test ./... && go vet ./... && gofmt -l .`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 0 | REQ-004, REQ-005 | compile | `go build ./...` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 0 | REQ-001, REQ-117 | fixture | `ls internal/provider/azure/testdata/feed.xml` | ❌ W0 | ⬜ pending |
| 1-01-03 | 01 | 0 | REQ-110, REQ-116 | unit | `go test ./internal/provider/azure/... -run TestParseDate` | ❌ W0 | ⬜ pending |
| 1-02-01 | 02 | 1 | REQ-001, REQ-002, REQ-003 | unit | `go test ./internal/provider/azure/... -run TestFetchFromFixture` | ✅ | ⬜ pending |
| 1-02-02 | 02 | 1 | REQ-003 | unit | `go test ./internal/provider/azure/... -run TestSinceFilter` | ✅ | ⬜ pending |
| 1-02-03 | 02 | 1 | REQ-002, REQ-005 | unit | `go test ./internal/provider/azure/... -run TestRawUpdateFields` | ✅ | ⬜ pending |
| 1-03-01 | 03 | 2 | REQ-020, REQ-021, REQ-022, REQ-023, REQ-024 | unit | `go test ./cmd/upstream/... -run TestOutput` | ❌ W0 | ⬜ pending |
| 1-03-02 | 03 | 2 | REQ-025 | manual | run with invalid `--format` value; verify exit 1 + stderr | — | ⬜ pending |
| 1-03-03 | 03 | 2 | REQ-026 | compile | `go build ./cmd/upstream` | ✅ | ⬜ pending |
| 1-04-01 | 04 | 3 | REQ-111 | static | `go vet ./...` | — | ⬜ pending |
| 1-04-02 | 04 | 3 | REQ-112 | static | `gofmt -l .` (must produce no output) | — | ⬜ pending |
| 1-04-03 | 04 | 3 | REQ-113 | static | `grep -rn 'log\.Fatal' internal/` (must be empty) | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/provider/provider.go` — Provider interface + RawUpdate struct (REQ-004, REQ-005)
- [ ] `internal/provider/azure/testdata/feed.xml` — real captured Azure RSS feed (REQ-001, REQ-117)
- [ ] `internal/provider/azure/azure_test.go` — table-driven stubs for parseDate, parse, Fetch (REQ-110, REQ-116)
- [ ] `go.mod` updated from `go 1.25.6` to `go 1.26.1`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Exit code 1 on error | REQ-025 | Process exit code not accessible via `go test` | Run `./upstream --format bad 2>&1; echo "exit: $?"` — must show error on stderr and `exit: 1` |
| `./upstream fetch` prints real Azure updates | REQ-020 | Requires live network access + stdout visual inspection | Run `./upstream fetch \| head -20` — verify title, date, URL, categories visible |
| `--since` filters correctly on live feed | REQ-021 | Live feed content changes; fixture covers logic | Run `./upstream fetch --since $(date -u +%Y-%m-%d)` — should return only today's or no updates |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
