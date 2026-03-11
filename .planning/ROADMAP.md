# Roadmap: Upstream

## Overview

Upstream is built in eleven phases that follow a strict dependency chain: each phase delivers
a coherent, verifiable capability that the next phase depends on. Phases 1-5 form the v1 core
loop — fetch, store, auth, classify, deliver — and produce a working product. Phases 6-8 add
the editorial layer, urgent alerts, and deprecation tracking that differentiate Upstream from
raw RSS aggregators. Phases 9-10 extend the provider surface to AWS and Kubernetes. Phase 11
is polish and contributor readiness.

Two architectural decisions made in Phases 2 and 3 are irreversible without a full data
migration: Cosmos DB partition key design and the canonical taxonomy config. These must be
correct before any data accumulates against them.

---

## Phases

- [ ] **Phase 1: Foundation** - Go CLI that fetches Azure Updates RSS, parses XML, prints formatted output to stdout
- [ ] **Phase 2: Azure Deployment** - Cosmos DB storage, Container Apps infrastructure, Bicep IaC, basic public web UI
- [ ] **Phase 3: Auth + Subscribers** - Magic link auth, subscriber signup, topic selection, preferences dashboard, taxonomy deliverable
- [ ] **Phase 4: AI Classification** - Claude API integration, type/severity/service tagging per update, idempotency guard, cost controls
- [ ] **Phase 5: Weekly Digest** - Digest assembly and personalized email delivery via ACS
- [ ] **Phase 6: Editorial Workflow** - AI draft + human review interface, editorial section prepended to every digest
- [ ] **Phase 7: Urgent Alerts** - Breaking change detection, human review gate, alert delivery scoped to subscriber platforms
- [ ] **Phase 8: Deprecation Watch** - Lifecycle state machine, monthly rollup digest, retirement timeline resurfacing
- [ ] **Phase 9: AWS Provider** - AWS What's New RSS ingestion and classification via existing pipeline
- [ ] **Phase 10: Kubernetes Provider** - CHANGELOG markdown parsing, blog RSS, SIG tag normalization
- [ ] **Phase 11: Polish** - Onboarding flow, digest preview, API docs, production-quality README and contributing guide

---

## Phase Details

### Phase 1: Foundation

**Goal**: A working Go CLI that fetches Azure Updates RSS, parses XML with correct date handling, and prints formatted output to stdout — establishing the code quality patterns that all subsequent phases inherit.

**Scope**:
- In: `internal/provider/azure` RSS fetch and XML parse, `cmd/upstream` CLI with `--since` and `--format` flags, table-driven tests with real `testdata/` fixtures, `go vet` / `gofmt` clean
- Out: Web UI, persistence, auth, AI, email, Azure deployment, any external dependencies

**Depends on**: Nothing (first phase)

**Requirements**: REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-020, REQ-021, REQ-022, REQ-023, REQ-024, REQ-025, REQ-026, REQ-110, REQ-111, REQ-112, REQ-113, REQ-114, REQ-115, REQ-116, REQ-117

**Success Criteria** (what must be TRUE when this phase is complete):
1. `./upstream fetch` prints Azure updates to stdout with title, date, URL, and categories visible
2. `./upstream fetch --since 2024-01-01` returns only updates published after that date, including correct handling of malformed or mixed-format pubDate fields in the RSS feed
3. `./upstream fetch --format json` emits a valid JSON array; `./upstream fetch --format text` (default) emits readable text; exit code 1 on error with message to stderr
4. `go test ./...` passes with table-driven tests using real Azure RSS fixture data in `testdata/`, covering date parsing edge cases
5. `go vet ./...` produces no output; all committed files are `gofmt`-clean; no `log.Fatal` exists outside `cmd/`; all errors in `internal/` are wrapped with `fmt.Errorf("package: op: %w", err)`

