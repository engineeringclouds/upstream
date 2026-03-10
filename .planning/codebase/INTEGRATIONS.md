# External Integrations

**Analysis Date:** 2026-03-09

## APIs & External Services

**Cloud Provider RSS Feeds (Phase 1 — active):**
- Azure Updates RSS — source of infrastructure change announcements
  - Endpoint: `https://azure.microsoft.com/en-us/updates/feed/` (standard public RSS)
  - SDK/Client: Go standard library `net/http` + `encoding/xml`
  - Auth: None (public feed)

**Cloud Provider RSS Feeds (future phases — not yet implemented):**
- AWS updates RSS feed
- Kubernetes changelog/release RSS

**AI Classification (future phases — not yet implemented):**
- Provider: Not yet selected
- Purpose: Classify and tag infrastructure updates
- Auth: API key via `UPSTREAM_` prefixed env var (pattern established, specific var not yet defined)

## Data Storage

**Databases:**
- None (Phase 1 has no persistence layer)
- Planned: Azure Cosmos DB (out of scope Phase 1)
  - Connection: env var (name not yet defined, will follow `UPSTREAM_` prefix)
  - Client: Not yet selected

**File Storage:**
- None

**Caching:**
- None

## Authentication & Identity

**Auth Provider:**
- None (Phase 1 has no auth)
- Planned: Custom or provider TBD (`internal/auth` package planned but out of scope Phase 1)

## Monitoring & Observability

**Error Tracking:**
- None

**Logs:**
- Standard library `log` package (assumed); sensitive values must never be logged per CLAUDE.md

## CI/CD & Deployment

**Hosting:**
- Planned: Azure (Bicep deployment — out of scope Phase 1)

**CI Pipeline:**
- Not yet configured

## Environment Configuration

**Required env vars:**
- None required for Phase 1 (public RSS feed, no auth)
- Future vars will follow `UPSTREAM_` prefix convention
- All vars parsed in `main.go` only; never read from inside packages

**Secrets location:**
- `.env` file (gitignored per `.gitignore`)

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- Planned: Email digest delivery (out of scope Phase 1)

---

*Integration audit: 2026-03-09*
