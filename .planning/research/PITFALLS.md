# Pitfalls Research

**Domain:** AI-enhanced infrastructure change awareness platform (Go, Azure-native, newsletter digest)
**Researched:** 2026-03-09
**Confidence:** HIGH (Azure-specific items verified against official docs; Go patterns from authoritative sources; AI cost patterns from Anthropic official pricing; email deliverability from RFC-grounded sources)

---

## Critical Pitfalls

### Pitfall 1: RSS Feed Date Parsing Failures Silently Drop Updates

**What goes wrong:**
Azure Updates RSS uses non-standard date formats and occasionally malformed pubDate fields. Go's `encoding/xml` unmarshals XML strictly but `time.Parse` fails silently when the format string doesn't match — the field stays at zero value, `--since` comparisons treat it as epoch, and valid updates are either always included or always excluded depending on comparison direction.

**Why it happens:**
Developers assume RSS dates are RFC 1123 or RFC 2822 and write a single `time.Parse` call. Azure's feed mixes formats across entries and occasionally emits dates with or without timezone offsets.

**How to avoid:**
Write a `parseDate` helper that attempts multiple formats in order (`time.RFC1123Z`, `time.RFC1123`, `time.RFC3339`, plus common RSS variants like `Mon, 02 Jan 2006 15:04:05 -0700`). Log a warning with the raw string when all formats fail. Never silently zero-date an entry. Add a dedicated test with a `testdata/` fixture that includes malformed date entries.

**Warning signs:**
- `--since yesterday` returns zero results despite live feed having entries
- Test coverage skips the date parsing path entirely
- The `pubDate` field type is `string` rather than a parsed `time.Time`

**Phase to address:** Phase 1 (Foundation — RSS parsing)

---

### Pitfall 2: AI Classification Cost Runaway from Repeated Re-Classification

**What goes wrong:**
Every feed poll re-classifies updates that were already classified. At Claude Sonnet pricing (~$3/MTok input, ~$15/MTok output), classifying 50 Azure updates per poll × 8 polls/day × 365 days = ~146,000 classifications/year. At ~500 tokens per classification round-trip, that's 73M tokens/year — hundreds of dollars annually for a $40/month budget target.

**Why it happens:**
The two-stage ingestion pipeline is architecturally correct (raw fetch → classify separately) but without a "classified" flag on stored updates, the classifier job has no idempotency guard and re-processes everything on each run.

**How to avoid:**
Store a `classifiedAt` timestamp (or boolean) on each `RawUpdate` document in Cosmos DB. The classifier job queries only `WHERE classifiedAt = null`. Add a dead-letter mechanism for updates that fail classification after N retries rather than retrying them infinitely. Use Anthropic's Batch API (50% cost reduction) for non-urgent classification workloads.

**Warning signs:**
- Cosmos DB query for "unclassified updates" returns the full update count on every run
- No `classifiedAt` or `classificationVersion` field in the data model
- AI API spend increases proportionally with total update count, not with new update count

**Phase to address:** Phase 4 (AI Classification Pipeline)

---

### Pitfall 3: Cosmos DB Serverless RU Spike on Digest Assembly

**What goes wrong:**
Weekly digest assembly queries subscriber preferences, then for each subscriber queries updates filtered by their topics. With 100 subscribers each watching 10 topics, the naive implementation executes 100 × 10 = 1,000 point queries at digest time. Each cross-partition query in Cosmos DB serverless can consume 10-100+ RUs. A single digest run can spike to hundreds of thousands of RUs — potentially $0.025–$0.25 in a single job execution, and the 5,000 RU/s serverless burst cap causes HTTP 429 throttling mid-job.

**Why it happens:**
Developers model subscriber preferences as a join between `subscribers` and `updates` and execute it imperatively in a loop rather than designing the query access pattern around the partition key.

**How to avoid:**
Model `updates` with a partition key of `/platform` (or `/platform/category`). Assemble the digest by fetching all updates for each platform in a single query, then filter in-memory per subscriber preference. This turns N×M queries into N platform queries (3 platforms initially). Cache the assembled platform update lists for the digest window to avoid redundant reads. Set a Cosmos DB alert threshold at 80% of expected monthly RU budget.

**Warning signs:**
- Data model has no defined partition key strategy before Phase 2
- Digest assembly function contains a loop with a database query inside it
- No RU consumption logging in digest job output

**Phase to address:** Phase 2 (Azure Deployment — data model design) — the partition key cannot be changed after collection creation without a full data migration.

---

### Pitfall 4: Magic Link Tokens Not Single-Use or Expiry Too Long