**Risks**:
- Azure Updates RSS mixes date formats and emits malformed `pubDate` fields — a single `time.Parse` call silently zero-dates entries, breaking `--since` filtering. Mitigation: implement a `parseDate` helper that tries multiple formats in order (`time.RFC1123Z`, `time.RFC1123`, `time.RFC3339`, common RSS variants) and logs a warning on failure; include a malformed-date fixture in `testdata/`.
- Go error handling anti-patterns set here propagate to all 10 subsequent phases. Mitigation: establish and enforce `fmt.Errorf("context: %w", err)` wrapping, no `log.Fatal` in `internal/`, no `_ =` discards before writing any Phase 2 code; add `errcheck` to CI.
- Zero Go experience — non-idiomatic patterns will be harder to correct in later phases. Mitigation: prioritize correctness and idiomatic style over speed; treat Phase 1 as the pattern-establishing baseline.

**Research flag**: No additional research needed. `encoding/xml` + `net/http` stdlib patterns are well-documented.

**Plans**: TBD

---

### Phase 2: Azure Deployment

**Goal**: The infrastructure substrate is deployed — Cosmos DB with correct partition keys, Container Apps environment, Bicep IaC for all resources, and a basic public web UI — with ACS sending domain DNS configured before any email is sent.

**Scope**:
- In: Cosmos DB (three containers with correct partition keys defined in Bicep before first deploy), Container Apps environment, Container Registry, `internal/store` interface + Cosmos DB implementation, chi + templ web UI (no auth), Bicep modules under `deploy/azure/`, ACS sending domain configured (SPF/DKIM/DMARC on dedicated subdomain)
- Out: Auth, subscriber management, AI, email delivery, digest assembly

**Depends on**: Phase 1 (provider interface and `RawUpdate` struct must exist for store design)

**Requirements**: REQ-030, REQ-031, REQ-032, REQ-033, REQ-035, REQ-036, REQ-100, REQ-101, REQ-103, REQ-104

**Success Criteria** (what must be TRUE when this phase is complete):
1. `az deployment group create` runs against the Bicep templates and produces a working Container Apps environment with Cosmos DB, Queue Storage, and Container Registry — no manual portal steps required
2. The web app is publicly accessible via HTTPS at the Container Apps FQDN and displays Azure updates fetched from the store (or directly from the provider in an initial state)
3. All Cosmos DB containers (`updates`, `subscribers`) are created with their correct partition keys (`/provider` and `/subscriberID` respectively) as defined in Bicep — verified by inspecting the deployed container configuration before any documents are written
4. The ACS sending subdomain (e.g., `mail.upstream.engineeringclouds.io`) passes `spf=pass` and `dkim=pass` verification via actual email header inspection — not the portal status indicator
5. `internal/store` operations compile, all errors wrap with `fmt.Errorf("store: op: %w", err)`, and the `Store` interface is defined with `RawUpdate` and `Update` read/write methods

**Risks**:
- Cosmos DB partition key is irreversible — wrong choice requires a full data migration to recover. Mitigation: review partition key design (`/provider` for updates, `/subscriberID` for subscribers) in Bicep before the first `az deployment group create` that creates Cosmos DB resources; include partition key in a code review checklist item.
- ACS portal status indicator is unreliable — "Not configured" can display even when DNS is correctly propagated. Mitigation: verify deliverability configuration via actual email header inspection (`Authentication-Results` header) rather than trusting the portal badge before closing this phase.
- Single Cosmos DB container anti-pattern — tempting to start with one container and `docType` discriminator. Mitigation: Bicep defines separate containers per entity type from day one; adding containers in serverless mode has no standing cost.

**Research flag**: ACS domain configuration and DNS verification approach warrants a focused spike — the portal status unreliability is confirmed, but the exact sequence of records and verification method should be validated against the current ACS documentation before implementation.

**Plans**: TBD

---

### Phase 3: Auth + Subscribers

**Goal**: Subscribers can sign up with email only, receive a magic link, log in, select topics from a hierarchical taxonomy, choose delivery channels, and manage preferences — and `config/taxonomy.yaml` is produced as an explicit, reviewed deliverable before Phase 4 begins.

