# Phase 1: Foundation - Research

**Researched:** 2026-03-10
**Domain:** Go CLI, RSS/XML parsing, standard library patterns
**Confidence:** HIGH

---

## Summary

Phase 1 is a greenfield Go CLI with zero external dependencies. The codebase has no existing `internal/` or `cmd/` directories — every file is new. The primary technical risk is the Azure RSS feed's non-standard `pubDate` format: the feed emits dates like `"Wed, 04 Mar 2026 21:15:02 Z"` where the timezone is a bare `Z` separated by a space — this is NOT matched by Go's `time.RFC1123Z` (which expects `-0700`) or `time.RFC1123` (which expects `GMT`). A custom multi-format `parseDate` helper is required.

The `Provider` interface, `RawUpdate` struct, and package layout decisions made here propagate to all 10 subsequent phases. Getting the interface signatures and error handling patterns correct in Phase 1 is cheaper than fixing them after 5 more packages depend on them. All Go error handling patterns established here — `fmt.Errorf("package: op: %w", err)` in `internal/`, `log.Fatal` only in `cmd/` — must be treated as immutable conventions, not stylistic preferences.

**Primary recommendation:** Implement `parseDate` with four format strings tried in order, covering the live feed's format first (`"Mon, 02 Jan 2006 15:04:05 Z"`), then fallbacks. Source `testdata/` from a real captured RSS response to get genuine malformed-date coverage rather than synthetic fixtures.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| REQ-001 | Azure RSS provider fetches `https://www.microsoft.com/releasecommunications/api/v2/azure/rss` and parses XML into typed `RawUpdate` structs | Live feed verified: XML structure, field names, and namespace confirmed |
| REQ-002 | Azure provider extracts: title, description, source URL, published date, raw category tags from RSS XML | Live feed: multiple `<category>` elements per item, `<link>`, `<title>`, `<description>`, `<pubDate>`, `<guid>` all present |
| REQ-003 | Azure provider accepts a `since time.Time` parameter and filters results to updates published after that time | parseDate helper required — live feed pubDate format is non-standard |
| REQ-004 | `Provider` interface defined in `internal/provider/provider.go` with `Name() string` and `Fetch(ctx, since) ([]RawUpdate, error)` | Interface-where-consumed pattern; defined at package root |
| REQ-005 | `RawUpdate` struct includes: ProviderID, Provider, Title, RawContent, ContentFormat, SourceURL, Published, RawCategories, RawMeta | Field names verified against live feed fields |
| REQ-006 | Provider implementations have zero dependencies on internal packages | Architecture rule: providers return []RawUpdate to caller; caller handles persistence |
| REQ-020 | `upstream fetch` command fetches and displays Azure updates to stdout | CLI entry point in `cmd/upstream/main.go` |
| REQ-021 | `--since` flag accepts a date string (e.g. `2024-01-01`) and filters updates | Parse flag as `time.Time` in main.go; pass to provider |
| REQ-022 | `--format` flag accepts `text` (default) and `json` output formats | `encoding/json` for JSON output; formatted text for default |
| REQ-023 | Text format shows: title, published date, URL, categories per update | Matches live feed fields: title, pubDate, link, category |
| REQ-024 | JSON format emits valid JSON array of update objects | `json.NewEncoder(os.Stdout).Encode(updates)` pattern |
| REQ-025 | Exit code 1 on errors; errors printed to stderr | `os.Exit(1)` only in `cmd/`; `fmt.Fprintln(os.Stderr, err)` |
| REQ-026 | `go build ./cmd/upstream` produces a working binary | Standard Go build; no cgo dependencies |
| REQ-110 | `go test ./...` passes with meaningful coverage of parsing logic | Table-driven tests; testdata/ fixture from real feed |
| REQ-111 | `go vet ./...` produces no output | Run as part of completion check |
| REQ-112 | All committed code passes `gofmt` (no diff) | `gofmt -l .` must produce no output |
| REQ-113 | All errors wrapped with context: `fmt.Errorf("package: operation: %w", err)` | Pattern established in Phase 1; must hold throughout all 11 phases |
| REQ-114 | No `os.Getenv()` calls inside packages — config injected via constructors from `main.go` | No env vars in Phase 1 scope, but pattern must be correct |
| REQ-115 | No sensitive values logged | No credentials in Phase 1 scope; still enforce log discipline |
| REQ-116 | Table-driven tests used for multiple input cases | Required especially for parseDate edge cases |
| REQ-117 | `testdata/` fixtures sourced from real feed responses (not synthetic) | Capture live feed response; commit as fixture file |
</phase_requirements>

