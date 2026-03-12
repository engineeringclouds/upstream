# Requirements

## Overview

Full product requirements for the Upstream infrastructure change awareness platform. Phase mapping indicates when each requirement is targeted.

**Phase 1** is the current build scope. All other phases are future scope.

---

## Epic 1 — Data Ingestion

### Providers

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-001 | Azure RSS provider fetches `https://www.microsoft.com/releasecommunications/api/v2/azure/rss` and parses XML into typed `RawUpdate` structs | 1 | Must |
| REQ-002 | Azure provider extracts: title, description, source URL, published date, raw category tags from RSS XML | 1 | Must |
| REQ-003 | Azure provider accepts a `since time.Time` parameter and filters results to updates published after that time | 1 | Must |
| REQ-004 | `Provider` interface defined in `internal/provider/provider.go` with `Name() string` and `Fetch(ctx, since) ([]RawUpdate, error)` | 1 | Must |
| REQ-005 | `RawUpdate` struct includes: ProviderID, Provider, Title, RawContent, ContentFormat, SourceURL, Published, RawCategories, RawMeta | 1 | Must |
| REQ-006 | Provider implementations have zero dependencies on internal packages | 1 | Must |
| REQ-007 | AWS What's New RSS provider fetches and parses into `RawUpdate` using the same `Provider` interface | 9 | Must |
| REQ-008 | Kubernetes CHANGELOG markdown parser fetches per-release CHANGELOG files from GitHub and parses section headings into `RawUpdate` structs | 10 | Must |
| REQ-009 | Kubernetes blog RSS feed ingested as supplementary source | 10 | Should |

### Taxonomy

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-010 | `config/taxonomy.yaml` defines canonical Platform → Category → Service hierarchy | 3 | Must |
| REQ-011 | Taxonomy maps source-specific terms (Azure category tags, AWS service names, K8s SIG tags) to normalized names | 3 | Must |
| REQ-012 | Taxonomy changes are versioned; version field included in config | 9 | Should |
| REQ-013 | Unmatched source tags are logged as warnings (taxonomy drift detection) | 9 | Should |

---

## Epic 2 — CLI Interface

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-020 | `upstream fetch` command fetches and displays Azure updates to stdout | 1 | Must |
| REQ-021 | `--since` flag accepts a date string (e.g. `2024-01-01`) and filters updates | 1 | Must |
| REQ-022 | `--format` flag accepts `text` (default) and `json` output formats | 1 | Must |
| REQ-023 | Text format shows: title, published date, URL, categories per update | 1 | Must |
| REQ-024 | JSON format emits valid JSON array of update objects | 1 | Should |
| REQ-025 | Exit code 1 on errors; errors printed to stderr | 1 | Must |
| REQ-026 | `go build ./cmd/upstream` produces a working binary | 1 | Must |

---

## Epic 3 — Storage

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-030 | `Store` interface defined with methods for writing and reading `Update` and `RawUpdate` documents | 2 | Must |
| REQ-031 | Cosmos DB implementation of `Store` using `azcosmos` SDK | 2 | Must |
| REQ-032 | `updates` container partitioned by `/platform` | 2 | Must |
| REQ-033 | `subscribers` container partitioned by `/id` | 2 | Must |
| REQ-034 | `digests` container partitioned by `/subscriberId` | 5 | Must |
| REQ-035 | All Cosmos DB containers use serverless throughput mode | 2 | Must |
| REQ-036 | Store operations wrap errors with context: `fmt.Errorf("store: operation: %w", err)` | 2 | Must |

---

## Epic 4 — AI Classification

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-040 | `Classifier` classifies each `RawUpdate` into `UpdateType`: feature, deprecation, security, preview, breaking, retirement | 4 | Must |
| REQ-041 | `Classifier` assigns `Severity`: informational, action-recommended, action-required, critical | 4 | Must |
| REQ-042 | `Classifier` tags `Services` using normalized taxonomy names | 4 | Must |
| REQ-043 | `Classifier` generates a plain-language `Summary` for each update | 4 | Must |
| REQ-044 | Summary cached per update — not regenerated per subscriber or per digest | 4 | Must |
| REQ-045 | Classifier checks `classifiedAt` field before processing — already-classified updates are skipped | 4 | Must |
| REQ-046 | `DeprecationInfo` populated for updates classified as deprecation or retirement | 4 | Must |
| REQ-047 | Claude API used for classification via `anthropic-sdk-go` | 4 | Must |
| REQ-048 | Classifier system prompt eligible for Anthropic prompt caching (consistent prompt hash) | 4 | Should |
| REQ-049 | Classification errors are logged and the update is left unclassified (not discarded) | 4 | Must |

---