**Scope**:
- In: Magic link auth via ACS (hand-rolled HMAC-SHA256 signing — no Go SDK exists), `internal/auth` token generation/verification/session, `internal/mail` ACS client, subscriber signup flow, hierarchical topic selection UI (htmx + templ), channel selection (weekly digest / urgent alerts / deprecation watch), preferences dashboard, unsubscribe link in all emails, `config/taxonomy.yaml` canonical taxonomy produced and reviewed, ACS sending domain fully configured
- Out: AI classification, digest assembly, email delivery of digests (only magic link auth emails sent in this phase)

**Depends on**: Phase 2 (Cosmos DB, Container Apps, ACS sending domain DNS, `internal/store` interface)

**Requirements**: REQ-010, REQ-011, REQ-050, REQ-051, REQ-052, REQ-053, REQ-054, REQ-055, REQ-056, REQ-057, REQ-105

**Success Criteria** (what must be TRUE when this phase is complete):
1. A user can enter their email, receive a magic link within 30 seconds, click it, and reach their preferences dashboard — the link works exactly once (second click returns an error) and expires after 15 minutes
2. A subscriber can select topics at three levels (Platform → Category → Service) using the hierarchical picker and save preferences; saved preferences persist across sessions
3. A subscriber can select delivery channels (weekly digest, urgent alerts, deprecation watch) and update those selections from the preferences dashboard at any time
4. The unsubscribe link in the magic link email lands on a page that immediately deactivates the subscription without requiring login
5. `config/taxonomy.yaml` exists in the repo, defines the Platform → Category → Service hierarchy for Azure services, maps Azure RSS category tags to normalized names, includes a `version` field, and has been explicitly reviewed before Phase 4 begins

**Risks**:
- ACS Email has no Go SDK — HMAC-SHA256 request signing must be hand-rolled. Mitigation: budget 1-2 days for this specifically; implement against ACS REST API version `2025-09-01`; test the signed request against the actual ACS endpoint (not a mock) before considering the mail package complete.
- Magic link token security: `math/rand` instead of `crypto/rand`, plaintext storage instead of SHA-256 hash, or multi-use tokens are all exploitable. Mitigation: use `crypto/rand` exclusively for token generation; store only the SHA-256 hash; mark `usedAt` atomically on first valid redemption; add an integration test that redeems the same token twice and asserts the second returns 401.
- Rate limiting omission — magic link endpoint without rate limiting allows inbox flooding and ACS abuse flags. Mitigation: implement rate limiting (5 requests per email per hour) before the endpoint is live.
- Taxonomy improvised rather than designed — if `taxonomy.yaml` is not a concrete deliverable it will be inlined as Go constants in Phase 4, making all downstream features (subscriber preferences, filtering, urgent alerts, deprecation watch) expensive to migrate. Mitigation: treat taxonomy sign-off as a hard gate before Phase 4 planning begins.
- htmx XSS from feed content — `templ.Raw()` on untrusted feed descriptions. Mitigation: audit all `templ.Raw()` usage; escape all feed content before rendering.

**Research flag**: Token storage and session management patterns in Go (including Cosmos DB-backed token documents, atomic `usedAt` marking, and session cookie signing) warrant a focused research pass before implementation.

**Plans**: TBD

---

### Phase 4: AI Classification

**Goal**: Every `RawUpdate` in Cosmos DB is classified exactly once — assigned a type, severity, service tags from the canonical taxonomy, and a plain-language summary — with idempotency, cost controls, and taxonomy drift detection in place from the first run.

**Scope**:
- In: `internal/classifier` pipeline (dequeue → get raw → AI → store enriched), `internal/ai` Claude API client, `internal/queue` Azure Queue Storage wrapper, Container Apps Jobs (Fetcher and Classifier), `classifiedAt` idempotency guard, `DeprecationInfo` struct, unmatched-tag logging and weekly report, dead-letter container for poison messages, queue visibility timeout ≥ 300s, AI output schema validation before persist, Anthropic Batch API for non-urgent workloads, prompt caching for repeated system prompts
- Out: Digest delivery, editorial workflow, urgent alert dispatch (classification output feeds these but they are not built here)

