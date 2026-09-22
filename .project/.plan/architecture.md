# Architecture

Status: implementation architecture of the canary release.

## Request path

```text
caller: project_id + DecisionIdentity + State + QuestionSet + optional frozen Context
  -> Framework HTTP middleware
  -> bearer auth + project binding + bounded strict JSON decoding
  -> schema + QuestionSet validation + response budget
  -> optional Context rebuild and compare
  -> pinned typed-decision adapter
  -> structural verifier -> sealed bundle
  -> DecisionSet + structural report + trace + replay bundle
```

Replay takes the bundle in a new request, rechecks it, and asks the embedded
Context runtime to rebuild any Context binding from the bundle's own snapshot
and pack request. It never calls a provider or retrieval service.

## Packages

```text
cmd/api                wiring: env -> adapter -> httpapi.Config
internal/httpapi       HTTP boundary: auth, negotiation, budgets, stages, problems
internal/wire          schemas, typed records, hashes, bundle sealing, replay verifier
internal/verify        answer checks and the finding-code catalog
internal/lifecycle     legal stage graphs for decision and replay
internal/canonical     RFC 8785 + SHA-256, the only hashing path
internal/evidence      strict decode of a frozen Context state and its rebuild
internal/adapters/*    provider transport behind httpapi.Decider
internal/conformance   obligation map and core boundary tests
```

Dependencies point downward. Provider details stay in `internal/adapters`.
Context public types stay in `internal/evidence`. No upstream `internal/`
imports.

## Deployment profile

One `net/http` handler from `cmd/api/main.go`, listening on `PORT`, with the
Vercel Go preset. Warm instances reuse only immutable configuration and HTTP
transports; requests may land on any instance. No background work or
correctness-critical post-response step.

Application data lives in request RAM; nothing is written to `/tmp` or mounted
storage. Platform access logs and provider-side retention are outside LeX and
are not claimed as zero retention. Evidence, credentials, and raw answers never
enter logs.

Configuration: `LEX_BEARER_TOKENS` (token to project list),
`LEX_TYPESAFE_API_KEY` (direct adapter), `LEX_OPENROUTER_API_KEY` with
`LEX_HOSTED_RESOLVED_MODEL` (hosted adapter), and `LEX_DECISION_ADAPTER` when
both are configured.

## Budgets

| Budget | Value |
|--------|-------|
| Request and response body | 2 MiB each |
| JSON depth | 32 |
| Questions / Choice options / Score levels | 64 / 255 / 10 |
| Context source text | 256 KiB |
| Request deadline | 20 s; server write timeout 5 s longer |
| Process admission | 4 concurrent requests, no queue |
| Provider retries | none |

The response budget is checked before the provider call and the provider body
is read only up to what still fits. Nothing is truncated to produce a
success-shaped response.

## SLO targets

Targets for the stable release, not measured performance: replay availability
99.9%, decision availability 99.5% including the provider, warm replay p95
250 ms, decision p95 10 s and p99 20 s, cold replay p95 3 s, over a rolling
28-day window from platform telemetry and an external probe. A technical
failure counts as bad; missing telemetry is reported as unknown.
