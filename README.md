# Upstream

AI-enhanced infrastructure change awareness platform — aggregating updates from Azure, AWS, and Kubernetes into personalized newsletter digests.

Phase 1 delivers a Go CLI that polls the Azure Updates RSS feed, parses it, and prints formatted output to stdout.

## Prerequisites

- Go 1.26.1+
- No external dependencies — Phase 1 uses stdlib only

## Build

```bash
go build -o upstream ./cmd/upstream
```

## Run

```bash
# Fetch all Azure updates
./upstream fetch

# Filter to updates published after a date
./upstream fetch --since 2026-01-01

# JSON output
./upstream fetch --format json

# Combine flags
./upstream fetch --since 2026-01-01 --format json
```

## Test

```bash
go test ./...
go vet ./...
gofmt -l .   # must produce no output
```

## Project Layout

```
cmd/upstream/            — CLI entrypoint (flag parsing, output formatting)
internal/provider/       — Provider interface and RawUpdate struct
internal/provider/azure/ — Azure RSS provider implementation
```