**Depends on**: Phase 3 (`config/taxonomy.yaml` reviewed and complete — hard gate; Phase 2 Cosmos DB and Queue Storage deployed; Phase 1 `Provider` interface and `RawUpdate` struct)

**Requirements**: REQ-040, REQ-041, REQ-042, REQ-043, REQ-044, REQ-045, REQ-046, REQ-047, REQ-048, REQ-049, REQ-093, REQ-102, REQ-106

**Success Criteria** (what must be TRUE when this phase is complete):
1. Running the classifier job against a batch of unclassified updates produces classified documents in Cosmos DB with `type`, `severity`, `serviceTags[]`, `summary`, and `classifiedAt` fields populated and conforming to the allowed enum values
2. Running the classifier job a second time against the same updates makes zero Claude API calls — the `classifiedAt` guard filters them out before any AI invocation (verified by checking API call count, not just output)
3. Updates with `type=deprecation` or `type=retirement` have `DeprecationInfo.RetirementDate` populated when the source provides a date; updates that fail classification are left unclassified with an error logged, not discarded
4. Feed tags that cannot be matched to a taxonomy entry are logged as warnings; a weekly unmatched-tag report is produced as a job output; classified documents include a `taxonomyVersion` field
5. The classifier job checks `DequeueCount` and moves messages exceeding the retry limit to a dead-letter container; queue visibility timeout is set to ≥ 300 seconds

**Risks**:
- AI cost runaway without idempotency guard — re-classifying all stored updates on every poll at Claude Sonnet pricing scales spend with total update count, not new updates. Mitigation: `classifiedAt` guard is implemented and verified before any Phase 4 code ships to production; verify by running the job twice and asserting zero API calls on the second run.
- Taxonomy must be complete before classification starts — classifying against an incomplete taxonomy produces service tags that don't match subscriber preferences, silently breaking all downstream filtering. Mitigation: `config/taxonomy.yaml` sign-off from Phase 3 is a hard prerequisite; do not begin Phase 4 planning until the file is committed and reviewed.
- Dapr not supported on Container Apps Jobs — queue access must go direct via `azqueue/v2` SDK. Mitigation: Bicep and classifier code use direct SDK access only; no Dapr configuration.
- Queue visibility timeout too short — AI latency can exceed 30-second default, causing duplicate classification of the same update. Mitigation: set visibility timeout to 300s; include `DequeueCount` check before processing.
- AI output hallucination — severity `critical` or type `breaking` falsely asserted could trigger downstream alert pipeline. Mitigation: validate all AI output against allowed enum values before persisting; reject and dead-letter responses that don't conform.

**Research flag**: NEEDS `gsd:research-phase` before planning. Prompt engineering for structured JSON output from Claude, Anthropic Batch API mechanics and latency tolerance, idempotency patterns for queue-triggered jobs, and dead-letter implementation in Azure Queue Storage all need a focused spike.

**Plans**: TBD

---

### Phase 5: Weekly Digest

**Goal**: Active subscribers receive a personalized weekly email digest filtered to their selected topics, assembled from classified updates, with bounce and complaint handling configured before any bulk send.

**Scope**:
- In: `internal/digest` assembly (bulk-fetch-per-platform then filter in-memory — not per-subscriber DB queries), Mailer Container Apps Job (cron: Monday 08:00 UTC), `digests` Cosmos DB container (partition key `/subscriberID`), ACS bulk email delivery (batched API calls), bounce and complaint webhook handlers, hard bounce suppression list, List-Unsubscribe header + one-click unsubscribe landing page, digest cached in `digests` container before sending
- Out: Editorial section (Phase 6), urgent alerts (Phase 7), deprecation rollup (Phase 8)

**Depends on**: Phase 4 (classified updates in Cosmos DB), Phase 3 (subscriber preferences and ACS email sending configured), Phase 2 (`digests` container — REQ-034 delivered here)

**Requirements**: REQ-034, REQ-060, REQ-061, REQ-062, REQ-063, REQ-064, REQ-065, REQ-066, REQ-067