---

## Standard Stack

### Core (Phase 1 — stdlib only)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/xml` | stdlib | Azure RSS/XML parsing | Exact field control via struct tags; teaches the Go pattern before gofeed is introduced in Phase 9 |
| `net/http` | stdlib | HTTP fetch of RSS feed | No external dependency; sufficient for a single GET with timeout |
| `encoding/json` | stdlib | JSON output format | Standard serialization; `json.NewEncoder` streams to stdout without buffering entire output |
| `flag` | stdlib | CLI flag parsing | Standard Go flag package; `--since` and `--format` are simple string flags |
| `time` | stdlib | Date parsing and filtering | `time.Parse` with multiple format strings for pubDate |
| `fmt` | stdlib | Error wrapping and output | `fmt.Errorf("azure: parse: %w", err)` is the required error pattern |
| `os` | stdlib | Exit code and stderr | `os.Exit(1)` in `cmd/` only; `os.Stderr` for error output |
| `context` | stdlib | Context propagation | First parameter on all functions that do I/O; passed to `http.NewRequestWithContext` |
| `testing` | stdlib | Test framework | Table-driven tests; no third-party test framework |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `encoding/xml` | `github.com/mmcdole/gofeed` | gofeed handles multi-format feeds but hides field structure; encoding/xml gives exact control for Phase 1 and teaches the pattern |
| `flag` | `github.com/spf13/cobra` | cobra adds sub-command structure; overkill for two flags; defer until Phase 2+ when more commands exist |
| `time.Parse` with format strings | Third-party date parsing | No justification for an external dep; the feed uses a predictable format once the `Z` issue is understood |

**Installation:** No installation needed — stdlib only.

---

## Architecture Patterns

### Recommended Project Structure

```
upstream/
├── cmd/
│   └── upstream/
│       └── main.go              # CLI wiring: parse flags, call provider, format output
├── internal/
│   └── provider/
│       ├── provider.go          # Provider interface + RawUpdate struct
│       └── azure/
│           ├── azure.go         # AzureProvider implementation
│           ├── azure_test.go    # Table-driven tests
│           └── testdata/
│               └── feed.xml     # Real captured Azure RSS response
├── go.mod
├── go.sum                       # empty at Phase 1 (no external deps)
└── README.md
```

**Rationale for structure:**
- `internal/provider/provider.go` defines the interface where it is consumed. Azure implementation lives in the `azure/` sub-package and never imports the `provider` parent package — it just satisfies the interface.
- `testdata/` lives inside the `azure/` package directory so `go test` can reference it with `testdata/feed.xml` relative to the test file.
- `cmd/upstream/main.go` is the only file that calls `log.Fatal` or `os.Exit`. It wires providers to formatters.

### Pattern 1: Provider Interface

**What:** Small interface at `internal/provider/provider.go`. Azure implementation in sub-package.
**When to use:** Always. New providers (Phase 9 AWS, Phase 10 Kubernetes) are drop-in additions.

```go
// internal/provider/provider.go

// Package provider defines the Provider interface consumed by the fetch pipeline.
package provider

import (
    "context"
    "time"
)

// Provider is implemented by each cloud update source.
// Implementations must not classify, enrich, or persist — return raw data only.
type Provider interface {
    // Name returns the canonical provider identifier ("azure", "aws", "kubernetes").
    Name() string

    // Fetch returns updates published after since.
    // Returns an empty slice (not nil) when no updates match.
    Fetch(ctx context.Context, since time.Time) ([]RawUpdate, error)
}

// RawUpdate is the normalized output of every provider.
type RawUpdate struct {
    ProviderID    string    // canonical provider name ("azure")
    ExternalID    string    // unique within provider — RSS <guid>
    Title         string
    RawContent    string    // <description> from RSS
    ContentFormat string    // "html" or "text"
    SourceURL     string    // <link> from RSS
    Published     time.Time // parsed from <pubDate>
    RawCategories []string  // all <category> elements from the item
    RawMeta       map[string]string // provider-specific extras (e.g., a10:updated)
}
```

