# Upstream

## What This Is

Upstream is an open-source, AI-enhanced infrastructure change awareness platform that aggregates updates from major cloud hyperscalers (Azure, AWS) and Kubernetes, classifies them with AI, and delivers personalized newsletter digests to subscribers. It runs as a hosted SaaS at `upstream.engineeringclouds.io` while the code is public on GitHub (`engineeringclouds/upstream`). It serves infrastructure practitioners who need a signal-to-noise filter across the sprawling update streams of the platforms they operate.

## Core Value

The editorial layer: a human-reviewed, opinionated "this week in infrastructure" section delivered in the Engineering Clouds voice — AI does the volume work, but a human shapes what matters and why.

## Requirements

### Validated

<!-- Shipped and confirmed valuable. -->

(None yet — ship to validate)

### Active

<!-- Current scope. Building toward these. Phase numbers align with the planned build order. -->

**Phase 1 — Azure CLI (Foundation)**
- [ ] CLI fetches Azure Updates RSS feed and displays formatted output to stdout
- [ ] `--since` flag filters updates by date
- [ ] `--format` flag controls output format
- [ ] Meaningful test coverage for XML parsing logic
- [ ] Code passes `go vet` and `gofmt` clean

**Phase 2 — Azure Deployment**
- [ ] Updates persisted to Cosmos DB (serverless)
- [ ] Application deployed to Azure Container Apps via Bicep IaC
- [ ] Basic web UI (no auth) publicly accessible

**Phase 3 — Auth + Subscriber Management**
- [ ] Magic link authentication via Azure Communication Services
- [ ] User signup and account creation
- [ ] Hierarchical topic selection (Platform → Category → Service)
- [ ] Preferences dashboard (htmx-based)
- [ ] Channel selection (weekly digest, urgent alerts, deprecation watch)

**Phase 4 — AI Classification Pipeline**
- [ ] Updates classified by type (feature, deprecation, security, preview, breaking, retirement)
- [ ] Updates classified by severity (informational, action-recommended, action-required, critical)
- [ ] Service tagging normalized via canonical taxonomy config
- [ ] Classification cached per update (not per subscriber)

**Phase 5 — Weekly Digest + Email Delivery**
- [ ] Weekly digest assembled from classified updates filtered to subscriber topics
- [ ] Digest delivered via Azure Communication Services
- [ ] Personalized digest section per subscriber below the editorial

**Phase 6 — Editorial Workflow**
- [ ] AI drafts the editorial "this week in infrastructure" section
- [ ] Human review and editing interface before publish
- [ ] Editorial section prepended to every subscriber's weekly digest

**Phase 7 — Urgent Alerts**
- [ ] Breaking changes and critical CVEs trigger real-time(ish) alert pipeline
- [ ] Human review gate before dispatch (no auto-send until classification is trusted)
- [ ] Scoped to subscriber's selected platforms

**Phase 8 — Deprecation Watch**
- [ ] Deprecation lifecycle tracked (announced → active → imminent → retired)
- [ ] Monthly rollup digest delivered per platform
- [ ] Timeline tracking resurfaces items approaching retirement

**Phase 9 — AWS Provider**
- [ ] AWS What's New RSS feed ingested and normalized
- [ ] AI classification essential (no structured type/service tags in source)

**Phase 10 — Kubernetes Provider**
- [ ] Kubernetes CHANGELOG markdown files parsed (not RSS)
- [ ] Kubernetes blog RSS ingested
- [ ] SIG tags mapped to normalized service taxonomy

**Phase 11 — Polish**
- [ ] Onboarding flow for new subscribers
- [ ] Digest preview before subscribing
- [ ] API documentation
- [ ] README and contributing guide production-quality

### Out of Scope

- Real-time chat — not core to the value proposition
- Mobile app — web-first; native app deferred indefinitely
- OAuth / Azure AD B2C — magic link sufficient for v1; no passwords
- Terraform — Bicep is the chosen IaC; portability not a goal
- Multi-region deployment — single Azure region sufficient at this scale
- Per-subscriber AI summarization — summaries are cached per update, not regenerated per subscriber

## Context

- **OSS portfolio project** for Engineering Clouds LLC — demonstrates cloud-native Go architecture publicly
- **Go learning project** — primary developer has zero Go experience entering Phase 1; idiomatic patterns take priority over shipping speed
- **Azure-native** — no portability abstractions; build directly against Azure SDKs and services
- **Two-stage ingestion pipeline** — providers fetch raw data (no classification); a centralized AI pipeline enriches `RawUpdate` → `Update`; summaries cached at update level
- **Source complexity varies significantly** — Azure is best-structured (explicit category tags, status prefixes); AWS requires AI for everything; Kubernetes requires markdown parsing, not feed consumption
- **AI cost optimization built in** — summarize once per update, apply voice profile per subscriber at assembly time

## Constraints

- **Go experience**: Zero entering Phase 1 — expect slower initial progress; explain non-obvious patterns
- **Azure-native deployment**: Azure Container Apps + Cosmos DB (serverless) + Queue Storage + Azure Communication Services
- **IaC**: Bicep only. Not Terraform.
- **Cost target**: Under $40/month at moderate scale
- **Human review**: Urgent alerts require human approval before dispatch — no auto-send until classification is trusted
- **Standard library preference**: Justify any external dependency; minimize third-party imports

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go | Cloud-native lingua franca, strong concurrency model, single binary deployment. Also the learning goal. | — Pending |
| htmx + templ for web UI | Stay in one language (Go), no Node toolchain, sufficient for subscriber preference management | — Pending |
| Azure Container Apps | Scale-to-zero, built-in HTTPS, Container Apps Jobs for scheduled workloads | — Pending |
| Cosmos DB serverless | Document-oriented model, pay-per-request pricing, near-zero cost at low volume | — Pending |
| Magic link auth (ACS) | No passwords, simpler than Azure AD B2C for v1, ACS handles both auth email and digest delivery | — Pending |
| Claude API for AI | Summarization + classification. Azure OpenAI as future alternative if needed. | — Pending |
| Azure Queue Storage | Decouples feed polling from AI classification processing | — Pending |
| Bicep for IaC | Not Terraform — Azure-native, primary IaC language at Engineering Clouds | — Pending |
| Two-stage ingestion | Providers fetch raw; classifier enriches separately — keeps providers simple and testable | — Pending |
| Editorial-first positioning | Curated voice is the differentiator vs. raw RSS aggregators; AI is table stakes | — Pending |

---
*Last updated: 2026-03-09 after initialization*