**Success Criteria** (what must be TRUE when this phase is complete):
1. A subscriber with selected topics receives a weekly email containing only updates matching their topic selections, with each update showing AI summary, type, severity, source link, and publication date
2. The digest email passes `spf=pass` and `dkim=pass` in received email headers; it renders correctly in Gmail and Outlook (mobile and desktop)
3. The digest assembly job issues at most one Cosmos DB query per platform (not per subscriber) — verified by reading the assembly code and confirming DB reads occur before the subscriber iteration loop
4. Bounce and complaint webhooks from ACS are implemented and tested on a small list (≤ 10 addresses) before any send to a larger list; hard-bounced addresses are suppressed automatically
5. Every digest email includes a functional one-click unsubscribe link that deactivates the subscription without requiring login

**Risks**:
- Digest assembly with per-subscriber DB queries spikes Cosmos DB RU consumption and triggers HTTP 429 throttling mid-job at low subscriber counts. Mitigation: fetch all updates per platform in bulk, filter in-memory per subscriber preferences; validate RU cost on a test dataset before production send.
- ACS account suspended for bounce rate exceeding threshold. Mitigation: implement bounce/complaint webhooks and test deliverability on a small list before any bulk send; never skip this step to ship faster.
- ACS SPF misconfiguration silently bulk-folders digest emails. Mitigation: verify `spf=pass` and `dkim=pass` in actual received email headers before considering the phase complete (same verification as Phase 2, now applied to digest sends).

**Research flag**: Digest assembly query pattern and Cosmos DB RU cost estimation against the actual `azcosmos` query API should be validated before implementation begins. The bulk-fetch-then-filter approach is architecturally sound but RU consumption at scale is directionally estimated, not verified.

**Plans**: TBD

---

### Phase 6: Editorial Workflow

**Goal**: Every weekly digest is preceded by a human-reviewed "this week in infrastructure" editorial section — AI drafts it, a human edits and approves it, and the approved text is prepended to every subscriber's digest automatically.

**Scope**:
- In: Claude API editorial drafting (using `claude-3-5-sonnet`), editorial review UI in the web app (htmx-based), draft/approved/sent state tracked in Cosmos DB, approved editorial prepended to digest assembly, editorial stored per week
- Out: Urgent alerts, deprecation watch (editorial enhances the digest but does not block or replace these)

**Depends on**: Phase 5 (weekly digest pipeline running and delivering), Phase 4 (AI integration via `internal/ai`)

**Requirements**: REQ-070, REQ-071, REQ-072, REQ-073, REQ-074

**Success Criteria** (what must be TRUE when this phase is complete):
1. The web app editorial interface shows an AI-drafted "this week in infrastructure" section, generated from the past week's classified updates, ready for a human editor to review
2. A human editor can read, revise, and approve the draft in the web UI without touching code or the database directly
3. Every subscriber's weekly digest email opens with the approved editorial section above their personalized updates — if no editorial has been approved for that week, the digest sends without it (editorial is additive, not blocking)
4. Editorial documents in Cosmos DB show publish state (`draft`, `approved`, `sent`) and can be audited by week

**Risks**:
- Editorial review UI requires auth — Phase 3 magic link auth must support an editor role or the review interface must be protected behind a separate mechanism. Mitigation: plan the auth model for the editorial interface explicitly (simplest option: a dedicated editor magic link with no subscriber preferences flow).
- Editorial approval must not block digest delivery — if no editorial is approved for a given week, digest still sends. Mitigation: digest assembly treats editorial as optional; absence of an approved editorial for the week is a valid state.

**Research flag**: No additional research needed. AI drafting uses `anthropic-sdk-go` already integrated in Phase 4; web UI patterns follow Phase 3 htmx + templ approach.

**Plans**: TBD

---

### Phase 7: Urgent Alerts

**Goal**: Breaking changes and critical CVEs trigger an out-of-band alert pipeline that routes to a human reviewer before any subscriber receives an email — auto-dispatch is blocked until classification precision is measured and documented.

