// Package azure provides an RSS-based provider for Azure service updates.
// It satisfies the provider.Provider interface defined in internal/provider.
//
// Architecture rule: this package has zero dependencies on other internal
// packages except internal/provider (for the RawUpdate type). It fetches
// and parses only — no classification, enrichment, or persistence.
package azure

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/engineeringclouds/upstream/internal/provider"
)

const feedURL = "https://www.microsoft.com/releasecommunications/api/v2/azure/rss"

// rssItem maps the XML structure of a single <item> element in the Azure RSS feed.
//
// Go pattern: struct tags control encoding/xml field mapping.
// The xml:"category" tag on a []string field captures all <category>
// elements — a single string would capture only the last one.
//
// These types are unexported (lowercase) because callers only receive
// []provider.RawUpdate — the RSS wire format is an implementation detail.
type rssItem struct {
	GUID        string   `xml:"guid"`
	Link        string   `xml:"link"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"` // kept as string; parseDate handles format variants
	Categories  []string `xml:"category"`
	// Atom namespace field: Go's encoding/xml requires the full namespace URI,
	// not the namespace prefix. xml:"a10:updated" is silently ignored —
	// the correct form uses the full URI "http://www.w3.org/2005/Atom".
	Updated string `xml:"http://www.w3.org/2005/Atom updated"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

// parseDate tries multiple RFC variants used by Azure RSS feeds.
// The live feed uses "Mon, 02 Jan 2006 15:04:05 Z" — a non-standard
// space-separated Z timezone not matched by time.RFC1123Z or time.RFC1123.
// All four formats are tried in order; an error is returned only if all fail.
func parseDate(s string) (time.Time, error) {
	formats := []string{
		"Mon, 02 Jan 2006 15:04:05 Z", // live Azure feed (space + Z)
		time.RFC1123Z,                 // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC1123,                  // "Mon, 02 Jan 2006 15:04:05 GMT"
		time.RFC3339,                  // "2006-01-02T15:04:05Z07:00" (a10:updated)
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("azure: parseDate: unrecognized format %q", s)
}

// parse decodes an Azure RSS feed from r and returns updates published after since.
// If since is the zero value, all items are returned.
// Items with unparseable dates are included with Published zero (not dropped)
// — a warning is logged via slog so the issue is visible without crashing.
func parse(r io.Reader, since time.Time) ([]provider.RawUpdate, error) {
	var feed rssFeed
	if err := xml.NewDecoder(r).Decode(&feed); err != nil {
		return nil, fmt.Errorf("azure: parse: decode XML: %w", err)
	}

	var updates []provider.RawUpdate
	for _, item := range feed.Channel.Items {
		pub, err := parseDate(item.PubDate)
		if err != nil {
			// Fallback to a10:updated (RFC3339 — parses cleanly via parseDate).
			pub, err = parseDate(item.Updated)
			if err != nil {
				// Log warning but include item with zero time — do not drop it.
				// Dropping silently would cause data loss that's hard to debug.
				slog.Warn("azure: unrecognized pubDate",
					"pubDate", item.PubDate,
					"guid", item.GUID,
				)
			}
		}

		// Apply since filter: skip items published at or before the threshold.
		// !since.IsZero() guards the zero-value case (meaning "no filter").
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
		updates = []provider.RawUpdate{} // always return empty slice, never nil
	}
	return updates, nil
}

// AzureProvider fetches Azure service updates from the Microsoft RSS feed.
// It has zero dependencies on other internal packages (architecture rule).
//
// Go pattern: pointer receiver (*AzureProvider) because the struct holds a
// reference to an http.Client. Value receivers are for small, stateless types;
// pointer receivers are used when the type holds meaningful state or resources.
type AzureProvider struct {
	client *http.Client
}

// New returns an AzureProvider ready to fetch. If client is nil, a default
// client with a 30-second timeout is used. Accepting *http.Client allows tests
// to inject a custom transport (e.g. a file-backed transport for fixtures)
// without making network calls.
func New(client *http.Client) *AzureProvider {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &AzureProvider{client: client}
}

// Name returns the canonical provider identifier.
func (p *AzureProvider) Name() string { return "azure" }

// Fetch retrieves updates from the Azure RSS feed published after since.
// Returns an empty (non-nil) slice when no updates match the filter.
//
// Error wrapping pattern: every error from internal/ packages uses
// fmt.Errorf("package: operation: %w", err). The %w verb preserves the
// original error so callers can inspect it with errors.Is() / errors.As().
// Never return bare (unwrapped) errors from internal packages.
func (p *AzureProvider) Fetch(ctx context.Context, since time.Time) ([]provider.RawUpdate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: fetch: build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("azure: fetch: do request: %w", err)
	}
	defer resp.Body.Close() // defer Close immediately after checking err from Do

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure: fetch: unexpected status %d", resp.StatusCode)
	}

	return parse(resp.Body, since)
}
