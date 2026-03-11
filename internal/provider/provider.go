// Package provider defines the Provider interface consumed by the fetch pipeline.
//
// Go interface placement note: interfaces are defined where they are CONSUMED,
// not where they are implemented. The azure sub-package satisfies this interface
// but never imports this package — it simply returns []RawUpdate to its callers,
// and the caller (e.g. cmd/upstream) wires the two together. This avoids import
// cycles and keeps the azure package dependency-free.
package provider

import (
	"context"
	"time"
)

// Provider is implemented by each cloud update source.
// Implementations must not classify, enrich, or persist — return raw data only.
//
// Keep this interface small (currently 2 methods). Resist adding methods;
// new behaviour should live in the pipeline, not the provider contract.
type Provider interface {
	// Name returns the canonical provider identifier ("azure", "aws", "kubernetes").
	Name() string

	// Fetch returns updates published after since.
	// Returns an empty slice (not nil) when no updates match.
	//
	// ctx is always the first parameter on any function that does I/O — this is
	// a Go convention that allows callers to cancel or time out the operation
	// (e.g. an HTTP request). Never ignore ctx: pass it to every downstream call.
	Fetch(ctx context.Context, since time.Time) ([]RawUpdate, error)
}

// RawUpdate is the normalised output of every provider.
// The classifier (Phase 4) enriches this into an Update — providers never classify.
//
// All string fields are populated from the RSS feed directly; no transformation
// is applied. Consumers must not assume a specific content format without
// inspecting ContentFormat.
type RawUpdate struct {
	// ProviderID is the canonical provider name, e.g. "azure".
	ProviderID string

	// ExternalID is a stable, unique identifier within the provider — the RSS <guid>.
	ExternalID string

	// Title is the item title from <title>.
	Title string

	// RawContent is the <description> from the RSS item. May contain HTML.
	RawContent string

	// ContentFormat indicates how to interpret RawContent: "html" or "text".
	ContentFormat string

	// SourceURL is the <link> from the RSS item.
	SourceURL string

	// Published is the parsed publication time from <pubDate>.
	Published time.Time

	// RawCategories holds all <category> elements from the RSS item, in document order.
	RawCategories []string

	// RawMeta holds provider-specific key/value extras not covered by the standard
	// fields above, e.g. {"a10:updated": "2026-03-04T21:15:02Z"} for Azure.
	RawMeta map[string]string
}
