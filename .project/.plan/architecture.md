# Architecture and Vercel profile

Status: implementation architecture of the generic typed-decision and legacy
claim-validation compatibility slices under the fixed scope in
[README.md](README.md). The capability checklist at the end of this file
states what is built, what is open, and what is deliberately kept out of the
protocol.

## Request path

```text
REST caller: project + DecisionIdentity + State + QuestionSet + optional Context frozen state
  -> Framework HTTP middleware
  -> authentication + project binding + bounded decoding
  -> typed-decision adapter -> frozen DecisionSet
  -> structural verifier -> DecisionSet + DecisionTrace + replay bundle
  -> caller decides any subsequent retrieval, tool, or action
```

Replay uses the caller-retained bundle. It does not contact a retrieval service
or a decision provider. When a Context binding exists it asks the embedded
Context runtime to rebuild the snapshot and pack from the frozen sources and
pack request, then compares that result with the saved state. The saved pack is
not replaced, and LeX does not recompute selection, budgeting, or rejection.

Profiles are compatibility-only objects: a profile pins its question set,
policy document and gate, entity kind, and Context focus. The generic decision
path has no profile registry; it validates a caller-defined QuestionSet and
returns structural results only.

Core logic uses Go domain types and explicit ports. Framework owns HTTP
composition. Context public types and compatibility checks stay in the evidence
adapter. Provider details stay in decision adapters. No upstream internal imports.

Deployment-owned policy and profiles are immutable embedded configuration or
validated configuration loaded at initialization. Caller-provided policy is
data to verify against authenticated authority, never permission to lower gates.
Request-local maps and byte buffers own all entity, evidence, and answer state.

## Embedded Context contract

Use the [pinned Context baseline](context-version.md), capability memory-exact-v1.
Create one runtime from approved source inputs per evaluation and call its
ContextPack method with explicit focus controls. Map the supported profile
subset without silently dropping requirements. The runtime performs exact
phrase retrieval only; it is not a dense or morphology-enabled runtime.

Export Snapshot and exact PackRequest alongside the frozen pack. Preserve its
upstream checksum separately from the LeX canonical envelope hash. Source trust
and evidence classes come from authenticated provenance mapping, not unchecked
caller labels. HTTP contextkit.Client is not on the RAM-only execution path.

## Vercel integration

Use one Go HTTP application, exposing the same `http.Handler` locally and on
Vercel. Prefer the Go framework preset and `cmd/api/main.go`, listening on
`PORT`. A thin `api/*.go` handler is an alternative only if the deployment
probe selects it; do not maintain two independent routing implementations.