**Scope**:
- In: Breaking change / critical CVE detection from classifier output, alert review queue in web app UI, human approval gate (no auto-send), alert delivery via ACS scoped to subscribers opted into urgent alert channel, alert triggered by classifier job (not cron)
- Out: Auto-dispatch without human review (explicitly deferred until REQ-083 gate is passed)

**Depends on**: Phase 4 (classification produces `type=breaking` and `severity=critical` tags), Phase 5 (ACS email delivery path established), Phase 3 (subscriber channel preferences — urgent alert opt-in)

**Requirements**: REQ-080, REQ-081, REQ-082, REQ-083, REQ-084

**Success Criteria** (what must be TRUE when this phase is complete):
1. When the classifier tags an update as `type=breaking` or `severity=critical`, it appears in the alert review queue in the web UI within 30 minutes (not waiting for the weekly digest cron)
2. A human reviewer can approve or dismiss each alert from the web UI; only approved alerts are dispatched to subscribers
3. Dispatched alerts are scoped to subscribers who have opted into the urgent alert channel and whose topic selections match the affected platform/service
4. The human review gate cannot be bypassed — there is no code path that sends an alert email without an explicit approval action recorded in the system

**Risks**:
- One false positive alert destroys trust in the alert channel permanently. Mitigation: human review gate is non-negotiable until classification precision is measured on a retrospective dataset; document this constraint in the Phase 7 implementation plan; REQ-083 (relaxing the gate) requires explicit evidence.
- Alert pipeline latency — classifier job runs on queue events, not on a fixed cron; alert must appear in review UI promptly. Mitigation: classifier job event trigger (queue depth = 1) ensures near-real-time processing; add a max review queue age alert to detect stale items.

**Research flag**: Alert delivery mechanics (ACS transactional send path vs. digest batch path) and the web UI pattern for an approval queue should be reviewed before planning.

**Plans**: TBD

---

### Phase 8: Deprecation Watch

**Goal**: Deprecation and retirement updates are tracked through their full lifecycle (announced → active → imminent → retired) and resurfaced in a monthly per-platform rollup that re-alerts subscribers as retirement dates approach — regardless of original publication date.

**Scope**:
- In: Deprecation lifecycle state machine in `internal/store`, state transitions driven by classifier output (`type=deprecation`, `type=retirement`) and approaching `RetirementDate`, monthly rollup Container Apps Job (cron-triggered), per-platform rollup email delivered via ACS to deprecation-watch channel subscribers, retirement timeline resurface logic (items within N days of retirement always included)
- Out: Taxonomy expansion for AWS and Kubernetes deprecations (those happen in Phases 9-10)

**Depends on**: Phase 4 (`DeprecationInfo.RetirementDate` populated for deprecation/retirement updates — REQ-093), Phase 5 (ACS email delivery), Phase 3 (deprecation watch channel opt-in)

**Requirements**: REQ-090, REQ-091, REQ-092

**Success Criteria** (what must be TRUE when this phase is complete):
1. An update classified as `type=deprecation` or `type=retirement` has a lifecycle state (`announced`, `active`, `imminent`, `retired`) stored in Cosmos DB and transitions between states as conditions change
2. Subscribers opted into the deprecation watch channel receive a monthly per-platform email listing active deprecations, with items approaching their retirement date highlighted regardless of when they were originally published
3. An item within 30 days of its `RetirementDate` is included in the next monthly rollup even if it was first announced more than a month ago

**Risks**:
- Deprecation state machine logic is non-trivial — state transitions must be idempotent and driven by both classifier output and time-based conditions. Mitigation: model state transitions explicitly before implementation; use a Cosmos DB TTL policy or a scheduled job for time-based transitions rather than relying on event-driven updates alone.
- `DeprecationInfo.RetirementDate` is not always present in source data — AI must infer or leave blank. Mitigation: treat missing `RetirementDate` as a valid state; resurface logic uses `announced` date as fallback for ordering.

**Research flag**: State machine design for Cosmos DB documents and the TTL / scheduled-job approach for time-based lifecycle transitions warrant a research pass before planning.

