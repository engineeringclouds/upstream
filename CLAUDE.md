# CLAUDE.md — Upstream

## Project

Upstream is an open-source, AI-enhanced infrastructure change awareness platform. Aggregates updates from cloud hyperscalers (Azure, AWS) and Kubernetes, classifies them with AI, and delivers personalized newsletter digests to subscribers.

**Repo**: `github.com/engineeringclouds/upstream` | **Language**: Go | **Owner**: Engineering Clouds LLC

Code quality, idiomatic Go, and documentation take priority over shipping speed.

## Developer Context

The primary developer is learning Go. This changes your behavior:

- **Explain non-obvious Go patterns** the first time they appear (context as first param, error return convention, value vs pointer receivers, etc.)
- **Flag non-idiomatic Go immediately.** Don't let bad patterns establish themselves.
- **Prefer standard library.** Justify any external dependency explicitly.
- **Error handling is not boilerplate.** Never swallow errors. Always wrap with `fmt.Errorf("context: %w", err)`. Use `errors.Is()`/`errors.As()`, not string comparison.

## Go Conventions

**Layout**: `cmd/upstream/` (thin entrypoint) → `internal/` (all application code) → `deploy/` → `config/`

**Naming**: Packages are short, lowercase, singular (`provider`, `store`, `mail`). Acronyms all caps (`URL`, `HTTP`, `ID`, `RSS`). Constants are `PascalCase`, not `SCREAMING_SNAKE`. Exported types get doc comments.

**Style**: `gofmt` and `go vet` always clean. Group imports: stdlib, third-party, internal. Struct literals use named fields. Early returns over deep nesting. Short functions.

**Interfaces**: Define where consumed, not implemented. Keep small (1-3 methods). Accept interfaces, return concrete types.

**Context**: Always first parameter. Never stored in structs.

**Testing**: TDD — red, green, refactor. Table-driven tests for multiple cases. Test files alongside code. Use `testdata/` for fixtures. Prefer `testing` package. Test public API, not internals.

**Config**: Environment variables prefixed `UPSTREAM_`. Parsed in `main.go`, injected via constructors. Never read env vars from inside packages. Never log sensitive values.

## Architecture Rules

**Dependency direction**:
```
cmd/upstream → internal/* (wiring only)
internal/web → internal/store, internal/digest, internal/auth
internal/digest → internal/store, internal/ai
internal/classifier → internal/ai
internal/provider/* → (external sources only, no internal deps)
```

Providers have zero internal dependencies. No circular deps — if you hit a cycle, fix the package boundaries.

## Commits

Format: `type(scope): description` (imperative present tense, under 72 chars)

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `style`
Scope: package or area (`provider/azure`, `store`, `web`, `deploy`)

## Post-Phase Gate

Every phase ends with a mandatory security review before merge. Run all checks from the repo root and resolve any findings before marking the phase complete.

**1. Vulnerability scan**
```bash
govulncheck ./...
```
Must return no vulnerabilities. If `govulncheck` is not installed: `go install golang.org/x/vuln/cmd/govulncheck@latest`.

**2. Secret/credential grep**
```bash
grep -rn \
  -e 'api[_-]?key\s*[:=]' \
  -e 'secret\s*[:=]' \
  -e 'password\s*[:=]' \
  -e 'token\s*[:=]' \
  -e 'AccountKey=' \
  -e 'DefaultEndpointsProtocol=' \
  -e 'connectionString\s*[:=]' \
  -e 'PRIVATE KEY' \
  --include='*.go' --include='*.yaml' --include='*.yml' \
  --include='*.json' --include='*.env' --include='*.toml' \
  --exclude-dir='.git' --exclude-dir='testdata' \
  .
```
Must return no matches in committed files. Matches in `testdata/` are acceptable only if they are clearly synthetic.

**3. .gitignore coverage**
Confirm these patterns are present:
- `*.env` or `.env`
- `*.pem`, `*.key`, `*.p12`, `*.pfx` (or `# secrets` equivalent)
- Built binaries (e.g., `upstream`)

**4. Dependency CVE check**
```bash
go list -m all | govulncheck -
# or if using Nancy/Snyk in CI, run equivalent
```
Any dependency with a known CVE must be updated or explicitly accepted with a documented reason.

**5. Hardcoded credential scan**
```bash
grep -rn \
  -e '[A-Za-z0-9+/]\{40,\}' \
  -e 'eyJ[A-Za-z0-9_-]\+\.' \
  --include='*.go' \
  --exclude-dir='.git' --exclude-dir='testdata' \
  .
```
Review any matches — long base64 strings or JWT-shaped tokens must not appear in source.

**Gate outcome**: All five checks must pass (zero findings or findings explicitly triaged) before a PR targeting `main` is approved.

## Current Phase: 1 — Foundation

**Goal**: Go CLI that polls Azure Updates RSS, parses XML, prints formatted output to stdout.

**In scope**: Azure RSS provider, XML parsing (`encoding/xml`), CLI with flags (`--since`, `--format`), tests for parsing, clean project structure.

**Out of scope**: Web UI, auth, AI, email, queues, Azure deployment, Bicep, AWS/K8s providers, Cosmos DB, any persistence layer.

**Success criteria**: `go build` works, `go test ./...` passes with meaningful coverage, `./upstream fetch` pulls and displays Azure updates, code passes `go vet`/`gofmt`, README documents build and run.