### Pattern 2: XML Struct Tags for RSS Parsing

**What:** Map RSS XML elements to Go struct fields using `xml:` tags. Multiple `<category>` elements map to a slice with `xml:"category"`.

**Live feed structure (verified 2026-03-10):**

```xml
<rss xmlns:a10="http://www.w3.org/2005/Atom" version="2.0">
  <channel>
    <item>
      <guid isPermaLink="false">558102</guid>
      <link>https://azure.microsoft.com/updates?id=558102</link>
      <category>Management and governance</category>
      <category>Azure Policy</category>
      <title>Retirement: Azure Policy faster enforcement...</title>
      <description>Over the years...</description>
      <pubDate>Wed, 04 Mar 2026 21:15:02 Z</pubDate>
      <a10:updated>2026-03-04T21:15:02Z</a10:updated>
    </item>
  </channel>
</rss>
```

```go
// internal/provider/azure/azure.go

type rssItem struct {
    GUID        string   `xml:"guid"`
    Link        string   `xml:"link"`
    Title       string   `xml:"title"`
    Description string   `xml:"description"`
    PubDate     string   `xml:"pubDate"`    // kept as string; parseDate handles it
    Categories  []string `xml:"category"`
    // Atom updated field via namespace
    Updated     string   `xml:"http://www.w3.org/2005/Atom updated"`
}

type rssChannel struct {
    Items []rssItem `xml:"item"`
}

type rssFeed struct {
    Channel rssChannel `xml:"channel"`
}
```

### Pattern 3: Multi-Format Date Parsing (CRITICAL)

**What:** The live Azure RSS feed uses `"Wed, 04 Mar 2026 21:15:02 Z"` — a bare space-separated `Z` for timezone. This does NOT match `time.RFC1123Z` (`-0700`) or `time.RFC1123` (`GMT`). A custom format string is required as the first attempt.

**Verified live format:** `"Mon, 02 Jan 2006 15:04:05 Z"` (Go reference time, with literal space-Z)

```go
// internal/provider/azure/azure.go

// parseDate attempts multiple date formats in order.
// The live Azure RSS feed uses a non-standard "Z" timezone suffix
// (space-separated, not RFC1123Z or RFC1123). All known format variants
// are tried before returning an error.
func parseDate(s string) (time.Time, error) {
    formats := []string{
        "Mon, 02 Jan 2006 15:04:05 Z",    // live Azure feed format (space + Z)
        time.RFC1123Z,                      // "Mon, 02 Jan 2006 15:04:05 -0700"
        time.RFC1123,                       // "Mon, 02 Jan 2006 15:04:05 GMT"
        time.RFC3339,                       // "2006-01-02T15:04:05Z07:00" (a10:updated)
    }
    for _, f := range formats {
        if t, err := time.Parse(f, s); err == nil {
            return t, nil
        }
    }
    return time.Time{}, fmt.Errorf("azure: parseDate: unrecognized format %q", s)
}
```

**Note on the `a10:updated` field:** If `pubDate` fails all formats, fall back to `a10:updated` (which is RFC3339 and parses cleanly). Log a warning with the raw string when both fail.

### Pattern 4: HTTP Fetch with Context and Timeout

```go
// internal/provider/azure/azure.go

const feedURL = "https://www.microsoft.com/releasecommunications/api/v2/azure/rss"

// AzureProvider fetches the Azure service updates RSS feed.
// It has zero dependencies on other internal packages.
type AzureProvider struct {
    client *http.Client
}

// New returns an AzureProvider with a 30-second timeout.
// The http.Client is injected to allow test overrides.
func New(client *http.Client) *AzureProvider {
    if client == nil {
        client = &http.Client{Timeout: 30 * time.Second}
    }
    return &AzureProvider{client: client}
}

func (p *AzureProvider) Name() string { return "azure" }

func (p *AzureProvider) Fetch(ctx context.Context, since time.Time) ([]provider.RawUpdate, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
    if err != nil {
        return nil, fmt.Errorf("azure: fetch: build request: %w", err)
    }

    resp, err := p.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("azure: fetch: do request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("azure: fetch: unexpected status %d", resp.StatusCode)
    }

    return parse(resp.Body, since)
}
```

