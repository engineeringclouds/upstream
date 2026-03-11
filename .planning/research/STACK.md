# Stack Research

**Domain:** Go-based infrastructure change awareness newsletter platform (RSS aggregation, AI classification, email digest delivery, Azure-native)
**Researched:** 2026-03-09
**Confidence:** HIGH (all versions verified against pkg.go.dev and official docs)

---

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.26.1 | Primary language | Current stable release (2026-03-05). Two supported releases at any time — target latest. Go 1.22+ ServeMux improvements matter for routing decision. |
| `encoding/xml` (stdlib) | stdlib | Azure RSS/Atom XML parsing | Phase 1 scope explicitly calls for this. Zero dependencies, full type safety via struct tags, sufficient for well-formed feeds like Azure Updates. |
| `net/http` (stdlib) | stdlib | HTTP server and outbound fetch | Go 1.22+ ServeMux supports method and path variable routing natively. No external dependency needed for Phase 2 web UI if routing is simple. |
| `github.com/go-chi/chi/v5` | v5.2.5 | HTTP router (Phase 2+) | When middleware grouping is needed (auth middleware, per-route logging, request ID injection), chi pays off. 100% `net/http` compatible. Zero external dependencies of its own. Upgrade from stdlib ServeMux with no interface changes. |
| `github.com/a-h/templ` | v0.3.1001 | Type-safe HTML templating | The standard Go + htmx pairing. Compiles `.templ` files to Go code — template errors caught at compile time, not runtime. Works with Go's `http.Handler` directly. Required for Phases 2–3 web UI. Note: pre-v1, but production-stable. |
| htmx | 2.x (CDN) | Partial page updates | Delivered via CDN in `<script>` tag — no build toolchain. Pairs with templ for the subscriber preferences dashboard. No Node.js required. |

### Azure SDK (Go)

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `github.com/Azure/azure-sdk-for-go/sdk/azidentity` | v1.13.1 | Authentication (Managed Identity, DefaultAzureCredential) | `DefaultAzureCredential` works in Container Apps (managed identity) and local dev (Azure CLI credential) without code changes. No secrets in config. Required by all other Azure SDKs. |
| `github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos` | v1.4.2 | Cosmos DB document store | Official Microsoft SDK. Stable v1. Serverless Cosmos DB suits pay-per-request model. Partition key design is critical — partition by `providerID` (e.g., `azure`, `aws`) to keep cross-partition queries rare. |
| `github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/v2` | v2.0.1 | Queue Storage (feed polling → AI pipeline decoupling) | Official Microsoft SDK. v2 stable as of Jan 2026. Decouples the RSS poller (Container Apps Job) from the AI classifier (separate Job). Prevents classification latency blocking ingestion. |

### AI Classification

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| `github.com/anthropics/anthropic-sdk-go` | v1.26.0 | Claude API client | Official Anthropic Go SDK. Requires Go 1.22+. Supports streaming, tool use, and structured output. Use `claude-3-5-haiku` for classification (cost-optimized) and `claude-3-5-sonnet` for editorial drafting. Defaults to `ANTHROPIC_API_KEY` env var. |

### Email Delivery