**What goes wrong:**
Magic link tokens that can be used multiple times, or that expire after 24+ hours, expose subscribers to account takeover if the email is forwarded, leaked in email logs, or accessed by a shared inbox. Newsletter-focused platforms are lower-value targets but the exploit is trivial: anyone with the email link URL gains full account access.

**Why it happens:**
Developers implement the "happy path" (generate token, send email, validate token) and defer security hardening. Tokens get stored but never marked consumed. Expiry is set long to reduce support burden.

**How to avoid:**
Enforce exactly: (1) cryptographically random token via `crypto/rand` (not `math/rand`), (2) 15-minute TTL stored as `expiresAt` in Cosmos DB, (3) single-use — mark `usedAt` atomically on first valid redemption, reject any subsequent use of the same token. Store only the token hash, not the raw value. Add rate limiting on the `/auth/send-magic-link` endpoint (5 requests per email per hour) to prevent token flooding.

**Warning signs:**
- Token generated with `math/rand` or `rand.Int()`
- No `usedAt` field on the token document
- Token expiry is configurable and defaults to hours
- `/auth/send-magic-link` has no rate limiting middleware

**Phase to address:** Phase 3 (Auth + Subscriber Management)

---

### Pitfall 5: Email Deliverability Collapses When Sending Volume Scales

**What goes wrong:**
Initial sends work fine because volume is low and ISPs give new senders a grace period. At ~500+ subscribers, ISPs begin applying reputation scoring. If SPF/DKIM/DMARC are not configured correctly on the sending subdomain, Gmail and Outlook bulk-folder digest emails. Subscribers stop seeing digests, assume the service is down or their preferences broken, and unsubscribe. A 5%+ bounce rate in Azure Communication Services triggers account review; 10% can cause suspension.

**Why it happens:**
Developers treat email sending as "point ACS at a domain and it works." Azure Communication Services requires exact SPF record format (`v=spf1 include:spf.protection.outlook.com -all`) and two DKIM CNAME records. The portal shows "Not configured" even when DNS is correctly propagated, causing developers to second-guess their configuration.

**How to avoid:**
Use a dedicated subdomain exclusively for ACS sends (e.g., `mail.upstream.engineeringclouds.io`). Configure SPF, both DKIM and DKIM2 CNAMEs, and a DMARC policy (`p=quarantine` initially) before sending the first production email. Verify using email header inspection (`spf=pass`, `dkim=pass`) rather than trusting the portal status indicator. Add bounce and complaint webhooks from ACS to automatically suppress addresses that hard-bounce. Never reuse the marketing domain for transactional sends.

**Warning signs:**
- Sending domain is the root domain, not a subdomain
- No DMARC record published
- No bounce/complaint webhook handler implemented
- First digest is sent to 200+ subscribers before deliverability validation on a small test list

**Phase to address:** Phase 2 (Azure Deployment — infrastructure setup) for DNS/domain config; Phase 5 (Weekly Digest) for bounce/complaint handling

---

### Pitfall 6: Taxonomy Drift Makes Subscriber Preferences Stale

**What goes wrong:**
Azure renames services (e.g., "Azure Container Instances" → something else), adds new service categories, or restructures the taxonomy visible in the RSS feed. The canonical taxonomy config hard-coded in Phase 4 becomes misaligned. Existing subscribers who selected "AKS" receive nothing because updates are now tagged "Azure Kubernetes Service" with a different normalized key. The platform silently delivers empty or wrong sections of the digest.

**Why it happens:**
Taxonomy is modeled as a static enum or constant file. Feed category strings are normalized against it at classification time. No monitoring detects when an incoming feed tag fails to match any taxonomy entry.

**Why it matters more for this project:**
AWS has zero structured tags — AI must infer all classification. Kubernetes uses SIG names that change between releases. Azure's categories are the most structured but still subject to Microsoft's nomenclature changes. Phase 9 and 10 multiply this risk significantly.

**How to avoid:**
Store the taxonomy in a versioned config file (`config/taxonomy.yaml`), not Go constants. Log a warning metric (or write to a dead-letter document) whenever a feed tag cannot be matched to a taxonomy entry. Include a `taxonomyVersion` field on classified updates. Build a weekly report of unmatched tags as a Phase 4 deliverable so drift is caught before subscribers notice empty digests.

**Warning signs:**
- Taxonomy is defined as `const` or `iota` in Go source
- No logging when a feed category tag fails normalization
- First AWS or Kubernetes provider added without updating the taxonomy config

