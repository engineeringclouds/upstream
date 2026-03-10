# Codebase Structure

**Analysis Date:** 2026-03-09

## Current State

The repository is in initial scaffolding state (Phase 1 not yet started). Only root-level files exist. The structure below documents both current reality and the intended layout defined in `CLAUDE.md`.

## Directory Layout

```
upstream/                       # Project root
├── cmd/
│   └── upstream/               # CLI entry point (Phase 1 target — not yet created)
│       └── main.go             # Flag parsing, wiring, exit handling only
├── internal/                   # All application code (Go enforces privacy)
│   ├── provider/               # External data source adapters (Phase 1 target)
│   │   └── azure/              # Azure Updates RSS provider
│   │       ├── azure.go        # Provider implementation
│   │       ├── azure_test.go   # Provider tests
│   │       └── testdata/       # Sample RSS XML fixtures
│   ├── store/                  # Persistence layer (planned, post-Phase 1)
│   ├── ai/                     # AI client wrappers (planned, post-Phase 1)
│   ├── classifier/             # Update classification (planned, post-Phase 1)
│   ├── digest/                 # Newsletter assembly (planned, post-Phase 1)
│   ├── web/                    # HTTP server and handlers (planned, post-Phase 1)
│   └── auth/                   # Authentication (planned, post-Phase 1)
├── deploy/                     # Infrastructure / deployment configs (planned)
├── config/                     # App configuration schemas (planned)
├── .planning/                  # GSD planning documents (not committed to production)
│   └── codebase/               # Codebase analysis documents
├── CLAUDE.md                   # Project instructions for Claude
├── README.md                   # Human-facing project documentation
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums (will appear when deps added)
├── .gitignore                  # Go-standard ignores + .env
└── LICENSE                     # Open source license
```

## Directory Purposes

**`cmd/upstream/`:**
- Purpose: Thin binary entry point — wiring only, zero business logic
- Contains: `main.go` exclusively
- Key files: `cmd/upstream/main.go`

**`internal/`:**
- Purpose: All application packages — enforced private by Go toolchain (nothing outside this module can import these)
- Contains: All business logic, providers, store, web, AI

**`internal/provider/`:**
- Purpose: Adapters that fetch raw updates from external sources
- Contains: One sub-directory per source
- Key files: `internal/provider/azure/azure.go`, `internal/provider/azure/azure_test.go`

**`internal/provider/azure/`:**
- Purpose: Fetch and parse Azure Updates RSS feed
- Contains: HTTP fetch logic, XML parsing, typed update structs
- Key files: `internal/provider/azure/azure.go`

**`internal/provider/azure/testdata/`:**
- Purpose: Sample RSS XML for deterministic tests
- Generated: No — manually maintained fixtures
- Committed: Yes

**`deploy/`:**
- Purpose: Infrastructure-as-code, CI/CD configs, container definitions
- Generated: No
- Committed: Yes

**`config/`:**
- Purpose: Application configuration schemas or default config files
- Generated: No
- Committed: Yes

**`.planning/`:**
- Purpose: GSD planning documents — codebase analysis, phase plans
- Generated: By GSD commands
- Committed: Depends on team preference; typically yes for shared context

## Key File Locations

**Entry Points:**
- `cmd/upstream/main.go`: CLI binary entry point (Phase 1 target)

**Configuration:**
- `go.mod`: Module path (`github.com/engineeringclouds/upstream`), Go version
- `CLAUDE.md`: Project conventions, architecture rules, phase scope

**Core Logic (Phase 1):**
- `internal/provider/azure/azure.go`: RSS fetch and XML parse
- `internal/provider/azure/azure_test.go`: Provider tests

**Testing Fixtures:**
- `internal/provider/azure/testdata/`: Sample Azure RSS XML files

## Naming Conventions

**Files:**
- Go source: `<noun>.go` — short, lowercase, matches package purpose (e.g., `azure.go`)
- Test files: `<noun>_test.go` — alongside the file under test, same directory
- Fixtures: `testdata/<descriptive-name>.xml` — lowercase, hyphenated if multi-word

**Directories:**
- Short, lowercase, singular: `provider`, `store`, `mail`, `web`
- Sub-packages use parent as namespace: `provider/azure`, `provider/aws`
- Acronyms all-caps in identifiers but not in directory names: `internal/ai/` not `internal/AI/`

**Go Identifiers:**
- Packages: short, lowercase, singular (`azure`, `provider`, `store`)
- Exported types/functions: `PascalCase` with doc comment
- Unexported: `camelCase`
- Acronyms: all caps (`URL`, `HTTP`, `ID`, `RSS`) in identifiers
- Constants: `PascalCase` (not `SCREAMING_SNAKE`)
- Environment variables: `UPSTREAM_` prefix, `SCREAMING_SNAKE` (OS convention)

## Where to Add New Code

**New provider (e.g., AWS):**
- Implementation: `internal/provider/aws/aws.go`
- Tests: `internal/provider/aws/aws_test.go`
- Fixtures: `internal/provider/aws/testdata/`
- Rule: Zero internal package imports allowed

**New internal package:**
- Implementation: `internal/<package>/<package>.go`
- Tests: `internal/<package>/<package>_test.go`
- Check dependency direction before adding any import from `internal/`

**New CLI subcommand:**
- Add to: `cmd/upstream/main.go` (or a new file in `cmd/upstream/` if it grows large)
- Keep wiring only — delegate immediately to `internal/`

**Shared utilities (if needed):**
- Shared helpers with no external deps: `internal/util/` or inline in the consuming package
- Avoid premature extraction — only create a shared package once 2+ packages need it

**Infrastructure / deployment:**
- Bicep, Dockerfiles, CI configs: `deploy/`

## Special Directories

**`internal/`:**
- Purpose: Go toolchain enforces that code outside this module cannot import these packages
- Generated: No
- Committed: Yes

**`testdata/`:**
- Purpose: Test fixture files (XML samples, golden files, etc.)
- Generated: No (manually curated)
- Committed: Yes — fixtures are part of the test suite
- Convention: Go toolchain ignores `testdata/` during build

---

*Structure analysis: 2026-03-09*