## Epic 5 — Subscriber Management

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-050 | User signs up with email address only — no password | 3 | Must |
| REQ-051 | Magic link sent via ACS; link is single-use and expires after 15 minutes | 3 | Must |
| REQ-052 | Magic link tokens stored as HMAC-SHA256 hash (not plaintext) at rest | 3 | Must |
| REQ-053 | Rate limiting applied to magic link requests per email address | 3 | Must |
| REQ-054 | Subscriber selects topics from hierarchical taxonomy (Platform → Category → Service) | 3 | Must |
| REQ-055 | Subscriber selects delivery channels: weekly digest, urgent alerts, deprecation watch | 3 | Must |
| REQ-056 | Preferences dashboard (htmx-based) allows post-signup updates | 3 | Must |
| REQ-057 | Unsubscribe link included in all email communications | 3 | Must |

---

## Epic 6 — Digest Assembly + Delivery

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-060 | Weekly digest scheduled via Container Apps Job (cron) | 5 | Must |
| REQ-061 | Digest assembler fetches classified updates from the past 7 days | 5 | Must |
| REQ-062 | Updates filtered to subscriber's selected topics before inclusion | 5 | Must |
| REQ-063 | Per-subscriber digest delivered via ACS email | 5 | Must |
| REQ-064 | Assembled digest cached in `digests` container before sending | 5 | Should |
| REQ-065 | ACS SPF configured as exact-match (not multi-include) on sending subdomain | 5 | Must |
| REQ-066 | Bounce and complaint handling configured before sending to lists > 10 addresses | 5 | Must |
| REQ-067 | Bulk send uses batched API calls (not one ACS call per subscriber) | 5 | Must |

---

## Epic 7 — Editorial Workflow

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-070 | Claude API drafts the "this week in infrastructure" editorial section | 6 | Must |
| REQ-071 | Editorial draft presented in a human review UI before publish | 6 | Must |
| REQ-072 | Human editor can revise the draft before approving | 6 | Must |
| REQ-073 | Approved editorial section prepended to every subscriber's weekly digest | 6 | Must |
| REQ-074 | Editorial stored in Cosmos DB; publish state tracked (draft, approved, sent) | 6 | Must |

---

## Epic 8 — Urgent Alerts

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-080 | Breaking change and critical CVE updates trigger the urgent alert pipeline | 7 | Must |
| REQ-081 | Human review gate required before alert dispatch (no auto-send) | 7 | Must |
| REQ-082 | Alerts scoped to subscriber's selected platforms | 7 | Must |
| REQ-083 | Alert review gate may be relaxed only after classification precision is measured and documented | 7 | Should |
| REQ-084 | Alert cadence is real-time(ish) — triggered by classification job, not cron | 7 | Must |

---

## Epic 9 — Deprecation Watch

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-090 | Deprecation lifecycle tracked through states: announced, active, imminent, retired | 8 | Must |
| REQ-091 | Monthly deprecation rollup delivered per platform | 8 | Must |
| REQ-092 | Items approaching retirement date resurfaced in rollup regardless of publication date | 8 | Must |
| REQ-093 | `DeprecationInfo.RetirementDate` populated when available; migration path noted | 4 | Should |

---

## Epic 10 — Infrastructure + Deployment

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-100 | Application deployed to Azure Container Apps via Bicep IaC | 2 | Must |
| REQ-101 | Scheduled jobs (polling, digest, deprecation) run as Container Apps Jobs | 2 | Must |
| REQ-102 | Queue-driven classifier job uses Azure Queue Storage as trigger (KEDA scaler) | 4 | Must |
| REQ-103 | Bicep modules organized under `deploy/azure/` | 2 | Must |
| REQ-104 | All infrastructure is scale-to-zero; no standing compute cost when idle | 2 | Must |
| REQ-105 | ACS sending domain configured with SPF, DKIM, DMARC before any production send | 3 | Must |
| REQ-106 | Queue access in Container Apps Jobs goes direct via SDK (Dapr not supported on Jobs) | 4 | Must |

---

## Epic 11 — Code Quality

| ID | Requirement | Phase | Priority |
|----|-------------|-------|----------|
| REQ-110 | `go test ./...` passes with meaningful coverage of parsing and classification logic | 1 | Must |
| REQ-111 | `go vet ./...` produces no output | 1 | Must |
| REQ-112 | All committed code passes `gofmt` (no diff) | 1 | Must |
| REQ-113 | All errors wrapped with context: `fmt.Errorf("package: operation: %w", err)` | 1 | Must |
| REQ-114 | No `os.Getenv()` calls inside packages — config injected via constructors from `main.go` | 1 | Must |
| REQ-115 | No sensitive values logged | 1 | Must |
| REQ-116 | Table-driven tests used for multiple input cases | 1 | Must |
| REQ-117 | `testdata/` fixtures sourced from real feed responses (not synthetic) | 1 | Should |

---

## Constraints (Cross-Cutting)

