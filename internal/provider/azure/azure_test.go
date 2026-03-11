package azure

import (
	"os"
	"testing"
	"time"

	"github.com/engineeringclouds/upstream/internal/provider"
)

// TestParseDate verifies the multi-format date parser handles all Azure RSS
// pubDate variants. Table-driven tests use a struct slice so adding new cases
// is a one-line addition — this is idiomatic Go test style.
//
// Azure's live feed uses a non-standard format: "Wed, 04 Mar 2026 21:15:02 Z"
// (space before Z instead of +0000 or GMT). The standard time.RFC1123 and
// time.RFC1123Z layouts don't parse this — parseDate must handle it explicitly.
func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantUTC string // expected UTC string from t.UTC().String()
	}{
		{
			name:    "live feed format (space + Z)",
			input:   "Wed, 04 Mar 2026 21:15:02 Z",
			wantUTC: "2026-03-04 21:15:02 +0000 UTC",
		},
		{
			name:    "RFC1123Z with numeric offset",
			input:   "Wed, 04 Mar 2026 21:15:02 +0000",
			wantUTC: "2026-03-04 21:15:02 +0000 UTC",
		},
		{
			name:    "RFC1123 with GMT",
			input:   "Wed, 04 Mar 2026 21:15:02 GMT",
			wantUTC: "2026-03-04 21:15:02 +0000 UTC",
		},
		{
			name:    "RFC3339 (a10:updated fallback format)",
			input:   "2026-03-04T21:15:02Z",
			wantUTC: "2026-03-04 21:15:02 +0000 UTC",
		},
		{
			name:    "unrecognized format returns error",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty string returns error",
			input:   "",
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

// TestFetchFromFixture verifies that the real captured Azure RSS fixture parses
// into RawUpdate structs with all required fields populated.
//
// The fixture at testdata/feed.xml is a real capture from the Azure RSS feed
// (captured 2026-03-10). Fixture-based tests are preferred over synthetic data
// because they catch real-world format quirks that synthetic data masks.
func TestFetchFromFixture(t *testing.T) {
	f, err := os.Open("testdata/feed.xml")
	if err != nil {
		t.Fatalf("open testdata/feed.xml: %v", err)
	}
	defer f.Close()

	// parse is an unexported function in the azure package — tests in the same
	// package (package azure, not package azure_test) can access unexported symbols.
	updates, err := parse(f, time.Time{}) // zero since = return all items
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(updates) == 0 {
		t.Fatal("expected at least one update from fixture, got none")
	}

	for _, u := range updates {
		if u.ExternalID == "" {
			t.Errorf("update %q: missing ExternalID (guid)", u.Title)
		}
		if u.ProviderID != "azure" {
			t.Errorf("update %q: ProviderID = %q, want \"azure\"", u.Title, u.ProviderID)
		}
		if u.SourceURL == "" {
			t.Errorf("update %q: missing SourceURL", u.Title)
		}
		if u.Published.IsZero() {
			t.Errorf("update %q: Published is zero (date parsing failed)", u.Title)
		}
		if len(u.RawCategories) == 0 {
			t.Errorf("update %q: no RawCategories (category parsing failed)", u.Title)
		}
	}
}

// TestSinceFilter verifies that updates at or before the since time are excluded.
func TestSinceFilter(t *testing.T) {
	f, err := os.Open("testdata/feed.xml")
	if err != nil {
		t.Fatalf("open testdata/feed.xml: %v", err)
	}
	defer f.Close()

	// Use a far-future since time — should return no updates.
	farFuture := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	updates, err := parse(f, farFuture)
	if err != nil {
		t.Fatalf("parse with far-future since: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("with since=2099, expected 0 updates, got %d", len(updates))
	}
}

// TestRawUpdateFields verifies the specific field mappings from RSS XML to RawUpdate.
func TestRawUpdateFields(t *testing.T) {
	f, err := os.Open("testdata/feed.xml")
	if err != nil {
		t.Fatalf("open testdata/feed.xml: %v", err)
	}
	defer f.Close()

	updates, err := parse(f, time.Time{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(updates) == 0 {
		t.Fatal("no updates to inspect")
	}

	u := updates[0]
	// Verify the struct satisfies the RawUpdate contract — this also confirms
	// that the azure package correctly builds provider.RawUpdate values.
	var _ provider.RawUpdate = u // compile-time check: u must be a provider.RawUpdate
	if u.ContentFormat != "html" {
		t.Errorf("ContentFormat = %q, want \"html\"", u.ContentFormat)
	}
}

// TestProviderInterface verifies that *AzureProvider satisfies the Provider interface.
// This is a compile-time check — if AzureProvider is missing Name() or Fetch(),
// this file will not compile. No runtime assertion is needed.
func TestProviderInterface(t *testing.T) {
	var _ provider.Provider = &AzureProvider{} // compile-time interface check
	t.Log("AzureProvider satisfies provider.Provider interface")
}