| Approach | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| ACS REST API via `net/http` | — | Magic link auth emails + weekly digest delivery | No official Go SDK exists for Azure Communication Services Email (confirmed March 2026). Use `net/http` with HMAC-SHA256 request signing. Microsoft provides a detailed tutorial for Go HMAC header construction. ACS REST API version `2025-09-01` is current. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/mmcdole/gofeed` | v1.3.0 | Multi-format RSS/Atom/JSON feed parser | Use for AWS (Phase 9) and Kubernetes blog (Phase 10) feeds where the feed format is less predictable. For Azure RSS (Phase 1), `encoding/xml` with a custom struct is preferred — it gives exact field control and teaches the pattern. |
| `golang.org/x/net/html` (stdlib extended) | stdlib | HTML parsing | For Kubernetes CHANGELOG markdown-to-structured parsing in Phase 10. Prefer `encoding/xml` or direct string processing first; reach for this only if needed. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `go vet` | Static analysis | Run as part of CI. Catches real bugs — not optional. |
| `gofmt` | Code formatting | Non-negotiable. Run on save in VS Code. |
| `templ generate` | Compile `.templ` files to Go | Run via `go tool templ` with Go 1.24+ tool directive. Add to `go.mod` as a tool dependency. |
| `air` or `reflex` | Live reload during dev | Run `templ generate --watch` alongside the Go server watcher. Templ docs recommend this workflow. |
| `go test -race` | Data race detection | Especially important once the queue consumer runs concurrently. Add to CI. |

### Infrastructure (Bicep / Azure)

| Component | Purpose | Notes |
|-----------|---------|-------|
| Azure Container Apps | Host the web server | Scale-to-zero. Built-in HTTPS. System-assigned managed identity for SDK auth. |
| Azure Container Apps Jobs (scheduled) | RSS feed polling | Cron-triggered job, one per provider. Outputs to Queue Storage. |
| Azure Container Apps Jobs (event-driven) | AI classification pipeline | Queue-triggered by KEDA scaler on Queue Storage depth. |
| Azure Container Registry | Container image storage | Required by Container Apps. Bicep module: `Microsoft.ContainerRegistry/registries`. |
| Azure Cosmos DB (NoSQL, serverless) | Primary data store | Serverless billing. Two containers: `updates` (partitioned by `providerId`) and `subscribers` (partitioned by `id`). |
| Azure Queue Storage | Ingestion → classification decoupling | Two queues: `raw-updates` (poller → classifier) and `digest-queue` (classifier → delivery). |
| Azure Communication Services | Email delivery + magic links | No Go SDK — use REST API directly. Domain verification required before sending. |
| Bicep | IaC | All infrastructure defined in `deploy/` directory. Use `az deployment group create` for deployment. |

---

## Installation

```bash
# Phase 1 — stdlib only, no deps needed

# Phase 2+ web UI
go get -u github.com/go-chi/chi/v5
go get -u github.com/a-h/templ

# Add templ as a tool dependency (Go 1.24+)
# In go.mod: tool github.com/a-h/templ/cmd/templ

# Azure SDKs
go get -u github.com/Azure/azure-sdk-for-go/sdk/azidentity
go get -u github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos
go get -u github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/v2

# AI
go get -u 'github.com/anthropics/anthropic-sdk-go@v1.26.0'

