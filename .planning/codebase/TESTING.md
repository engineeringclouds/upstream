# Testing Patterns

**Analysis Date:** 2026-03-09

## Test Framework

**Runner:**
- Go standard library `testing` package — no third-party test runner
- Go version: 1.25.6 (module: `github.com/engineeringclouds/upstream`)
- Config: none — `go test` is configured via flags, not a config file

**Assertion Library:**
- Standard library only — no `testify` or similar
- Use `t.Errorf`, `t.Fatalf`, `t.Helper()` for custom assertion helpers

**Run Commands:**
```bash
go test ./...              # Run all tests
go test ./... -v           # Verbose output
go test ./... -run TestFoo # Run specific test
go test -count=1 ./...     # Disable test caching (force re-run)
go test -cover ./...       # Show coverage summary
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out  # HTML coverage report
go vet ./...               # Must pass before any commit
```

## Test File Organization

**Location:**
- Co-located with source: `internal/provider/azure.go` → `internal/provider/azure_test.go`
- Same package for white-box tests: `package provider`
- Separate `_test` package for black-box tests of public API: `package provider_test`
- Prefer black-box (`package foo_test`) for testing public API; use same-package only when testing unexported helpers is necessary

**Naming:**
- Test functions: `TestFunctionName`, `TestTypeName_MethodName`
- Subtests: `t.Run("case description", func(t *testing.T) { ... })`
- Benchmark functions: `BenchmarkFunctionName`

**Structure:**
```
internal/
  provider/
    azure.go
    azure_test.go
    testdata/
      azure_feed.xml       # Sample RSS feed for parsing tests
  digest/
    digest.go
    digest_test.go
```

## Test Structure

**Methodology:** TDD — write the failing test first, implement to pass, refactor.

**Table-driven tests for multiple cases:**
```go
func TestParseItem(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Item
        wantErr bool
    }{
        {
            name:  "valid item",
            input: "<item>...</item>",
            want:  Item{Title: "...", PublishedAt: someTime},
        },
        {
            name:    "missing title",
            input:   "<item></item>",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseItem(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("ParseItem() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && got != tt.want {
                t.Errorf("ParseItem() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Setup/teardown patterns:**
```go
func TestMain(m *testing.M) {
    // Suite-level setup
    os.Exit(m.Run())
}

func setup(t *testing.T) (*SomeType, func()) {
    t.Helper()
    s := NewSomeType()
    return s, func() { s.Close() }
}
```

## Mocking

**Framework:** Standard library interfaces only — no `gomock`, no `testify/mock`

**Pattern — test doubles via interface:**
```go
// In production code (internal/digest/digest.go):
type Provider interface {
    Fetch(ctx context.Context) ([]Item, error)
}

// In test file (internal/digest/digest_test.go):
type fakeProvider struct {
    items []Item
    err   error
}

func (f *fakeProvider) Fetch(_ context.Context) ([]Item, error) {
    return f.items, f.err
}
```

**What to mock:**
- External HTTP calls (use `httptest.NewServer` for HTTP handlers)
- Any `interface` parameter the function under test accepts
- File I/O when testing parsing logic (use `strings.NewReader` or `bytes.Buffer`)

**What NOT to mock:**
- The standard library itself
- Pure functions with no I/O
- The package under test's own internal helpers

**HTTP mocking:**
```go
func TestFetchFeed(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "testdata/azure_feed.xml")
    }))
    defer ts.Close()

    provider := NewAzureProvider(ts.URL)
    items, err := provider.Fetch(context.Background())
    // ...
}
```

## Fixtures and Factories

**Test Data:**
- Static fixtures in `testdata/` subdirectory alongside the test file
- `testdata/` is ignored by the Go build tool — safe to place any file type there
- Use real-world-shaped data (actual RSS XML structure, not minimal stubs)

**Location:**
- `internal/provider/testdata/azure_feed.xml` — sample Azure RSS feed
- `internal/<pkg>/testdata/` pattern per package as needed

**Loading fixtures:**
```go
data, err := os.ReadFile("testdata/azure_feed.xml")
if err != nil {
    t.Fatalf("read fixture: %v", err)
}
```

## Coverage

**Requirements:** No enforced minimum yet; meaningful coverage is the goal
- Parse logic and error paths must be tested
- Happy path + at least one error case per exported function

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

**Note:** `coverage.*` and `*.coverprofile` are `.gitignore`d — do not commit them.

## Test Types

**Unit Tests:**
- Scope: single function or method in isolation
- Location: co-located `_test.go` file
- All provider parsing logic, error handling paths, date formatting

**Integration Tests:**
- Not yet defined — out of scope for Phase 1
- Future: end-to-end fetch against live RSS endpoints (behind a build tag)

**E2E Tests:**
- Not used in Phase 1

**Build Tags (future pattern):**
```go
//go:build integration

package provider_test
```

## Common Patterns

**Async/context testing:**
```go
func TestFetchWithCancel(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // immediately cancelled

    _, err := provider.Fetch(ctx)
    if !errors.Is(err, context.Canceled) {
        t.Errorf("expected context.Canceled, got %v", err)
    }
}
```

**Error testing:**
```go
if err == nil {
    t.Fatal("expected error, got nil")
}
if !errors.Is(err, ErrSomeExpected) {
    t.Errorf("got %v, want %v", err, ErrSomeExpected)
}
```

**t.Helper() for shared assertion logic:**
```go
func assertEqual(t *testing.T, got, want string) {
    t.Helper() // makes failures point to the caller, not here
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

---

*Testing analysis: 2026-03-09 — derived from CLAUDE.md (authoritative source). No Go source files exist yet; this document reflects the prescribed testing patterns for all new code.*