### Pattern 5: CLI Structure in main.go

```go
// cmd/upstream/main.go

func main() {
    sinceStr := flag.String("since", "", "filter updates after this date (YYYY-MM-DD)")
    format   := flag.String("format", "text", "output format: text or json")
    flag.Parse()

    var since time.Time
    if *sinceStr != "" {
        t, err := time.Parse("2006-01-02", *sinceStr)
        if err != nil {
            fmt.Fprintln(os.Stderr, "upstream: invalid --since date:", err)
            os.Exit(1)
        }
        since = t
    }

    ctx := context.Background()
    p := azure.New(nil)

    updates, err := p.Fetch(ctx, since)
    if err != nil {
        fmt.Fprintln(os.Stderr, "upstream: fetch:", err)
        os.Exit(1)
    }

    if err := output(os.Stdout, updates, *format); err != nil {
        fmt.Fprintln(os.Stderr, "upstream: output:", err)
        os.Exit(1)
    }
}
```

### Pattern 6: Table-Driven Tests with testdata Fixture

```go
// internal/provider/azure/azure_test.go

func TestParseDate(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
        wantUTC string // expected UTC representation
    }{
        {
            name:    "live feed format (space-Z)",
            input:   "Wed, 04 Mar 2026 21:15:02 Z",
            wantUTC: "2026-03-04 21:15:02 +0000 UTC",
        },
        {
            name:    "RFC1123Z",
            input:   "Wed, 04 Mar 2026 21:15:02 +0000",
            wantUTC: "2026-03-04 21:15:02 +0000 UTC",
        },
        {
            name:    "RFC3339 (a10:updated fallback)",
            input:   "2026-03-04T21:15:02Z",
            wantUTC: "2026-03-04 21:15:02 +0000 UTC",
        },
        {
            name:    "unrecognized format",
            input:   "not a date",
            wantErr: true,
        },
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            got, err := parseDate(tc.input)
            if (err != nil) != tc.wantErr {
                t.Fatalf("parseDate(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
            }
            if !tc.wantErr && got.UTC().String() != tc.wantUTC {
                t.Errorf("parseDate(%q) = %v, want %v", tc.input, got.UTC(), tc.wantUTC)
            }
        })
    }
}

func TestFetchFromFixture(t *testing.T) {
    // Use real captured feed as HTTP response body
    f, err := os.Open("testdata/feed.xml")
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()

    updates, err := parse(f, time.Time{}) // zero since = return all
    if err != nil {
        t.Fatalf("parse: %v", err)
    }

    if len(updates) == 0 {
        t.Fatal("expected updates from fixture, got none")
    }

    // Verify a known item from the fixture
    for _, u := range updates {
        if u.ExternalID == "" {
            t.Error("update missing ExternalID (guid)")
        }
        if u.Published.IsZero() {
            t.Errorf("update %q has zero Published time", u.Title)
        }
        if len(u.RawCategories) == 0 {
            t.Errorf("update %q has no categories", u.Title)
        }
    }
}
```

### Anti-Patterns to Avoid

- **`log.Fatal` in `internal/`:** Terminates the process instead of returning an error. Only `cmd/upstream/main.go` calls `log.Fatal`. Every function in `internal/` returns `error`.
- **Swallowing errors with `_ = someFunc()`:** All errors must be handled or returned with context.
- **`resp.Body` not closed:** Always `defer resp.Body.Close()` immediately after checking the error on `client.Do`.
- **Classifying inside the provider:** The Azure RSS title prefixes (`[Launched]`, `Retirement:`) are tempting to parse into classification — don't. Surface them as `RawCategories`. Classification is Phase 4.
- **`math/rand` for anything random:** No random values in Phase 1, but the convention is `crypto/rand` when needed.
- **Bare `return` after printing error:** Print to stderr AND return the error to the caller. Don't do both print and return nil.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| XML parsing | Custom string splitting or regex | `encoding/xml` | Handles namespaces, CDATA, entity decoding, and streaming; regex on XML is fragile |
| JSON output | Manual string concatenation | `encoding/json` | Handles escaping, nested structures, and streaming |
| HTTP client | Raw TCP connection | `net/http` | Connection pooling, redirect handling, timeout management |
| Date formatting | Custom format logic | `time.Format` with reference time | Go's format uses a reference time — use it consistently |