# Phase 9+ (AWS/multi-format feeds)
go get -u github.com/mmcdole/gofeed
```

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `encoding/xml` (stdlib) | `github.com/mmcdole/gofeed` | Use gofeed when you cannot predict the feed format (AWS, K8s blog). For Azure in Phase 1, stdlib gives more control and teaches Go patterns. |
| `net/http` ServeMux | `github.com/go-chi/chi/v5` | Use stdlib ServeMux for Phase 1 (no web UI). Introduce chi in Phase 2 when middleware grouping (auth, logging) justifies the dependency. |
| `templ` | `html/template` (stdlib) | Use stdlib templates only if you want zero toolchain dependencies and can tolerate runtime template errors. Templ's compile-time checking is worth the tool dependency for a multi-phase project. |
| ACS via REST API | SendGrid, Mailgun, Resend | Use a third-party email service only if ACS proves unreliable or pricing becomes prohibitive. The project is Azure-native — ACS keeps the cost model unified and avoids a second vendor. |
| Claude API (direct) | Azure OpenAI | Switch to Azure OpenAI if data residency or enterprise agreement pricing becomes a requirement. The anthropic-sdk-go API is clean enough that the switch is a ~1-day refactor. |
| Bicep | Terraform | Terraform if portability across clouds were a goal. It is explicitly not a goal. Bicep is the IaC standard at Engineering Clouds. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `github.com/Azure/azure-storage-queue-go` | Deprecated. The old SDK was marked deprecated — Microsoft migrated everything to `azure-sdk-for-go`. | `github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/v2` |
| `github.com/vippsas/go-cosmosdb` | Community SDK, not Microsoft-maintained. No managed identity support. | `github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos` |
| `github.com/unfunco/anthropic-sdk-go` | Community SDK, predates the official one. | `github.com/anthropics/anthropic-sdk-go` (official) |
| `github.com/gin-gonic/gin` | Heavier framework with its own context type that diverges from `net/http`. Harder to compose with standard middleware. Chi stays `http.Handler`-compatible. | `github.com/go-chi/chi/v5` |
| `database/sql` with a SQL driver | Cosmos DB is a NoSQL document store. Using a SQL wrapper obscures the partition key model and costs extra latency. | `azcosmos` directly |
| Environment variables read inside packages | Violates the project's architecture rule. Packages accept config via constructors, never `os.Getenv`. | Parse env vars in `cmd/upstream/main.go`, inject via constructors. |
| Azure AD B2C for auth | Over-engineered for v1. Requires app registration, policy configuration, redirect flows. | Magic link via ACS email (Phase 3) |

---

## Stack Patterns by Phase

**Phase 1 (Azure CLI — Foundation):**
- `encoding/xml` + `net/http` for fetching and parsing
- Zero external dependencies — pure stdlib
- CLI entry point in `cmd/upstream/main.go`

**Phase 2 (Azure Deployment):**
- Introduce `chi` for HTTP routing
- Introduce `templ` for HTML generation
- `azcosmos` for persistence
- `azidentity` for all Azure auth
- Bicep modules in `deploy/`

**Phase 3 (Auth + Subscriber Management):**
- ACS REST API for magic link emails (hand-rolled HMAC signing)
- `chi` middleware for session/token validation
- `templ` + htmx for preferences dashboard

**Phase 4 (AI Classification):**
- `anthropic-sdk-go` for Claude API
- `azqueue/v2` for `raw-updates` queue consumer
- Classification results written back to Cosmos DB as `Update` documents

**Phase 5 (Email Digests):**
- ACS REST API for digest delivery
- `azqueue/v2` for `digest-queue`

**Phases 9–10 (AWS / Kubernetes providers):**
- `gofeed` for multi-format feed parsing (AWS)
- Custom markdown parser for Kubernetes CHANGELOG

---

## Version Compatibility

| Package | Requires | Notes |
|---------|----------|-------|
| `anthropic-sdk-go v1.26.0` | Go 1.22+ | Upgrade from Go 1.21 required if still on older minor |
| `azcosmos v1.4.2` | Go 1.18+ | No constraint issues with Go 1.26 |
| `azqueue/v2 v2.0.1` | Go 1.18+ | v2 module path: `sdk/storage/azqueue/v2` |
| `azidentity v1.13.1` | Go 1.18+ | No constraint issues |
| `chi/v5 v5.2.5` | Go 1.14+ | No constraint issues |
| `templ v0.3.1001` | Go 1.21+ (estimated) | Uses generics; tool directive requires Go 1.24+ |
| `gofeed v1.3.0` | Go 1.16+ | No constraint issues |

Minimum Go version for the full stack: **Go 1.22** (driven by `anthropic-sdk-go`). Current stable is 1.26.1 — use that.

---

## Sources

- `pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos` — version v1.4.2 verified (Dec 2025)
- `pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity` — version v1.13.1 verified (Nov 2025)
- `pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue/v2` — version v2.0.1 verified (Jan 2026)
- `github.com/anthropics/anthropic-sdk-go` — version v1.26.0, Go 1.22+ requirement verified
- `pkg.go.dev/github.com/a-h/templ` — version v0.3.1001 verified (Feb 2026), pre-v1 status confirmed
- `pkg.go.dev/github.com/go-chi/chi/v5` — version v5.2.5 verified (Feb 2026)
- `pkg.go.dev/github.com/mmcdole/gofeed` — version v1.3.0 verified (Mar 2024)
- `go.dev/doc/devel/release` — Go 1.26.1 confirmed as latest stable (Mar 2026)
- Microsoft Learn Q&A — No official ACS Email Go SDK confirmed; REST API with HMAC-SHA256 is the recommended approach
- `learn.microsoft.com/en-us/azure/communication-services/tutorials/hmac-header-tutorial` — HMAC signing tutorial for ACS REST API
- `learn.microsoft.com/en-us/azure/container-apps/jobs` — Container Apps Jobs for scheduled and event-driven workloads
- [Go chi router analysis](https://www.calhoun.io/go-servemux-vs-chi/) — stdlib vs chi tradeoff analysis (MEDIUM confidence, community source)

---
*Stack research for: Upstream — Go infrastructure change awareness newsletter platform*
*Researched: 2026-03-09*
