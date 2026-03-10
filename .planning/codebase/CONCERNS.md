# Codebase Concerns

**Analysis Date:** 2026-03-09

> **Note:** This codebase is at initial scaffold stage. No source code exists yet — only `go.mod`, `.gitignore`, `README.md`, and `CLAUDE.md`. All concerns are pre-implementation risks and structural gaps that must be addressed as Phase 1 is built.

---

## Tech Debt

**No source code structure exists:**
- Issue: `cmd/upstream/`, `internal/`, `deploy/`, `config/` directories are absent. The layout defined in CLAUDE.md has not been created.
- Files: `go.mod` (only file that matters right now)
- Impact: Nothing compiles or runs. Phase 1 success criteria are entirely unmet.
- Fix approach: Create `cmd/upstream/main.go` as the entrypoint, then `internal/provider/azure/` for the RSS provider. Wire together in `main.go`.

**go.mod specifies a non-existent Go version:**
- Issue: `go.mod` declares `go 1.25.6` which does not exist as of this writing. Current stable Go is 1.22.x/1.23.x. This will cause `go build` to fail or warn depending on toolchain version.
- Files: `go.mod`
- Impact: `go build` and `go test` may refuse to run or produce unexpected toolchain download behavior.
- Fix approach: Update `go.mod` to a real released Go version (e.g., `go 1.23.0` or `go 1.22.5`).

**README is a stub:**
- Issue: `README.md` contains only a one-line project description. Phase 1 success criteria explicitly require the README to document build and run instructions.
- Files: `README.md`
- Impact: Fails Phase 1 acceptance criteria. No developer onboarding path.
- Fix approach: Add build, run, and test instructions once the CLI is implemented.

---

## Known Bugs

No source code exists — no bugs to report yet.

---

## Security Considerations

**No `.env` handling or secret management established:**
- Risk: When network calls to Azure RSS feeds are added, there is no pattern yet for injecting configuration (proxies, auth tokens for future authenticated feeds). Risk of developers hardcoding values.
- Files: None yet. Will be relevant in `cmd/upstream/main.go` and future `internal/` packages.
- Current mitigation: CLAUDE.md mandates `UPSTREAM_`-prefixed env vars, parsed only in `main.go`, never read from inside packages.
- Recommendations: Establish a `Config` struct in `cmd/upstream/main.go` immediately on first commit. Validate presence of required vars at startup and fail fast with a clear error.

**No HTTP client timeout defined (anticipated):**
- Risk: When the Azure RSS provider makes outbound HTTP requests, an unconfigured `http.Client` uses no timeout by default. A hanging upstream feed will block the process indefinitely.
- Files: Will be relevant in `internal/provider/azure/` (not yet created).
- Current mitigation: None established.
- Recommendations: Always construct `http.Client{Timeout: N * time.Second}` explicitly. Never use `http.DefaultClient` for outbound requests in production code.

---

## Performance Bottlenecks

No source code exists. One anticipated bottleneck:

**Synchronous RSS fetch (anticipated):**
- Problem: Phase 1 fetches from a single Azure RSS feed. If future phases add multiple providers (AWS, K8s), sequential HTTP fetches will be slow.
- Files: Will be relevant in `internal/provider/` (not yet created).
- Cause: Single-threaded sequential polling.
- Improvement path: Design the provider interface from the start to support concurrent invocation via goroutines + `errgroup`. The interface shape chosen in Phase 1 constrains this — define `Fetch(ctx context.Context) ([]Item, error)` so callers can fan out easily.

---

## Fragile Areas

**go.mod module path vs repository reality:**
- Files: `go.mod`
- Why fragile: Module path is `github.com/engineeringclouds/upstream` but the repo URL has not been confirmed to exist at that path. If the repo is hosted elsewhere or renamed, all internal import paths break.
- Safe modification: Verify the GitHub org and repo name match before writing any `internal/` packages, since every import path embeds this module path.
- Test coverage: N/A at this stage.

**Primary developer is learning Go (stated in CLAUDE.md):**
- Files: All future source files.
- Why fragile: Common Go anti-patterns (storing context in structs, swallowing errors, using `interface{}` instead of typed interfaces, misusing goroutines) can establish themselves early and become load-bearing before they're caught.
- Safe modification: Every PR should run `go vet ./...` and `gofmt -l .` as a gate. Consider adding a `Makefile` target for `lint` in Phase 1.
- Test coverage: TDD discipline (mandated in CLAUDE.md) is the primary mitigation — untested code is where bad patterns hide.

---

## Scaling Limits

**Phase 1 is stdout-only with no persistence:**
- Current capacity: Single run, prints to stdout, exits.
- Limit: No deduplication between runs. Running `./upstream fetch` twice will print duplicate items.
- Scaling path: Phase 2 store layer (`internal/store`) will need a seen-item record. Design the Azure provider's output type (`Item` or equivalent) with a stable unique key (e.g., RSS `<guid>` or `<link>`) from Phase 1 so the store can key on it without schema changes.

---

## Dependencies at Risk

**No external dependencies yet (`go.sum` absent):**
- Risk: None currently. Risk emerges if external packages are added without justification.
- Impact: Each unjustified external dependency increases supply chain attack surface and maintenance burden.
- Migration plan: CLAUDE.md mandates standard library preference. Enforce this — the only likely justified dependency in Phase 1 is none (stdlib `encoding/xml` and `net/http` cover the entire scope).

---

## Missing Critical Features

**No CI pipeline:**
- Problem: No GitHub Actions or other CI configuration exists. `go build`, `go test ./...`, `go vet`, and `gofmt` checks are not automated.
- Blocks: No safety net for catching regressions or convention violations in PRs.

**No Makefile or build script:**
- Problem: No `Makefile` exists. Developer must remember individual `go` commands.
- Blocks: Inconsistent build/test/lint invocation; onboarding friction.

**`--since` and `--format` CLI flags not implemented:**
- Problem: These are explicitly in Phase 1 scope but no CLI package exists yet.
- Blocks: Phase 1 success criteria cannot be met.

**No `testdata/` fixtures:**
- Problem: CLAUDE.md requires `testdata/` directories for XML parsing tests. None exist.
- Blocks: XML parsing tests cannot be written without a sample Azure RSS feed fixture.

---

## Test Coverage Gaps

**Zero test coverage (0%):**
- What's not tested: Everything — no source files exist.
- Files: None yet. Will be `internal/provider/azure/` when created.
- Risk: Phase 1 ships without any validation of XML parsing correctness, `--since` date filtering logic, or output formatting.
- Priority: High. CLAUDE.md mandates TDD. The Azure RSS XML parsing logic is the first thing that needs a test, written before the implementation.

---

*Concerns audit: 2026-03-09*
