# Scope

Status: normative boundary, canary release. `[x]` is built and tested, `[ ]`
is open until the stable release, `[-]` is refused so the protocol stays thin.
Open proof items are tracked in [progress.md](../.plan/progress.md).

## Decision

- [x] Caller-owned State and versioned QuestionSet with Noul, Choice, and Score
- [x] Up to 64 questions, 255 Choice options, 10 Score levels
- [x] Raw DecisionSet with adapter id, adapter version, and resolved model
- [x] Provider request id, timing, and usage returned when supplied, never sealed
- [x] Exact model pin; aliases refused; no silent model or provider fallback
- [x] Direct adapter live; hosted adapter passes local conformance fixtures
- [ ] Hosted adapter proven on the deployment
- [ ] Per-adapter calibration report
- [-] Domain profiles, question catalogues, or automatic question generation
- [-] Thresholds, verdicts, or next-action selection inside LeX
- [-] Provider orchestration, fan-out, voting, or retries

## Evidence

- [x] Optional frozen Context binding, rebuilt by Context and compared before the provider call
- [x] Context never merged into State
- [-] Retrieval, crawling, URL fetching, or building a pack from raw text inside LeX
- [-] A second retrieval engine, dense or fuzzy search, or morphology in LeX

## Verification and replay

- [x] JCS (RFC 8785) and SHA-256 hashes, checked against independent Python vectors
- [x] Structural verifier with stable finding codes
- [x] Self-hashed replay bundle; tampering detected
- [x] Replay without network, provider, or retrieval; fresh-instance safe
- [x] Replay refused outside the principal's projects
- [ ] Race evidence on a gcc-capable runner
- [-] Bundle signatures, issuer authentication, or OIDC
- [-] Server-side history, run lookup, durable jobs, or idempotency store

## HTTP surface

- [x] `GET /healthz`, `GET /v1/capabilities`, `POST /v1/decisions`, `POST /v1/replays`
- [x] JSON Schema 2020-12 and OpenAPI 3.1.1, validated against live responses in tests
- [x] RFC 9457 problems with stable reasons and JSON-pointer diagnostics
- [x] Bearer tokens bound to projects; CORS and cookies refused
- [x] Body, depth, response, deadline, and admission budgets
- [x] Client guide and portable agent skill
- [-] SDKs, gRPC, streaming, or asynchronous messaging

## Operations

- [x] One Go handler on Vercel, request RAM only, no filesystem writes
- [x] Revision-pinned canary with a named rollback deployment
- [ ] SLO measurements over a 28-day window
- [ ] Vulnerability review repeated per release
- [ ] Deployed toolchain, region, and cold start recorded
- [-] Database, queue, distributed workers, or billing

## Outside this release

- [-] Executor, operation contract, receipt, or any side effect
- [-] Product, chat, or agent shell on top of the protocol