**Plans**: TBD

---

### Phase 9: AWS Provider

**Goal**: AWS What's New RSS updates are ingested, normalized to `RawUpdate`, classified through the existing pipeline, and delivered to subscribers who have selected AWS topics — with the taxonomy explicitly extended for AWS services before classification runs.

**Scope**:
- In: `internal/provider/aws` implementing the `Provider` interface, `gofeed` dependency introduced, taxonomy extended for AWS service names in `config/taxonomy.yaml`, AWS updates flowing through existing Fetcher → Queue → Classifier → Digest pipeline, unmatched-tag logging active from day one
- Out: Changes to the classification pipeline (it accepts the new provider without modification if the Provider interface is correctly designed)

**Depends on**: Phase 4 (classification pipeline operational), Phase 3 (taxonomy.yaml versioned and extensible — AWS expansion is a breaking taxonomy change that requires a version bump)

**Requirements**: REQ-007, REQ-012, REQ-013

**Success Criteria** (what must be TRUE when this phase is complete):
1. `./upstream fetch --provider aws` (or equivalent) returns AWS What's New updates normalized to `RawUpdate` structs with the same fields as Azure updates (absent fields are zero-valued, not absent)
2. AWS updates flow through the classifier job and emerge as classified `Update` documents with AWS service tags mapped to normalized taxonomy names; unmatched AWS tags are logged as warnings from the first run
3. A subscriber who selects AWS topics receives AWS updates in their weekly digest alongside (or instead of) Azure updates based on their topic preferences

**Risks**:
- AWS What's New RSS has no structured category tags — classification depends entirely on AI inferring service and type from title and description. Mitigation: validate classification quality on a sample AWS feed before shipping to subscribers; tune the classifier prompt if AWS-specific patterns require it.
- Taxonomy expansion for AWS is a versioned change — existing classified Azure updates reference `taxonomyVersion: N`; the taxonomy bump must not invalidate their service tags. Mitigation: taxonomy changes are additive (new AWS entries); existing Azure entries are unchanged; bump `taxonomyVersion` in `taxonomy.yaml`.

**Research flag**: NEEDS `gsd:research-phase` before planning. AWS What's New RSS feed format, service name normalization strategy against the existing taxonomy, and any adjustments to the classifier prompt for unstructured input (vs. Azure's structured category tags) need research before implementation.

**Plans**: TBD

---

### Phase 10: Kubernetes Provider

**Goal**: Kubernetes release CHANGELOGs and blog RSS updates are parsed and normalized to `RawUpdate`, classified through the existing pipeline, with SIG tags mapped to the canonical taxonomy.

**Scope**:
- In: `internal/provider/kubernetes` implementing the `Provider` interface, CHANGELOG markdown parser (fetching per-release CHANGELOG files from GitHub), Kubernetes blog RSS ingestion as supplementary source, SIG tag normalization added to `config/taxonomy.yaml`, Kubernetes updates flowing through existing pipeline
- Out: Changes to the classification or delivery pipeline

**Depends on**: Phase 9 (or Phase 4 minimum — Kubernetes can be built in parallel with AWS if demand warrants), Phase 3 (taxonomy extensible)

**Requirements**: REQ-008, REQ-009

**Success Criteria** (what must be TRUE when this phase is complete):
1. Kubernetes CHANGELOG section headings from a recent release are parsed into `RawUpdate` structs with correct title, description (the CHANGELOG section body), and published date derived from the release tag
2. Kubernetes blog RSS entries are ingested as supplementary `RawUpdate` structs alongside CHANGELOG entries
3. Kubernetes SIG tags are mapped to normalized taxonomy names; unmatched SIG tags are logged as warnings from the first run

**Risks**:
- Kubernetes CHANGELOG format is markdown, not RSS — parsing is more fragile than XML and depends on the CHANGELOG file structure remaining consistent across releases. Mitigation: write the parser defensively; test against multiple historical releases; treat parsing failures as logged warnings rather than hard errors.
- SIG tag normalization for Kubernetes is a different taxonomy problem than Azure or AWS — SIG names (`sig-network`, `sig-storage`) do not map 1:1 to service categories. Mitigation: research SIG tag taxonomy structure before implementation (see research flag).