**Key insight:** Phase 1 is entirely stdlib. The only "don't hand-roll" risk is date parsing — developers sometimes write ad-hoc string splitting for dates. Use `time.Parse` with the correct format strings.

---

## Common Pitfalls

### Pitfall 1: pubDate "Z" Format Not Matched by Standard Go Constants

**What goes wrong:** `time.Parse(time.RFC1123Z, "Wed, 04 Mar 2026 21:15:02 Z")` returns an error. `time.RFC1123Z` expects `-0700` (e.g., `+0000`), not a bare `Z`. The item's `Published` field stays zero-valued. The `--since` filter then either returns all items or no items depending on comparison direction.

**Why it happens:** Developers assume RFC1123Z covers all RFC 2822 variants. The `Z` timezone suffix with a preceding space is technically non-standard — the live Azure feed uses it consistently.

**How to avoid:** Use the custom format `"Mon, 02 Jan 2006 15:04:05 Z"` as the first format string in `parseDate`. Verify with `TestParseDate` before writing any filter logic.

**Warning signs:** `--since 2024-01-01` returns all items (including old ones) or zero items on a live feed.

### Pitfall 2: Multiple `<category>` Elements Require a Slice Field

**What goes wrong:** A single `Category string` field with `xml:"category"` captures only the last `<category>` element. Items have 3-6 categories — the rest are silently dropped.

**Why it happens:** RSS examples in tutorials typically show one category.

**How to avoid:** Use `Categories []string `xml:"category"`` — `encoding/xml` populates slices correctly when multiple elements share the same name.

**Warning signs:** `RawCategories` always has length 1 even for items with multiple `<category>` entries.

### Pitfall 3: Atom Namespace Field Requires Full Namespace URI in Tag

**What goes wrong:** `xml:"a10:updated"` does NOT work — Go's `encoding/xml` requires the full namespace URI, not the prefix. The field is silently ignored.

**How to avoid:** Use `xml:"http://www.w3.org/2005/Atom updated"`.

### Pitfall 4: Error Handling Patterns Established in Phase 1 Persist Forever

**What goes wrong:** One `log.Fatal` in `internal/` or one swallowed error in Phase 1 sets the precedent. By Phase 5, the pattern appears in 30 files.

**How to avoid:** Every function in `internal/` returns `error`. `main.go` is the only terminal. Error wrapping format: `fmt.Errorf("azure: fetch: do request: %w", err)`. Verify with `go vet ./...` before merging.

### Pitfall 5: Response Body Not Closed on Error Path

**What goes wrong:** Early return before `defer resp.Body.Close()` leaks the connection. The HTTP client's connection pool fills up and subsequent requests fail.

**How to avoid:** `defer resp.Body.Close()` immediately after checking `err` from `client.Do`, not after checking the status code.

### Pitfall 6: go.mod Version Mismatch

**What goes wrong:** The current `go.mod` says `go 1.25.6`. The project MEMORY and STACK research specify Go 1.26.1 as the target. Using a lower version in go.mod is fine for Phase 1 (stdlib only) but needs to be aligned before Phase 2+ external dependencies are added.

**How to avoid:** Update `go.mod` to `go 1.26.1` in the first task of Phase 1. This is a one-line change with no other impact.

---

## Code Examples

### Verified: XML Struct for Azure RSS Feed

```go
// Source: live feed inspection 2026-03-10
// Namespace: xmlns:a10="http://www.w3.org/2005/Atom"

type rssItem struct {
    GUID        string   `xml:"guid"`
    Link        string   `xml:"link"`
    Title       string   `xml:"title"`
    Description string   `xml:"description"`
    PubDate     string   `xml:"pubDate"`
    Categories  []string `xml:"category"`
    Updated     string   `xml:"http://www.w3.org/2005/Atom updated"`
}

type rssChannel struct {
    Items []rssItem `xml:"item"`
}

type rssFeed struct {
    Channel rssChannel `xml:"channel"`
}
```

### Verified: XML Decoder Usage

