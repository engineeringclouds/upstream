# Technology Stack

**Analysis Date:** 2026-03-09

## Languages

**Primary:**
- Go 1.25.6 - All application code

**Secondary:**
- None

## Runtime

**Environment:**
- Go 1.25.6 (module: `github.com/engineeringclouds/upstream`)

**Package Manager:**
- Go modules (`go mod`)
- Lockfile: `go.sum` (not yet present — no external dependencies declared)

## Frameworks

**Core:**
- None — standard library only (Phase 1 constraint)

**Testing:**
- `testing` package (Go standard library)

**Build/Dev:**
- `go build` — compilation
- `go test ./...` — test runner
- `gofmt` — formatter (required, all code must be clean)
- `go vet` — static analysis (required, all code must be clean)

## Key Dependencies

**Critical:**
- None declared. `go.mod` contains only the module declaration and Go version. No external dependencies.

**Planned (future phases, not yet added):**
- AI classification integration (unspecified SDK — out of scope Phase 1)
- Email delivery (unspecified — out of scope Phase 1)
- Cosmos DB client (out of scope Phase 1)

## Configuration

**Environment:**
- Variables prefixed `UPSTREAM_`
- Parsed exclusively in `main.go`, injected into packages via constructors
- Packages must not read env vars directly
- Sensitive values must never be logged

**Build:**
- `go.mod` — module definition at repo root
- `.gitignore` — standard Go gitignore; `.env` excluded from version control

## Platform Requirements

**Development:**
- Go 1.25.6+
- macOS or Linux (no platform-specific constraints identified)

**Production:**
- Not yet defined (Azure deployment is out of scope for Phase 1)
- Planned: Azure (Bicep deployment, out of scope Phase 1)

---

*Stack analysis: 2026-03-09*