**Research flag**: NEEDS `gsd:research-phase` before planning. Kubernetes CHANGELOG markdown structure across releases, GitHub fetch mechanics for per-release files, and SIG tag taxonomy design are not well-explored.

**Plans**: TBD

---

### Phase 11: Polish

**Goal**: New subscribers can preview what they will receive before signing up, the codebase is ready for external contributors, and API documentation is production-quality.

**Scope**:
- In: Onboarding flow for new subscribers (guided topic selection), digest preview page (showing recent real content without requiring signup), API documentation for any public endpoints, production-quality README (build, run, deploy instructions), contributing guide (Bicep-only IaC decision documented, Go conventions, PR process)
- Out: New features — this phase is polish and contributor readiness only

**Depends on**: Phase 5 (digest rendering pipeline produces the preview), all earlier phases complete

**Requirements**: REQ-006

**Success Criteria** (what must be TRUE when this phase is complete):
1. A first-time visitor can view a digest preview page showing real recent content from the past week without creating an account
2. An external contributor can follow the README to build, run locally, and execute the test suite without needing undocumented setup steps
3. The contributing guide explicitly documents the Bicep-only IaC decision, Go naming and error handling conventions, and the PR process

**Risks**:
- Digest preview page using real content requires read-only access to classified updates — ensure no subscriber PII is exposed in the preview. Mitigation: preview shows update content only (title, summary, type, severity, service); no subscriber data is referenced.

**Research flag**: No additional research needed. Standard web UI and documentation patterns.

**Plans**: TBD

---

## Progress

**Execution Order:** 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 0/TBD | Not started | - |
| 2. Azure Deployment | 0/TBD | Not started | - |
| 3. Auth + Subscribers | 0/TBD | Not started | - |
| 4. AI Classification | 0/TBD | Not started | - |
| 5. Weekly Digest | 0/TBD | Not started | - |
| 6. Editorial Workflow | 0/TBD | Not started | - |
| 7. Urgent Alerts | 0/TBD | Not started | - |
| 8. Deprecation Watch | 0/TBD | Not started | - |
| 9. AWS Provider | 0/TBD | Not started | - |
| 10. Kubernetes Provider | 0/TBD | Not started | - |
| 11. Polish | 0/TBD | Not started | - |

---

## Coverage

| Requirement | Phase | Status |
|-------------|-------|--------|
| REQ-001 | Phase 1 | Pending |
| REQ-002 | Phase 1 | Pending |
| REQ-003 | Phase 1 | Pending |
| REQ-004 | Phase 1 | Pending |
| REQ-005 | Phase 1 | Pending |
| REQ-006 | Phase 11 | Pending |
| REQ-007 | Phase 9 | Pending |
| REQ-008 | Phase 10 | Pending |
| REQ-009 | Phase 10 | Pending |
| REQ-010 | Phase 3 | Pending |
| REQ-011 | Phase 3 | Pending |
| REQ-012 | Phase 9 | Pending |
| REQ-013 | Phase 9 | Pending |
| REQ-020 | Phase 1 | Pending |
| REQ-021 | Phase 1 | Pending |
| REQ-022 | Phase 1 | Pending |
| REQ-023 | Phase 1 | Pending |
| REQ-024 | Phase 1 | Pending |
| REQ-025 | Phase 1 | Pending |
| REQ-026 | Phase 1 | Pending |
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
| REQ-113 | Phase 1 | Pending |
| REQ-114 | Phase 1 | Pending |
| REQ-115 | Phase 1 | Pending |
| REQ-116 | Phase 1 | Pending |
| REQ-117 | Phase 1 | Pending |

**Total: 67/67 requirements mapped. No orphans.**

---

*Roadmap created: 2026-03-10*
*Requirements source: .planning/REQUIREMENTS.md (67 requirements, 11 epics)*
*Research source: .planning/research/SUMMARY.md (2026-03-09)*
