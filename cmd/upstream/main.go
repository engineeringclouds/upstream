// cmd/upstream/main.go
//
// Upstream CLI — fetches cloud provider update feeds and prints them.
// This is the thin wiring layer: parse flags, call providers, format output.
// All business logic lives in internal/provider.
//
// Go pattern: cmd/ is the ONLY place allowed to call os.Exit or log.Fatal.
// Everything else returns errors up the call stack. main() is the error terminal.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/engineeringclouds/upstream/internal/provider"
	"github.com/engineeringclouds/upstream/internal/provider/azure"
)

func main() {
	// FlagSet for the "fetch" subcommand.
	// Go pattern: flag.FlagSet scopes flags to a subcommand. This lets us add
	// more subcommands later (e.g., "version", "config") without flag collisions.
	fetchCmd := flag.NewFlagSet("fetch", flag.ContinueOnError)
	sinceStr := fetchCmd.String("since", "", "filter updates after this date (YYYY-MM-DD)")
	format := fetchCmd.String("format", "text", "output format: text or json")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: upstream <command> [flags]")
		fmt.Fprintln(os.Stderr, "commands: fetch")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "fetch":
		if err := fetchCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "upstream:", err)
			os.Exit(1)
		}
		if err := runFetch(*sinceStr, *format); err != nil {
			fmt.Fprintln(os.Stderr, "upstream:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "upstream: unknown command %q\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "commands: fetch")
		os.Exit(1)
	}
}

// runFetch is extracted from main() so it can return an error rather than
// calling os.Exit directly — this makes the logic testable.
func runFetch(sinceStr, format string) error {
	var since time.Time
	if sinceStr != "" {
		t, err := time.Parse("2006-01-02", sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since date %q: expected YYYY-MM-DD", sinceStr)
		}
		since = t
	}

	ctx := context.Background()
	p := azure.New(nil) // nil uses the default 30s http.Client

	updates, err := p.Fetch(ctx, since)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	return output(os.Stdout, updates, format)
}

// output writes updates to w in the requested format.
// Separating output from main() allows unit tests to capture output.
func output(w io.Writer, updates []provider.RawUpdate, format string) error {
	switch format {
	case "text":
		return outputText(w, updates)
	case "json":
		return outputJSON(w, updates)
	default:
		return fmt.Errorf("unknown --format %q: use text or json", format)
	}
}

// outputText writes one block per update to w.
// Format:
//
//	Title: <title>
//	Date:  <published RFC3339>
//	URL:   <sourceURL>
//	Tags:  <category1>, <category2>
//	---
func outputText(w io.Writer, updates []provider.RawUpdate) error {
	for _, u := range updates {
		_, err := fmt.Fprintf(w,
			"Title: %s\nDate:  %s\nURL:   %s\nTags:  %s\n---\n",
			u.Title,
			u.Published.UTC().Format(time.RFC3339),
			u.SourceURL,
			strings.Join(u.RawCategories, ", "),
		)
		if err != nil {
			return fmt.Errorf("output: write: %w", err)
		}
	}
	return nil
}

// outputJSON writes a JSON array of updates to w.
// Uses json.NewEncoder for streaming output (no full in-memory buffer).
func outputJSON(w io.Writer, updates []provider.RawUpdate) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(updates); err != nil {
		return fmt.Errorf("output: json encode: %w", err)
	}
	return nil
}
