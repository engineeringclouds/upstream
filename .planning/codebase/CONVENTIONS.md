# Coding Conventions

**Analysis Date:** 2026-03-09

## Naming Patterns

**Packages:**
- Short, lowercase, singular: `provider`, `store`, `mail`, `digest`, `auth`
- No underscores, no camelCase in package names

**Files:**
- Lowercase with underscores where needed: `azure_provider.go`, `rss_feed.go`
- Test files: `<name>_test.go` co-located with source

**Functions and Methods:**
- Exported: `PascalCase` — `FetchUpdates`, `ParseFeed`
- Unexported: `camelCase` — `parsePubDate`, `buildURL`

**Variables:**
- `camelCase` throughout: `feedURL`, `httpClient`, `lastSeen`
- Short names for short scopes: `err`, `i`, `n`, `r`, `w`

**Types:**
- Exported structs, interfaces, type aliases: `PascalCase`
- Doc comments required on all exported types

**Constants:**
- `PascalCase`, NOT `SCREAMING_SNAKE_CASE`
- Correct: `DefaultTimeout`, `MaxRetries`
- Wrong: `DEFAULT_TIMEOUT`, `MAX_RETRIES`

**Acronyms:**
- All-caps in names: `URL`, `HTTP`, `ID`, `RSS`, `XML`, `API`
- Correct: `ParseURL`, `feedID`, `httpClient`, `rssProvider`
- Wrong: `ParseUrl`, `feedId`, `HttpClient`, `RssProvider`

## Code Style

**Formatting:**
- `gofmt` — must always be clean. No exceptions.
- `go vet` — must always be clean before commit.

**Linting:**
- No linter config file present yet; `go vet` is the baseline
- Future: `golangci-lint` expected when linting is introduced

**Key style rules:**
- Early returns over deep nesting — guard clauses, not pyramids
- Short functions — single responsibility
- Named fields in struct literals always:
  ```go
  // Correct
  item := FeedItem{
      Title:       "...",
      PublishedAt: t,
  }

  // Wrong
  item := FeedItem{"...", t}
  ```

## Import Organization

**Order (three groups, blank-line separated):**
1. Standard library
2. Third-party packages
3. Internal packages (`github.com/engineeringclouds/upstream/internal/...`)

**Example:**
```go
import (
    "context"
    "encoding/xml"
    "fmt"
    "net/http"

    "github.com/some/external"

    "github.com/engineeringclouds/upstream/internal/provider"
)
```

**Path Aliases:**
- Avoid aliases unless resolving a genuine name collision
- Internal packages: `github.com/engineeringclouds/upstream/internal/<pkg>`

## Error Handling

**Core rules:**
- Never swallow errors — every error must be handled or returned
- Always wrap with context using `fmt.Errorf`:
  ```go
  if err != nil {
      return fmt.Errorf("fetch azure feed: %w", err)
  }
  ```
- Use `%w` verb (not `%v`) to allow callers to use `errors.Is()`/`errors.As()`
- Use `errors.Is()` and `errors.As()` for error inspection — never string comparison
- Sentinel errors use `PascalCase`: `var ErrNotFound = errors.New("not found")`

**Pattern:**
```go
result, err := doSomething(ctx)
if err != nil {
    return fmt.Errorf("doSomething: %w", err)
}
```

## Context Handling

**Rules:**
- `ctx context.Context` is always the first parameter of any function that does I/O or calls external services
- Context is NEVER stored in a struct field
- Pass context through, never create `context.Background()` deep in a call chain (only in `main` or test setup)

**Example:**
```go
// Correct
func (p *AzureProvider) Fetch(ctx context.Context) ([]Item, error) { ... }

// Wrong — context stored in struct
type Provider struct {
    ctx context.Context  // never do this
}
```

## Interfaces

**Rules:**
- Define interfaces where they are consumed, not where they are implemented
- Keep interfaces small: 1-3 methods maximum
- Accept interfaces as parameters; return concrete types

**Example:**
```go
// Defined in internal/digest, where it's consumed
type Provider interface {
    Fetch(ctx context.Context) ([]Item, error)
}
```

## Logging

**Framework:** Standard library `log/slog` (Go 1.21+)
**Sensitive values:** Never log API keys, tokens, connection strings, or personally identifiable information

## Comments

**Exported symbols:**
- All exported types, functions, methods, and constants require doc comments
- Format: `// TypeName does X.` or `// FunctionName does X.`

**Inline comments:**
- Explain why, not what
- Non-obvious logic warrants a comment; obvious code does not

## Configuration

**Rules:**
- All environment variables prefixed `UPSTREAM_`
- Parsed exclusively in `cmd/upstream/main.go`
- Injected into packages via constructors — packages never call `os.Getenv()`
- Never log configuration values that could be sensitive

## Module Design

**Exports:**
- Export only what external callers need
- Keep unexported helpers private

**Barrel files:**
- Not a Go convention — do not use `index.go` barrel-style re-exports
- Each package exposes its own public API directly

---

*Convention analysis: 2026-03-09 — derived from CLAUDE.md (authoritative source). No Go source files exist yet; this document reflects the prescribed conventions for all new code.*