| ID | Constraint | Notes |
|----|------------|-------|
| CON-001 | Standard library only in Phase 1 | No external dependencies until Phase 2 |
| CON-002 | External dependencies require explicit justification | Minimize third-party surface area |
| CON-003 | `context.Context` always first parameter | Never stored in structs |
| CON-004 | Packages are short, lowercase, singular | `provider`, `store`, `mail` — not `providers` |
| CON-005 | Go 1.26.1+ | Minimum floor from `anthropic-sdk-go` requirement (1.22), but use current |
| CON-006 | Azure-native deployment | No portability abstractions; build directly against Azure SDKs |
| CON-007 | Bicep only for IaC | Not Terraform |
| CON-008 | Cost target ≤ $40/month at moderate scale | Drives serverless choices throughout |
| CON-009 | No auto-send for urgent alerts until classification precision documented | Human review gate until then |
| CON-010 | ACS Email: no Go SDK — must hand-roll HMAC-SHA256 request signing | ~1-2 days implementation in Phase 3 |

---

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| REQ-001 | Phase 1 | Complete |
| REQ-002 | Phase 1 | Complete |
| REQ-003 | Phase 1 | Complete |
| REQ-004 | Phase 1 | Pending |
| REQ-005 | Phase 1 | Pending |
| REQ-006 | Phase 11 | Complete |
| REQ-007 | Phase 9 | Pending |
| REQ-008 | Phase 10 | Pending |
| REQ-009 | Phase 10 | Pending |
| REQ-010 | Phase 3 | Pending |
| REQ-011 | Phase 3 | Pending |
| REQ-012 | Phase 9 | Pending |
| REQ-013 | Phase 9 | Pending |
| REQ-020 | Phase 1 | Complete |
| REQ-021 | Phase 1 | Complete |
| REQ-022 | Phase 1 | Complete |
| REQ-023 | Phase 1 | Complete |
| REQ-024 | Phase 1 | Complete |
| REQ-025 | Phase 1 | Complete |
| REQ-026 | Phase 1 | Complete |
| REQ-030 | Phase 2 | Pending |
| REQ-031 | Phase 2 | Pending |
| REQ-032 | Phase 2 | Pending |
| REQ-033 | Phase 2 | Pending |
| REQ-034 | Phase 5 | Pending |
| REQ-035 | Phase 2 | Pending |
| REQ-036 | Phase 2 | Pending |
| REQ-040 | Phase 4 | Pending |
| REQ-041 | Phase 4 | Pending |
| REQ-042 | Phase 4 | Pending |
| REQ-043 | Phase 4 | Pending |
| REQ-044 | Phase 4 | Pending |
| REQ-045 | Phase 4 | Pending |
| REQ-046 | Phase 4 | Pending |
| REQ-047 | Phase 4 | Pending |
| REQ-048 | Phase 4 | Pending |
| REQ-049 | Phase 4 | Pending |
| REQ-050 | Phase 3 | Pending |
| REQ-051 | Phase 3 | Pending |
| REQ-052 | Phase 3 | Pending |
| REQ-053 | Phase 3 | Pending |
| REQ-054 | Phase 3 | Pending |
| REQ-055 | Phase 3 | Pending |
| REQ-056 | Phase 3 | Pending |
| REQ-057 | Phase 3 | Pending |
| REQ-060 | Phase 5 | Pending |
| REQ-061 | Phase 5 | Pending |
| REQ-062 | Phase 5 | Pending |
| REQ-063 | Phase 5 | Pending |
| REQ-064 | Phase 5 | Pending |
| REQ-065 | Phase 5 | Pending |
| REQ-066 | Phase 5 | Pending |
| REQ-067 | Phase 5 | Pending |
| REQ-070 | Phase 6 | Pending |
| REQ-071 | Phase 6 | Pending |
| REQ-072 | Phase 6 | Pending |
| REQ-073 | Phase 6 | Pending |
| REQ-074 | Phase 6 | Pending |
| REQ-080 | Phase 7 | Pending |
| REQ-081 | Phase 7 | Pending |
| REQ-082 | Phase 7 | Pending |
| REQ-083 | Phase 7 | Pending |
| REQ-084 | Phase 7 | Pending |
| REQ-090 | Phase 8 | Pending |
| REQ-091 | Phase 8 | Pending |
| REQ-092 | Phase 8 | Pending |
| REQ-093 | Phase 4 | Pending |
| REQ-100 | Phase 2 | Pending |
| REQ-101 | Phase 2 | Pending |
| REQ-102 | Phase 4 | Pending |
| REQ-103 | Phase 2 | Pending |
| REQ-104 | Phase 2 | Pending |
| REQ-105 | Phase 3 | Pending |
| REQ-106 | Phase 4 | Pending |
| REQ-110 | Phase 1 | Pending |
| REQ-111 | Phase 1 | Pending |
| REQ-112 | Phase 1 | Pending |
| REQ-113 | Phase 1 | Complete |
| REQ-114 | Phase 1 | Complete |
| REQ-115 | Phase 1 | Complete |
| REQ-116 | Phase 1 | Pending |
| REQ-117 | Phase 1 | Pending |

**Total: 67/67 requirements mapped. No orphans.**

---

*Last updated: 2026-03-10 after roadmap creation*
