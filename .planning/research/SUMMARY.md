# Project Research Summary

**Project:** Upstream — AI-enhanced infrastructure change awareness platform
**Domain:** Go newsletter platform (RSS aggregation, AI classification, personalized email digest, Azure-native)
**Researched:** 2026-03-09
**Confidence:** HIGH (stack and architecture from official docs); MEDIUM (features from competitive landscape survey)

---

## Executive Summary

Upstream is a multi-phase newsletter platform that turns high-volume cloud update streams into a personalized, editorially-curated digest. The architecture follows a two-stage ingestion pipeline: scheduled fetcher jobs pull raw RSS (encoding/xml for Azure in Phase 1; gofeed for AWS in Phase 9), write unclassified documents to Cosmos DB, and enqueue IDs to Azure Queue Storage; event-driven classifier jobs consume that queue and call the Claude API once per update, storing the enriched result back to Cosmos DB with severity, type, and service tags. Digest assembly reads classified updates at query time, filters by subscriber topic preferences, and renders email via ACS. Classification runs once per update and is never re-run per subscriber — this single decision controls AI spend across the entire lifecycle.

The recommended stack is standard library-first, introducing dependencies phase-by-phase: encoding/xml and net/http suffice for Phase 1; chi, templ, and htmx enter at Phase 2; the Azure SDK modules (azidentity, azcosmos, azqueue/v2) and the Anthropic SDK (anthropic-sdk-go v1.26.0, requires Go 1.22+) follow in Phases 2–4. There is no official Go SDK for Azure Communication Services — email delivery requires hand-rolled HMAC-SHA256 request signing, which is a concrete ~1–2 day implementation task in Phase 3 rather than a gap in the plan.

The three highest-consequence risks are architectural decisions that cannot be cheaply reversed: (1) Cosmos DB partition key strategy must be correct before the first document is written in Phase 2 — post-production migration is a full data migration; (2) AI classification idempotency (the classifiedAt guard) must be in place before Phase 4 goes live — without it, every feed poll re-classifies all known updates and spend scales with total update count rather than net-new updates; (3) taxonomy config must be an explicit artifact produced in Phase 3 (canonical before Phase 4 begins) — every downstream feature (filtering, classification, subscriber preferences, deprecation watch) depends on it being stable.

---

## Key Findings

### Recommended Stack