```go
// Source: go.dev/pkg/encoding/xml
func parse(r io.Reader, since time.Time) ([]provider.RawUpdate, error) {
    var feed rssFeed
    if err := xml.NewDecoder(r).Decode(&feed); err != nil {
        return nil, fmt.Errorf("azure: parse: decode XML: %w", err)
    }

    var updates []provider.RawUpdate
    for _, item := range feed.Channel.Items {
        pub, err := parseDate(item.PubDate)
        if err != nil {
            // Try a10:updated as fallback
            pub, err = parseDate(item.Updated)
            if err != nil {
                // Log warning but do not drop the item
                // Use zero time; filter logic will treat as unknown
                fmt.Fprintf(os.Stderr, "azure: parse: unrecognized pubDate %q, skipping date filter\n", item.PubDate)
            }
        }

        // Apply since filter: skip if published at or before since
        if !since.IsZero() && !pub.After(since) {
            continue
        }

        updates = append(updates, provider.RawUpdate{
            ProviderID:    "azure",
            ExternalID:    item.GUID,
            Title:         item.Title,
            RawContent:    item.Description,
            ContentFormat: "html",
            SourceURL:     item.Link,
            Published:     pub,
            RawCategories: item.Categories,
        })
    }

    if updates == nil {
        updates = []provider.RawUpdate{} // return empty slice, not nil
    }
    return updates, nil
}
```

### Verified: JSON Output Pattern

```go
// Source: go.dev/pkg/encoding/json
func outputJSON(w io.Writer, updates []provider.RawUpdate) error {
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    if err := enc.Encode(updates); err != nil {
        return fmt.Errorf("output: json encode: %w", err)
    }
    return nil
}
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| `github.com/mmcdole/gofeed` for all feeds | `encoding/xml` for Azure (Phase 1); gofeed deferred to Phase 9 | Fewer dependencies; teaches struct-tag XML parsing |
| `github.com/spf13/cobra` from day 1 | `flag` stdlib for Phase 1 CLI | No external dep; cobra introduced if/when sub-command complexity warrants it |
| `log.Fatal` anywhere convenient | `log.Fatal` only in `cmd/`; `error` returns everywhere else | Pattern that survives to Phase 11 if established correctly in Phase 1 |
| Single `time.Parse` call with RFC1123Z | Multi-format `parseDate` helper | Handles the live Azure feed's non-standard `Z` timezone suffix |

**Deprecated/outdated:**
- `ioutil.ReadAll`: replaced by `io.ReadAll` in Go 1.16+. Use `io.ReadAll` if bulk reading is needed (though streaming via `xml.NewDecoder` is preferred).
- `http.Get(url)`: use `http.NewRequestWithContext` to support context cancellation and timeouts.

---

## Open Questions

1. **go.mod version alignment**
   - What we know: `go.mod` currently says `go 1.25.6`; the project target is Go 1.26.1
   - What's unclear: Whether 1.25.6 is intentional or an artifact of `go mod init`
   - Recommendation: Update to `go 1.26.1` in Wave 0 of Phase 1; no external deps means no compatibility risk

2. **testdata fixture freshness**
   - What we know: The live feed has ~100 items spanning recent months
   - What's unclear: Whether any historical items in the feed have different pubDate formats
   - Recommendation: Capture the live feed at task time and commit as `testdata/feed.xml`; add at least one synthetic malformed-date item to test the error path

3. **`--since` zero value behavior**
   - What we know: `since time.Time{}` (zero) means "return all items"
   - What's unclear: Whether the CLI should require `--since` or make it optional
   - Recommendation: Make `--since` optional; when absent, return all items in the feed (the feed is already time-bounded to recent items by Microsoft)

4. **stderr vs structured logging**
   - What we know: Phase 1 has no logger; `fmt.Fprintln(os.Stderr, ...)` is sufficient
   - What's unclear: Whether to introduce `log/slog` (stdlib, Go 1.21+) now for structured output
   - Recommendation: Use `log/slog` from Phase 1 for the warning in `parseDate`. It is stdlib, structured, and aligns with how Phase 2+ will need to log. Configure it to write to stderr with level INFO as default.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `testing` (stdlib) |
| Config file | none — `go test` convention |
| Quick run command | `go test ./internal/provider/azure/...` |
| Full suite command | `go test ./...` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-001 | Fetch and parse real Azure RSS feed | integration | `go test ./internal/provider/azure/... -run TestFetchFromFixture` | Wave 0 |
| REQ-002 | Extract all required fields from RSS XML | unit | `go test ./internal/provider/azure/... -run TestParse` | Wave 0 |
| REQ-003 | Filter by `since` timestamp correctly | unit | `go test ./internal/provider/azure/... -run TestSinceFilter` | Wave 0 |
| REQ-004 | Provider interface satisfied by AzureProvider | compile-time | `go build ./...` | Wave 0 |
| REQ-005 | RawUpdate fields populated from feed | unit | `go test ./internal/provider/azure/... -run TestRawUpdateFields` | Wave 0 |
| REQ-022 | `--format json` emits valid JSON array | unit | `go test ./cmd/upstream/... -run TestOutputJSON` | Wave 0 |
| REQ-023 | `--format text` emits readable output | unit | `go test ./cmd/upstream/... -run TestOutputText` | Wave 0 |
| REQ-025 | Exit code 1 on error | manual | run with invalid `--format` value | — |
| REQ-110 | parseDate handles all format variants + malformed | unit | `go test ./internal/provider/azure/... -run TestParseDate` | Wave 0 |
| REQ-111 | go vet clean | static | `go vet ./...` | — |
| REQ-112 | gofmt clean | static | `gofmt -l .` | — |
| REQ-113 | Errors wrapped with fmt.Errorf | code review | `grep -r 'errors.New\|fmt.Errorf' internal/` | — |
| REQ-116 | Table-driven tests used | code review | `go test -v ./...` shows subtests | Wave 0 |
| REQ-117 | testdata from real feed | code review | `ls internal/provider/azure/testdata/` | Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/provider/azure/... && go vet ./...`
- **Per wave merge:** `go test ./... && go vet ./... && gofmt -l .`
- **Phase gate:** Full suite green + `gofmt -l .` produces no output before marking Phase 1 complete