**Phase to address:** Phase 4 (AI Classification Pipeline) for taxonomy config structure; Phase 9 and 10 require explicit taxonomy expansion tasks

---

### Pitfall 7: Go Beginner Error Handling Patterns That Become Systemic Debt

**What goes wrong:**
A Go-learning developer establishes error handling patterns in Phase 1 that propagate across the codebase. Three specific anti-patterns are high-risk: (1) `if err != nil { log.Fatal(err) }` in library code (kills the process rather than returning an error), (2) `fmt.Println(err); return` without returning the error to the caller (swallows errors), (3) `errors.New("failed")` without context (useless in production logs). Once these patterns exist in 10 files, fixing them is a refactor rather than a code review.

**Why it happens:**
Early Go tutorials use `log.Fatal` for brevity. The developer doesn't yet have the intuition for the call-stack boundary between "library code that returns errors" and "main.go that terminates on fatal errors."

**How to avoid:**
Establish the pattern in Phase 1 and never deviate: all functions in `internal/` return errors wrapped with `fmt.Errorf("context: %w", err)`. Only `main.go` and `cmd/` call `log.Fatal`. Never swallow errors — if you can't handle it, wrap and return it. Add `go vet` and `errcheck` to the CI pipeline in Phase 1 before any other phases add code.

**Warning signs:**
- `log.Fatal` appears in any file outside `cmd/` or `main.go`
- Error variables named `e` or `er` instead of `err`
- Functions with `error` return type that never return a non-nil error (but silently discard them internally)
- `_ = someFunc()` discarding error returns