Current [Vercel Go documentation](https://vercel.com/docs/functions/runtimes/go)
describes both modes, Go/toolchain selection from root go.mod, and Beta status.
Pin exact module revisions and a supported toolchain after a real build probe.
Do not infer Go support from Node.js/Fluid examples.

Framework's public `App.Handler()` permits composition. Disable static assets,
HTML redirects, locale behavior, browser authentication flows, and background
workers unless needed by the API. Test middleware response formats and caps.
No shutdown hook or work continuing after the response may be needed for correctness.

Warm instances may reuse immutable configuration and concurrency-safe HTTP
transports. They must not cache mutable protocol artifacts or authoritative
per-caller state. Do not assume requests reach the same instance.

## RAM ownership and replay

Freeze and retain raw answers in request RAM before interpretation. Return the
full replay material before a successful response ends. Bundle contents include
original evidence needed by bindings, pinned profiles/policy/questions,
DecisionSet, verifier identity, findings, and hashes. Avoid duplicating payloads.

Replay supplies the bundle in a new request; it neither retrieves evidence nor
calls a model. A digest detects tampering relative to an expected value but
does not authenticate who issued a bundle. Replay reports reproducibility;
it does not grant current authority or prove historical server issuance.

There is no GET-by-run-id history, durable 202 job, resumable upload, global
deduplication, or exactly-once guarantee. Lost responses cannot be recovered
from the server. A new evaluation may call a provider again and incur cost.
Client disconnects cancel work where possible but cannot recall an accepted
provider request.

## Memory-only interpretation

LeX application data uses RAM, with no writes to /tmp or mounted storage.
Build artifacts and read-only embedded schemas are allowed. Platform access
logs and provider-side retention are outside application RAM ownership;
document their actual settings and never put evidence, credentials, or raw
answers into telemetry. Do not claim zero retention by Vercel or providers.

## Bounded execution

Apply [SLO budgets](slo.md) to decoded input, provider responses, bundle size,
concurrency, and time. Validate the maximum possible response budget before
calling a provider; reject excessive upstream output. Never truncate evidence
or omit required raw answers to produce a success-shaped response.

## Capability checklist

Updated: 2026-09-22. `[x]` is built and covered by tests in this repository;
`[ ]` is open; `[-]` is deliberately excluded from this slice so the protocol
stays thin. Proof status per criterion is in
[conformance-report.md](conformance-report.md); the open proof list is in
[progress.md](progress.md).

### Protocol core

- [x] Normative text in `.project/.lex/` with RFC 2119 obligations, each mapped to a test (`internal/conformance/obligations_test.go`)
- [x] Ten invariants: evidence, judgment, authority, execution, and verified success kept distinct
- [x] Six verdicts with total precedence `error > conflict > insufficient > manual_review > rejected > validated`; all findings retained
- [x] Stage-classified failures `retrieval | pack | question | decision | policy | verification`; policy denial is a finding, not `policy_error`
- [x] Legacy wire envelope `0.1`; legacy verifier `0.2.0`; unsupported legacy versions and the `0.1` profile refused
- [x] Generic envelope `0.2` decision bundle: DecisionIdentity, State hash, caller QuestionSet, raw DecisionSet, optional Context binding, structural report, and self-hash
- [x] Envelopes frozen for canary (`0.2` generic, `0.1` legacy) with the compatibility rule in [governance.md](../.lex/governance.md): additive within a version, breaking changes get a new version
- [-] Universal ontology, automatic question generation, or a question DSL

### Evidence plane

- [x] Embedded Context `v0.1.0`, capability `memory-exact-v1`, one runtime per evaluation, 256 KiB input, 128 sources
- [x] Input `sources`: caller text frozen through Context under the profile focus
- [x] Input `frozen_context`: caller-frozen pack, snapshot, and pack request, accepted only after Context reproduces them
- [x] Evidence controls owned by LeX: project and runtime binding, profile focus with no caller controls, provenance, byte checksum, full-source surface, admissibility
- [x] Selection, budgeting, and rejection owned by Context; LeX compares the rebuild, never recomputes
- [x] Instruction and policy items refused as evidence; inference-only packs are `insufficient`
- [x] Empty exact selection is `insufficient` with a sealed bundle and no provider call
- [-] Context HTTP client on the evaluation path
- [-] Second retrieval engine, dense or morphology retrieval, or fuzzy matching inside LeX
- [-] Fetching evidence by URL

### Profiles and policy

- [x] Legacy profile as an immutable compatibility object: question set, policy document and gate, entity kind, Context focus, canonical hashes
- [x] Legacy registry composed by the deployment; legacy evaluation uses the first profile, legacy replay resolves the profile a bundle names
- [x] `claim-validation` `0.2.0`: five independent Noul predicates and one action Choice with `manual_review` and `other`
- [x] Thresholds external to the model and disclosed as `uncalibrated`
- [x] Calibration procedure and one recorded trial that did not change thresholds
- [ ] A calibration report per adapter path that supports a threshold decision
- [-] A second core profile registry; generic use cases define per-request QuestionSets instead
- [-] Caller-supplied policy that can lower a gate
- [-] Score inside `claim-validation` (it stays a conforming adapter type)

### Decision plane

- [x] Generic `/v1/decisions`: caller-owned State + QuestionSet, all Noul/Choice/Score primitives, no semantic Verdict
- [x] Generic structural verifier and provider-free replay, including state/question/answer/pin bindings
- [x] Optional generic frozen Context binding is verified but never merged into State
- [x] Provider-neutral `DecisionSet`: raw typed answers, adapter id and version, resolved model, request id, timing, usage
- [x] Adapter contract `0.1.0` with common conformance fixtures
- [x] Direct System One adapter live (`direct-systemone`, `jev-1.13.0`)
- [x] Hosted OpenRouter adapter passing local fixtures
- [x] Retryable provider failure reported as `provider_unavailable`, not retried; no silent model or provider fallback
- [x] Credential never retained in a response, including a Unicode-escaped copy
- [ ] Hosted adapter proven on the deployment
- [-] Provider orchestration, fan-out across providers, or majority voting
- [-] Provider names inside protocol object names or verdict semantics

### Verification and replay

- [x] Generic deterministic structural verifier: answer domains, state/question/context bindings, model pins, and replay self-hash
- [x] Legacy deterministic verifier: profile policy, provenance, thresholds, action consistency, and Verdict
- [x] Typed replay bundle mirrored from `replay-bundle.schema.json`; drift fails a test
- [x] JCS (RFC 8785) and SHA-256 with independent Python vectors agreeing with Go
- [x] Self-hash seal; tampering detected on replay
- [x] Replay without retrieval or provider; Context rebuild from the bundle's own snapshot; fresh-instance safe
- [x] Replay refused for a bundle outside the principal's projects
- [ ] Race evidence on a gcc-capable runner
- [-] Bundle signatures, issuer authentication, or OIDC
- [-] Server-side history, GET by run id, or durable jobs

### HTTP surface

- [x] `GET /healthz`, `GET /v1/capabilities`, `POST /v1/decisions`, deprecated `POST /v1/evaluations`, `POST /v1/replays`
- [x] JSON Schema 2020-12 for requests, responses, and bundles; OpenAPI 3.1.1 `0.2` validated in tests
- [x] RFC 9457 problem responses with stage and reason; HTTP status agrees with problem status
- [x] Method, Accept specificity, media type, body cap, response budget, `no-store`, 499/502/503/504 mappings
- [x] Capabilities disclose runtime, retrieval mode, evidence inputs, policy calibration, limits, and retention
- [x] Lifecycle trace rendered through the state machine; illegal sequences cannot be emitted
- [x] OpenAPI promoted to the canary contract `0.2`
- [x] External client guide ([client-integration.md](../.lex/client-integration.md)) and portable agent skill (`.cursor/skills/lex-api`)
- [-] TypeScript SDK, binary transports, gRPC, or asynchronous messaging
- [-] Durable 202 jobs, resumable upload, or exactly-once guarantees

### Deployment and operations

- [x] Go handler on Vercel, `PORT`, RAM-only, no `/tmp` or mounted storage writes
- [x] Authenticated principal-to-project binding; secrets outside the repository and traces
- [x] `govulncheck` run once with no vulnerabilities in called code
- [x] Local latency sample published as a measurement
- [x] Canary on production `beta`, revision-pinned in [conformance-report.md](conformance-report.md) with a named rollback deployment
- [ ] Release conformance report that closes the deferred exclusions, plus recorded toolchain, region, and cold start (ADR-0001, ADR-0002)
- [ ] SLO measurements with the 28-day window, published as measurements
- [ ] Vulnerability review named by the release checklist, repeated per release
- [-] Database, durable queue, distributed workers, or multi-tenant billing

### Outside this slice by design

- [-] Executor, authorized mutation, operation receipt, and `execution_error`
- [-] Any side effect triggered by a verdict
- [-] Product, chat, or agent shell on top of the protocol

### Readiness statement

LeX is at **canary**. The protocol core, generic decision API, evidence
plane, verifier, replay, and HTTP surface are built, tested locally, and
proven on a revision-pinned production deployment through the direct adapter;
the canary revision and rollback point are recorded in
[conformance-report.md](conformance-report.md). External applications can
call it through [client-integration.md](../.lex/client-integration.md).

The items still open are stable-release proof, not canary function: a
conformance report that closes every deferred exclusion, hosted-adapter proof,
race evidence, SLO measurements, a vulnerability review per release, a
calibration report, and use by several independent products.