### Wave 0 Gaps

- [ ] `internal/provider/azure/testdata/feed.xml` — real captured Azure RSS feed (covers REQ-001, REQ-117)
- [ ] `internal/provider/azure/azure_test.go` — table-driven tests for parseDate, parse, Fetch (covers REQ-110, REQ-116)
- [ ] `internal/provider/provider.go` — interface and RawUpdate struct must exist before implementation (covers REQ-004, REQ-005)
- [ ] `go.mod` — update `go 1.25.6` to `go 1.26.1`

---

## Sources

### Primary (HIGH confidence)

- Live Azure RSS feed (`https://www.microsoft.com/releasecommunications/api/v2/azure/rss`) — inspected 2026-03-10; XML structure, pubDate format, category structure, Atom namespace all verified
- `go.dev/pkg/encoding/xml` — struct tag syntax, namespace handling, Decode pattern
- `go.dev/pkg/time` — RFC1123Z, RFC1123, RFC3339 format constants; Parse semantics
- `.planning/research/STACK.md` — stack decisions verified 2026-03-09 (HIGH confidence)
- `.planning/research/ARCHITECTURE.md` — provider interface pattern, project structure
- `.planning/research/PITFALLS.md` — RSS date parsing pitfall (confirmed against live feed)

### Secondary (MEDIUM confidence)

- `go.dev/wiki/CommonMistakes` — error handling patterns, defer placement
- `100go.co` — Go beginner error handling anti-patterns

### Tertiary (LOW confidence)

- None for Phase 1 domain — all findings are either verified against live feed or official stdlib docs.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only; well-documented
- Architecture: HIGH — patterns from official go.dev canonical layout; verified against prior project research
- pubDate parsing: HIGH — live feed inspected; format `"Mon, 02 Jan 2006 15:04:05 Z"` confirmed empirically
- XML struct tags: HIGH — live feed field names and namespaces verified
- Pitfalls: HIGH — date format pitfall confirmed against live feed; error pattern pitfalls from authoritative Go sources

**Research date:** 2026-03-10
**Valid until:** 2026-06-10 (stable stdlib; feed format may drift — re-verify pubDate before starting if > 30 days elapsed)