**Phase to address:** Phase 1 (Foundation) — establish patterns before they proliferate

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hard-code taxonomy as Go constants | Faster Phase 4 delivery | Requires code change + redeploy for every Azure service rename; breaks subscriber preferences silently | Never — use config file from the start |
| Single Cosmos DB container for all document types | Simpler schema, faster initial setup | Cannot change partition key later without full migration; cross-type queries are expensive | Never — define container-per-entity-type in Phase 2 |
| Store raw AI API response as freeform JSON | Flexible during exploration | Classification downstream must parse untyped JSON; schema changes break silently | Only during Phase 4 spike/exploration, never in production |
| Skip bounce/complaint webhooks until "later" | Faster Phase 5 delivery | ACS account suspension at scale; undeliverable emails counted as sends | Never — implement before first bulk send |
| `math/rand` for token generation | Simpler code | Cryptographically predictable tokens; account takeover risk | Never — `crypto/rand` is not meaningfully harder |
| Classify updates inline in the provider goroutine | Fewer moving parts initially | AI latency blocks feed polling; classification errors break fetching; tightly coupled | Only as a Phase 1 exploration spike, never in production pipeline |
| Re-use the Go `html/template` stdlib for email rendering | No dependency | Email clients strip CSS, ignore most HTML features; template debugging is painful | Acceptable for Phase 5 v1 if the template is tested against real clients |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Azure Communication Services — SPF | Using root domain or multi-include SPF record | Dedicated subdomain; exact SPF: `v=spf1 include:spf.protection.outlook.com -all`; configure both DKIM and DKIM2 CNAMEs |
| Azure Communication Services — portal status | Treating "Not configured" in portal as broken | Verify via email header inspection (`spf=pass`, `dkim=pass`), not portal badge |
| Cosmos DB — partition key | Choosing partition key post-data-model finalization | Define partition key strategy before writing a single document; it cannot be changed without full migration |
| Cosmos DB — serverless burst | No retry logic on HTTP 429 responses | Implement exponential backoff with jitter on all Cosmos DB writes and queries; use the Azure SDK's built-in retry policy |
| Azure Queue Storage — message visibility | Not extending visibility timeout for long-running classification jobs | If AI classification exceeds the default 30-second visibility timeout, the message becomes visible again and is processed twice; extend visibility or use Container Apps Jobs with longer execution limits |
| Azure Queue Storage — poison messages | Ignoring `DequeueCount` | After N failures (5 is reasonable), move message to a dead-letter container; alert on dead-letter queue depth |
| Claude API — prompt caching | Not using prompt caching for repeated system prompts | Classification prompts with static system context qualify for Anthropic's prompt caching (90% cost reduction on cached tokens); pass the same system prompt hash consistently |
| Azure RSS feed — encoding | Assuming UTF-8 throughout | Azure's RSS feed is UTF-8 but some older entries contain HTML entities that `encoding/xml` strict mode rejects; use a lenient XML decoder or pre-process with `golang.org/x/net/html` |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Per-subscriber AI summarization | AI spend grows linearly with subscriber count | Cache summaries per update, not per subscriber; apply voice profile at assembly time in Go (no AI call) | First week if naively implemented |
| Digest assembly loop with DB queries | Digest job takes minutes, RU budget exceeded, HTTP 429 mid-job | Fetch updates per platform in bulk, filter in-memory per subscriber preferences | ~50 subscribers with 5+ topics each |
| No pagination on RSS fetch | Memory spike when Azure publishes a large batch; timeout on slow connections | Implement streaming XML parse via `xml.Decoder.Token()` rather than full unmarshal | Batch updates during Azure announcements (Build, Ignite) |
| Cosmos DB cross-partition query for recent updates | Query fan-out across all partitions, high RU cost | Partition updates by platform; always include partition key in query predicates | As update count grows past a few thousand |
| Container Apps Job cold start for digest pipeline | Digest delivery delayed by 30-60 seconds of container startup | Pre-build minimal binary images; set `minReplicas: 1` for the web tier (free on Container Apps); accept cold start only for scheduled jobs | Not a blocking issue at low volume but visible to subscribers |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Magic link token stored as plaintext in Cosmos DB | If Cosmos DB is breached, all pending magic links are immediately exploitable | Store SHA-256 hash of token; compare hash on redemption |
| Magic link endpoint with no rate limiting | Attacker floods subscriber's inbox with magic link emails; ACS account flagged for abuse | Rate limit: 5 magic link requests per email per hour using an in-memory or Cosmos DB backed counter |
| Subscriber email address exposed in URL params or logs | PII exposure, GDPR risk | Never log email addresses; use subscriber ID in URLs; use ACS opaque message IDs in logs |
| AI classification results treated as trusted for business logic without validation | Hallucinated severity `critical` triggers alert pipeline; wrong classification drives subscriber filtering | Validate AI output against the enum of allowed values before persisting; reject and dead-letter responses that don't conform to schema |
| htmx + templ raw HTML injection from feed content | XSS if feed `description` fields rendered as `templ.Raw()` | Sanitize all feed content with `html.EscapeString()` or a whitelist-based HTML sanitizer before rendering; never use `templ.Raw()` on untrusted content |
| Cosmos DB connection string in environment variables logged at startup | Credentials in cloud logs | Log config at startup but redact all values containing "key", "secret", "password", "connection" |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Topic preference UI shows flat list of all Azure services (~200+) | Overwhelming; subscribers select nothing or everything | Hierarchical three-level picker: Platform → Category → Service; default to all selected; let subscribers narrow down |
| Digest sent at midnight UTC regardless of subscriber timezone | Subscribers receive digest at inconvenient local times | Store timezone preference; default to 08:00 subscriber local time; UTC is acceptable for v1 with a note in onboarding |
| Empty digest section ("No updates for your topics this week") without explanation | Subscribers think something is broken | If a subscriber's digest section would be empty, include the top 3 updates from their platform regardless of topic match, labeled "You might also be interested in" |
| Magic link expiry not communicated in the email | Subscriber clicks link 20 minutes later, gets an error with no explanation | Email body must state "This link expires in 15 minutes" prominently; expired link page must offer one-click resend |
| No preview of digest before first subscribe | Subscribers can't calibrate expectations; churn on first delivery | Phase 11 digest preview is deprioritized — but a static example digest in the marketing copy is a cheap substitute |

---

## "Looks Done But Isn't" Checklist