The full stack is Go 1.26.1 (current stable, required by anthropic-sdk-go's Go 1.22+ floor), stdlib for Phase 1, then chi v5.2.5 + templ v0.3.1001 for Phase 2 web UI. templ is pre-v1 but production-stable — pin the exact version. All Azure services use the official azure-sdk-for-go modules with DefaultAzureCredential (managed identity in Container Apps, Azure CLI locally, zero config changes between environments). The deprecated azure-storage-queue-go must not be used; the correct module path is `sdk/storage/azqueue/v2`.

**Core technologies:**
- Go 1.26.1: primary language; minimum 1.22 for anthropic-sdk-go
- encoding/xml (stdlib): Azure RSS parsing in Phase 1; exact field control, teaches the pattern
- chi v5.2.5: HTTP routing; defer until Phase 2 when middleware grouping justifies the dependency
- templ v0.3.1001: type-safe HTML templates; compile-time errors; pre-v1, pin version
- azidentity v1.13.1: DefaultAzureCredential; zero-secret local and deployed auth
- azcosmos v1.4.2: Cosmos DB document store; serverless billing
- azqueue/v2 v2.0.1: Queue Storage; note v2 module path
- anthropic-sdk-go v1.26.0: Claude API; official SDK; requires Go 1.22+
- ACS via net/http: no official Go SDK exists; HMAC-SHA256 signing required; Microsoft has a Go tutorial
- gofeed v1.3.0: multi-format feed parser; defer until Phase 9 (AWS)
- Bicep: IaC only; Terraform explicitly out of scope

**What NOT to use:** azure-storage-queue-go (deprecated), vippsas/go-cosmosdb (community, no managed identity), unfunco/anthropic-sdk-go (community, predates official), gin-gonic/gin (incompatible context type), os.Getenv inside packages (violates architecture rules), Azure AD B2C (over-engineered for v1).

### Expected Features

Research confirms the feature sequencing in PROJECT.md is correct. The critical dependency chain is: taxonomy config → AI classification → topic filtering → digest assembly → all personalization. Everything downstream of Phase 4 degrades if classification quality is poor.

**Must have (table stakes for v1):**
- Azure RSS ingestion with XML parsing
- Cosmos DB persistence (updates survive a single fetch run)
- Magic link auth via ACS (gate to all personalization)
- Hierarchical topic selection (Platform → Category → Service)
- Weekly digest assembly and email delivery via ACS
- AI classification (type + severity) and per-update cached summaries
- Unsubscribe in one click (CAN-SPAM/GDPR)
- Mobile-responsive web UI and email templates

**Differentiators worth building:**
- Deprecation lifecycle tracking (announced → active → imminent → retired) — genuine competitive white space; no existing product tracks deprecations with a notification layer across their full lifecycle
- Urgent/breaking change alerts with mandatory human review gate
- Severity-tiered delivery channels (weekly, urgent, deprecation-monthly)
- Human editorial "this week in infrastructure" section — the primary retention driver once AI summaries are commoditized
- Open source codebase as a trust and credibility signal with the infrastructure practitioner audience

**Defer to v2+:**
- AWS provider (Phase 9) — no structured tags; requires full AI classification; defer until Azure loop is proven
- Kubernetes provider (Phase 10) — markdown parsing, not feed parsing; separate problem
- Digest preview before subscribing (Phase 11) — low friction reduction; a static example in marketing copy substitutes cheaply
- Per-subscriber AI summarization — never; cost is O(subscribers × updates)
- Mobile app, real-time chat, OAuth/SSO — all explicitly out of scope

**Anti-features (explicitly do not build):**
- Per-subscriber AI summarization: multiplies cost by subscriber count
- Auto-dispatch of urgent alerts without human review: one false positive destroys alert channel trust permanently
- Passwords or OAuth: magic link is less friction for the target audience

### Architecture Approach

The system runs three Container Apps Jobs (Fetcher — scheduled every 6h; Classifier — event-driven on queue depth; Mailer — scheduled Monday 08:00 UTC) and one Container Apps App (web server). Jobs do NOT support Dapr or Ingress — queue access is direct via azqueue SDK only. The Fetcher writes unclassified updates to Cosmos DB and enqueues IDs; the Classifier dequeues, calls Claude once per update, and stores the enriched result; the Mailer reads classified updates per platform, filters in-memory per subscriber preferences, assembles and sends digests. The web app handles subscriber management, preferences, and the editorial review interface.

**Major components:**
1. `internal/provider/azure` — fetch RSS, return []RawUpdate; zero internal dependencies
2. `internal/store` — all Cosmos DB I/O; single point for persistence; the only package that imports azcosmos
3. `internal/queue` — wraps azqueue SDK; single abstraction for both Fetcher and Classifier
4. `internal/classifier` — dequeue → get raw update → AI → store enriched update; owns idempotency guard
5. `internal/ai` — pure HTTP client to Claude API; no business logic; classifier owns what to do with responses
6. `internal/digest` — assemble per-subscriber digest from classified updates; never calls AI
7. `internal/mail` — ACS REST API client; HMAC-SHA256 signing; shared by web (magic links) and Mailer Job
8. `internal/auth` — magic link token generation and verification; session management
9. `internal/web` — HTTP handlers, templ templates, htmx partials; depends on store, auth, mail, digest
10. `config/taxonomy.yaml` — canonical service taxonomy; versioned config file, not Go constants

**Partition key decisions (irreversible without migration):**
- `updates` container: `/provider` (azure, aws, kubernetes) — fetcher and classifier always scope to one provider; digest queries within-partition
- `subscribers` container: `/subscriberID` — all auth and preference ops are single-subscriber point reads
- `digests` container: `/subscriberID` — digest assembly reads and updates one subscriber at a time

### Critical Pitfalls

1. **Cosmos DB partition key is irreversible** — define partition keys in Bicep before Phase 2 deploys the first document. Wrong choice = full data migration to recover. Use `/provider` for updates, `/subscriberID` for subscribers and digests.

2. **AI classification cost trap** — without a `classifiedAt` guard, every feed poll re-classifies all stored updates. The classifier job must query only `WHERE classifiedAt = null`. Verify by running the job twice and asserting zero AI API calls on the second run. Use Anthropic Batch API (50% cost reduction) for non-urgent bulk classification. Use prompt caching (90% cost reduction on repeated system prompts).

3. **No official Go SDK for ACS Email** — hand-roll HMAC-SHA256 signing using net/http. Microsoft publishes a Go HMAC tutorial for ACS REST API version 2025-09-01. Budget 1–2 days in Phase 3. This also means ACS SPF must be exact-match (`v=spf1 include:spf.protection.outlook.com -all`) on a dedicated subdomain — do not use the root domain or multi-include records. Portal status indicator is unreliable; verify via email header inspection.

4. **Magic link tokens must be hashed at rest + single-use** — store SHA-256 hash only; mark `usedAt` atomically on first redemption; enforce 15-minute TTL; use crypto/rand not math/rand. Rate limit the send endpoint (5 requests per email per hour). A second redemption of the same token must return 401 — verify with an integration test.

5. **Taxonomy config must be a versioned file, not Go constants** — every downstream feature depends on the taxonomy being stable and evolvable without a code deploy. Store in `config/taxonomy.yaml`. Log a warning metric when any feed tag fails normalization. Include `taxonomyVersion` on classified updates. Produce an unmatched-tag report as a Phase 4 deliverable.

6. **RSS date parsing fails silently** — Azure Updates RSS mixes date formats and occasionally emits malformed pubDate fields. Write a `parseDate` helper that tries multiple formats in order and logs a warning (never silently zero-dates an entry). Test with a testdata/ fixture that includes malformed dates.

7. **Go error handling patterns set in Phase 1 propagate everywhere** — establish the pattern before any other code exists: all `internal/` functions return wrapped errors (`fmt.Errorf("context: %w", err)`); only cmd/ and main.go call log.Fatal; no errors discarded with `_ =`. Add errcheck to CI in Phase 1.

---

## Implications for Roadmap

The 11-phase structure in PROJECT.md is correct and the dependency ordering is sound. The synthesis adds watch items and sequencing constraints within phases.

### Phase 1: Azure CLI Foundation
**Rationale:** Zero dependencies, teaches Go patterns, validates the RSS parsing problem before committing to the full stack.
**Delivers:** Working CLI, tested XML parser, correct error handling patterns established.
**Must avoid:** Silently zero-dating entries (multi-format parseDate helper required); log.Fatal in internal packages; classifying inside the provider.
**Research flag:** Standard patterns — no additional research needed. encoding/xml + net/http is well-documented.

### Phase 2: Azure Deployment + Data Model
**Rationale:** Establishes the infrastructure and data model before any code depends on it. Partition key decisions made here are irreversible.
**Delivers:** Cosmos DB (three containers with correct partition keys), Container Apps environment, basic public web UI, email deliverability DNS configured on sending subdomain.
**Critical sequence:** Bicep must define partition keys on all containers BEFORE first deploy. DNS/DKIM/DMARC for ACS sending subdomain must be configured BEFORE any production email is sent.
**Must avoid:** Single container for all document types; wrong partition key; root domain for ACS sending.
**Research flag:** ACS domain configuration deserves a focused spike — portal status indicator is unreliable, verify via actual email headers.

### Phase 3: Auth + Subscriber Management
**Rationale:** Auth gates all personalization. Must be correct before subscriber data accumulates.
**Delivers:** Magic link auth, subscriber signup, hierarchical topic selection, preferences dashboard, channel selection.
**Critical implementation items:**
  - ACS HMAC-SHA256 signing (~1–2 days; no SDK)
  - Token hashing at rest (SHA-256 hash only, never raw value)
  - Single-use enforcement with atomic usedAt marking
  - Rate limiting on magic link send endpoint
  - Taxonomy config artifact (`config/taxonomy.yaml`) produced and reviewed — this is a Phase 3 deliverable, not Phase 4, because the subscriber preferences UI depends on the taxonomy hierarchy
**Must avoid:** math/rand for tokens; plaintext token storage; missing rate limiting; taxonomy improvised at classification time.
**Research flag:** Token storage and session management patterns in Go — worth a focused research pass before implementation.

### Phase 4: AI Classification Pipeline
**Rationale:** Linchpin phase — urgent alerts, deprecation watch, and personalized filtering all depend on reliable classification output. Highest-risk phase.
**Delivers:** Classifier job, Claude API integration, type + severity + service tag classification, classification cached per update.
**Critical implementation items:**
  - classifiedAt guard is a correctness requirement, not optimization
  - Taxonomy must be complete before classification starts (taxonomy.yaml from Phase 3)
  - Unmatched-tag logging and weekly report as explicit deliverables
  - Dead-letter container for messages that fail after N retries
  - Queue visibility timeout ≥ 300s (exceeds worst-case AI latency)
  - Validate AI output against allowed enum values before persisting
  - Batch API for non-urgent classification (50% cost reduction)
  - Prompt caching for repeated system prompts (90% cache token reduction)
**Must avoid:** Re-classifying already-classified updates; classifying inside providers; storing raw AI response as freeform JSON; auto-dispatch of anything from this pipeline.
**Research flag:** NEEDS research-phase before planning. Prompt engineering for structured output, batch API mechanics, and idempotency patterns need a focused spike.

### Phase 5: Weekly Digest + Email Delivery
**Rationale:** First value delivery to subscribers. Depends on all prior phases.
**Delivers:** Weekly digest assembled from classified updates, personalized per subscriber topics, delivered via ACS.
**Critical implementation items:**
  - Digest assembly must NOT query the database inside a per-subscriber loop — fetch all updates per platform in bulk, filter in-memory
  - Bounce and complaint webhooks from ACS must be implemented before any bulk send
  - Hard bounce suppression list
  - List-unsubscribe header + one-click unsubscribe landing page (CAN-SPAM/GDPR requirement)
**Must avoid:** Per-subscriber DB queries in digest loop; skipping bounce handling until later; sending to large lists before deliverability validation on a small test set.
**Research flag:** Digest assembly query pattern and RU cost estimation — validate the bulk-fetch-then-filter approach against the actual Cosmos DB query API before coding.

### Phase 6: Editorial Workflow
**Rationale:** Enhances but does not block digest delivery. Add once the delivery loop is running and there is an editor to operate it.
**Delivers:** AI-drafted editorial section, human review interface, editorial prepended to every digest.
**Research flag:** Standard patterns for the web UI layer. AI drafting uses the same anthropic-sdk-go already in use.

### Phase 7: Urgent Alerts
**Rationale:** Requires measurable classification precision before enabling. Do not enable until Phase 4 quality is validated.
**Delivers:** Out-of-band breaking change alerts, human review gate, subscriber opt-in scoping.
**Critical gate:** Human review is mandatory until false positive rate is measured and acceptable on a retrospective dataset. Auto-dispatch is an anti-feature.
**Research flag:** Alert delivery mechanics (ACS transactional send vs. digest send path) need review.

### Phase 8: Deprecation Watch
**Rationale:** State machine for deprecation lifecycle. Depends on classification (type=deprecation|retirement) and Cosmos DB persistence.
**Delivers:** Lifecycle tracking, monthly rollup digest, timeline resurface logic.
**Research flag:** State machine design and Cosmos DB TTL policy for lifecycle documents — worth a focused research pass.

### Phase 9: AWS Provider
**Rationale:** Proves the Provider interface design. AWS has no structured tags — full AI classification required.
**Delivers:** AWS What's New ingestion, normalized to RawUpdate, classified via existing pipeline.
**Critical item:** Taxonomy must be explicitly extended for AWS services before classification runs.
**Research flag:** NEEDS research-phase. AWS feed format, service normalization strategy, and AI classification prompt adjustments for unstructured input need research before planning.

### Phase 10: Kubernetes Provider
**Rationale:** Different ingest problem (markdown parsing, not feed parsing). Defer until multi-cloud demand is confirmed.
**Delivers:** CHANGELOG markdown parsing, blog RSS ingestion, SIG tag normalization.
**Research flag:** NEEDS research-phase. Markdown parsing approach and SIG tag taxonomy are not well-explored.

### Phase 11: Polish
**Rationale:** Onboarding and discovery improvements. No blocking dependencies.
**Delivers:** Digest preview, contributor guide, API docs.
**Research flag:** Standard patterns — no additional research needed.

### Phase Ordering Rationale

- Phases 1–2 establish the substrate (parsing, persistence, infrastructure) before any feature logic is layered on
- Phase 3 must precede Phase 4 because the taxonomy config — a Phase 3 artifact — is a hard prerequisite for classification
- Phase 5 (digest delivery) cannot be built before Phase 4 (classification) because an unclassified digest is undifferentiated noise
- Phases 6–8 enhance a working delivery loop and can be sequenced by demand
- Phases 9–10 are pure provider additions — the pipeline accepts them without modification if the Provider interface is correctly designed in Phase 1
- The two irreversible decisions (Cosmos DB partition keys, ACS sending domain) must be made in Phase 2, before any data accumulates

---

## Risks Ranked by Consequence

| Rank | Risk | Consequence | Mitigation | Phase |
|------|------|-------------|------------|-------|
| 1 | Wrong Cosmos DB partition key deployed | Full data migration to recover; no partial fix | Define in Bicep before Phase 2 deploys first document | Phase 2 |
| 2 | AI classification without idempotency guard | Spend scales with total update count, not new updates; undetected until billing surprises | classifiedAt guard before any Phase 4 goes live; verify by running job twice | Phase 4 |
| 3 | ACS account suspended for bounces | Cannot send to subscribers; 3–5 day review to restore; reputation damage | Bounce webhook before first bulk send; test deliverability on small list first | Phase 5 |
| 4 | Magic link token stored plaintext or multi-use | Account takeover on Cosmos DB breach or email interception | Hash at rest; single-use with atomic marking; crypto/rand; 15-min TTL | Phase 3 |
| 5 | Taxonomy improvised in Phase 4 (not a designed artifact) | All downstream features (filtering, subscriber preferences, urgent alerts, deprecation watch) built on unstable foundation; expensive to migrate | Produce taxonomy.yaml as explicit Phase 3 deliverable; require sign-off before Phase 4 starts | Phase 3 |
| 6 | ACS SPF misconfiguration | Digest emails bulk-foldered at Gmail/Outlook; silent subscriber churn | Exact SPF on dedicated subdomain; verify via email headers not portal | Phase 2 |
| 7 | Go error handling anti-patterns set in Phase 1 | Systemic debt across all packages; requires a refactor sweep before adding features | Establish pattern in Phase 1; errcheck in CI from day one | Phase 1 |
| 8 | Queue visibility timeout too short | Classifier processes same update twice during AI call; duplicate classifications | Visibility timeout ≥ 300s; DequeueCount check for dead-lettering | Phase 4 |
| 9 | Taxonomy drift after Phase 4 goes live | Subscriber topic selections silently stop matching; empty digest sections | Unmatched-tag logging; taxonomyVersion on documents; weekly unmatched report | Phase 4 |
| 10 | RSS date format parsing fails silently | Updates dropped or always included regardless of --since filter | Multi-format parseDate helper; testdata/ fixture with malformed dates | Phase 1 |

---

## Open Questions

1. **Sending subdomain:** What is the confirmed sending subdomain for ACS? (`mail.upstream.engineeringclouds.io`?) DNS records must be configured before Phase 2 completes. This is a prerequisite for Phase 3 magic link testing.

2. **Cosmos DB account naming:** The Bicep template needs the Cosmos DB account name and resource group before Phase 2 infrastructure is deployed. Confirm naming convention with Engineering Clouds standards.

3. **Claude model selection:** Research recommends `claude-3-5-haiku` for classification (cost-optimized) and `claude-3-5-sonnet` for editorial drafting. Confirm model availability and pricing against the $40/month budget at expected volume before Phase 4.

4. **Batch API eligibility:** Anthropic Batch API provides 50% cost reduction for non-urgent workloads. Confirm latency tolerance for classification (hours acceptable vs. minutes required) to decide whether to use batch from Phase 4 launch.

5. **Taxonomy review process:** Who reviews and approves `config/taxonomy.yaml` before Phase 4 classification begins? This file needs a sign-off step before Phase 4 starts, not after.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All versions verified against pkg.go.dev and official Microsoft/Anthropic docs. Go 1.26.1 current. ACS no-SDK finding confirmed via Microsoft Learn Q&A. |
| Features | MEDIUM | Competitive landscape surveyed via direct site inspection and secondary sources. No primary user research. Deprecation white space finding is solid (AWS Product Lifecycle launched May 2025 with no notification layer; Azure Charts has no email). |
| Architecture | HIGH | Azure Container Apps Jobs, Cosmos DB partition key model, and azqueue v2 from official Microsoft docs. Go package layout from go.dev canonical guidance. Dapr/Ingress restriction on Jobs confirmed. |
| Pitfalls | HIGH | Azure-specific items from official docs. Go error patterns from 100go.co and Go wiki. AI cost math from Anthropic official pricing. ACS deliverability from RFC-grounded sources. |

**Overall confidence:** HIGH for the technical decisions; MEDIUM for the competitive positioning claims.

### Gaps to Address

- **ACS HMAC-SHA256 implementation:** The tutorial exists but this is a hand-roll — budget explicit time in Phase 3 and test against actual ACS endpoint before the magic link feature is considered complete.
- **Classification prompt quality:** Cannot know classification precision until Phase 4 runs against a live dataset. Build the unmatched-tag report and a retrospective evaluation step into Phase 4 before Phase 7 (urgent alerts) is planned.
- **Digest assembly RU cost at scale:** The bulk-fetch-then-filter approach is sound in theory. Validate against real Cosmos DB query costs before Phase 5 coding begins — the RU calculation in PITFALLS.md is directionally correct but not a verified cost model.

---

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev` — all Go library versions verified (azcosmos v1.4.2, azidentity v1.13.1, azqueue/v2 v2.0.1, chi v5.2.5, templ v0.3.1001, gofeed v1.3.0)
- `go.dev/doc/devel/release` — Go 1.26.1 confirmed current stable
- `github.com/anthropics/anthropic-sdk-go` — v1.26.0, Go 1.22+ requirement confirmed
- `learn.microsoft.com/en-us/azure/container-apps/jobs` — Jobs trigger types, Dapr/Ingress restriction, event-driven scaling
- `learn.microsoft.com/en-us/azure/cosmos-db/hierarchical-partition-keys` — partition key design, 20 GB logical limit
- `learn.microsoft.com/en-us/azure/communication-services/concepts/email/email-domain-and-sender-authentication` — SPF exact-match requirement, DKIM CNAMEs
- `learn.microsoft.com/en-us/azure/communication-services/tutorials/hmac-header-tutorial` — ACS HMAC signing for Go
- `platform.claude.com/docs/en/about-claude/pricing` — Batch API pricing, prompt caching rates
- Microsoft Learn Q&A — No official ACS Email Go SDK confirmed (March 2026)

### Secondary (MEDIUM confidence)
- `azurecharts.com/updates` — Azure Charts feature set, filtering capabilities, no email delivery
- `lastweekinaws.com` — Last Week in AWS editorial model, no topic filtering
- AWS Product Lifecycle page announcement (InfoQ, May 2025) — deprecation tracking gap
- `github.com/azure-deprecation/dashboard` — existing deprecation tracking, no notification layer
- `100go.co` and `go.dev/wiki/CommonMistakes` — Go error handling anti-patterns
- `mailgun.com/blog/deliverability/state-of-deliverability-takeaways/` — email deliverability standards 2025

### Tertiary (LOW confidence)
- Feedly Pro/Teams feature analysis (third-party review sites) — feature comparison only
- Community Go project structure sources (2025) — supplementary to official go.dev guidance

---

*Research completed: 2026-03-09*
*Ready for roadmap: yes*