- [ ] **RSS parser:** Tested against real Azure Updates RSS fixture with actual malformed dates, HTML entities in descriptions, and missing optional fields — verify `go test ./...` uses `testdata/` fixtures, not mocked strings
- [ ] **Classification idempotency:** Re-running the classifier job against already-classified updates produces zero AI API calls — verify by checking `classifiedAt` guard before any Claude API invocation
- [ ] **Magic link auth:** Token is single-use and marked consumed atomically — verify with a test that redeems the same token twice and asserts the second redemption fails
- [ ] **Email deliverability:** First production send passes `spf=pass` and `dkim=pass` in actual received email headers — verify before sending to a list larger than 10 addresses
- [ ] **Cosmos DB partition strategy:** The partition key is defined before Phase 2 data model is deployed — verify that the Bicep definition includes explicit `partitionKey` on every container
- [ ] **Digest assembly:** Digest job does not issue a database query inside a per-subscriber loop — verify by reading the assembly code and confirming DB reads happen before the subscriber iteration
- [ ] **Queue poison messages:** Dead-letter container exists and classifier job checks `DequeueCount` before processing — verify in integration test
- [ ] **htmx error responses:** Form submissions return partial HTML fragments (not full page reloads) with appropriate HTTP status codes — verify by testing with JavaScript disabled and with HTMX request headers

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Wrong Cosmos DB partition key in production | HIGH | Create new container with correct partition key; migrate data via Azure Data Factory or a one-off Go migration script; update all application query paths; redeploy |
| AI cost runaway (already billed) | MEDIUM | Add `classifiedAt` guard immediately; set Anthropic API spend alert; review billing to identify the runaway query; no data loss |
| ACS email account suspended for bounces | HIGH | Clean subscriber list against email verification API; implement bounce webhook before re-applying for sending quota restoration; allow 3-5 business days for review |
| Magic link tokens compromised | HIGH | Invalidate all pending tokens (delete documents from Cosmos DB token collection); force all subscribers to re-authenticate; rotate the token signing secret if one was used |
| Taxonomy drift causing empty digests | MEDIUM | Add unmatched-tag logging retroactively; audit recent unmatched tags; update taxonomy config; re-classify affected updates (re-runs classifier on `classificationVersion < current`) |
| Go error swallowing patterns discovered late | MEDIUM | Systematic grep for `_ =`, `log.Fatal` outside cmd/, and bare `return` after print; fix in a single refactor PR before adding new features |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| RSS date parsing failures | Phase 1 | `go test ./...` passes with `testdata/` fixture containing malformed dates |
| AI cost runaway from re-classification | Phase 4 | Classifier job query filters on `classifiedAt = null`; confirmed by running job twice and checking AI API call count = 0 on second run |
| Cosmos DB RU spike on digest assembly | Phase 2 (data model), Phase 5 (query design) | Digest job executes ≤ (number of platforms) DB queries regardless of subscriber count |
| Magic link single-use and expiry | Phase 3 | Integration test: second redemption of used token returns 401 |
| Email deliverability configuration | Phase 2 (DNS), Phase 5 (bounce handling) | SPF and DKIM pass verified in email headers before any bulk send |
| Taxonomy drift | Phase 4 | Unmatched-tag logging in place; taxonomy sourced from versioned config file |
| Go error handling debt | Phase 1 | `errcheck` lint passes; no `log.Fatal` outside `cmd/`; all error returns wrapped |
| Partition key lock-in | Phase 2 | Bicep template reviewed for explicit partition key on all containers before first deploy |
| Queue poison messages | Phase 4 (classifier) | Dead-letter container defined in Bicep; `DequeueCount` checked in classifier |
| htmx XSS from feed content | Phase 3 (web UI) | `templ.Raw()` usage audited; feed descriptions escaped before render |

---

## Sources

- Azure Communication Services SPF/DKIM requirements: https://learn.microsoft.com/en-us/azure/communication-services/concepts/email/email-domain-and-sender-authentication
- Azure Communication Services domain troubleshooting: https://learn.microsoft.com/en-us/azure/communication-services/concepts/email/email-domain-configuration-troubleshooting
- Cosmos DB serverless limits and burst cap: https://learn.microsoft.com/en-us/azure/cosmos-db/serverless-performance
- Cosmos DB multi-tenancy partitioning: https://learn.microsoft.com/en-us/azure/architecture/guide/multitenant/service/cosmos-db
- Cosmos DB throughput vs serverless comparison: https://learn.microsoft.com/en-us/azure/cosmos-db/throughput-serverless
- Anthropic Claude API pricing (prompt caching, batch): https://platform.claude.com/docs/en/about-claude/pricing
- Azure Queue Storage visibility timeout and poison messages: https://learn.microsoft.com/en-us/answers/questions/26918/azure-queue-storage-dequeue-behavior-timeout-and-d
- Azure Container Apps cold start: https://learn.microsoft.com/en-us/azure/container-apps/cold-start
- Azure Container Apps Jobs: https://learn.microsoft.com/en-us/azure/container-apps/jobs
- Magic link security best practices: https://guptadeepak.com/mastering-magic-link-security-a-deep-dive-for-developers/
- Magic link vulnerability (deep link token interception): https://www.dfns.co/article/the-magic-link-vulnerability
- Common Go mistakes: https://100go.co/
- Go common mistakes wiki: https://go.dev/wiki/CommonMistakes
- htmx + templ pitfalls: https://emilytburak.net/posts/2025-06-09-htmx-golang-learnings/
- Email deliverability state 2025: https://www.mailgun.com/blog/deliverability/state-of-deliverability-takeaways/

---
*Pitfalls research for: AI-enhanced infrastructure newsletter platform (Go + Azure)*
*Researched: 2026-03-09*
