# Protocol progress

Updated: 2026-09-22. This file is the visible checklist for the validation and
replay slice. Checked items have evidence in the repo or on the running direct
adapter. Unchecked items are still open.

## Running slice

- [x] Go REST service, Framework `v0.3.0`, Context `v0.1.0`, capability `memory-exact-v1`
- [x] Vercel deployment of synchronous evaluation and caller-owned replay
- [x] Routes `GET /healthz`, `GET /v1/capabilities`, `POST /v1/evaluations`, `POST /v1/replays`
- [x] Embedded profile `claim-validation` `0.2.0` with `support`, `established`, `refuted`, `conflict`, `safe_to_auto_act`, and one action Choice
- [x] Verdict precedence `error > conflict > insufficient > manual_review > rejected > validated`
- [x] Policy disclosure `uncalibrated`; wire envelope `0.1-draft`; profile `0.1` bundles rejected
- [x] Direct adapter live as `direct-systemone` / `jev-1.13.0`
- [x] Empty exact selection returns `insufficient` without a provider call
- [x] Technical `error` is HTTP 422 and still returns the replay bundle
- [x] Replay reproduces the verdict without retrieval or a provider
- [x] Schemas, canonical hashes, draft OpenAPI, and local adversarial tests
- [x] Research scenarios exercised through `POST /v1/evaluations`

## Closed in this pass

- [x] Plan and ADR status text match the running slice (`analysis.md`, `delivery.md`, plan `README.md`, ADR register)
- [x] ADR-0004 accepted: JSON Schema 2020-12 fixtures and OpenAPI checks in `internal/wire` and `internal/httpapi`
- [x] ADR-0005 accepted: `scripts/jcs_vectors.py` and `scripts/verdict_gate.py` agree with the Go verifier on the golden hashes and verdicts (`internal/wire/golden_test.go`, `internal/canonical/vectors_test.go`)
- [x] ADR-0006 accepted: method, Accept, no-store, and verdict/stage status mapping in `internal/httpapi/handler_test.go`
- [x] ADR-0007 accepted: network-free replay, no relative file writes, project isolation, and fresh-instance requests
- [x] ADR-0008 accepted: verdict truth table and deadline/disconnect coverage at pack, decide, verify, and replay
- [x] `safe_to_auto_act` label gloss matches the embedded question (accept the claim without a person reading it first)
- [x] Calibration decision recorded: trial `20260922-0947` does not select new thresholds; `claim-validation` `0.2.0` stays `uncalibrated`

## Architecture pass (2026-09-22)

- [x] Profile as object: `internal/profile` pins question set, policy document and gate, entity kind, and Context focus, and hashes them; `internal/profile/claimvalidation` holds `0.2.0`; a `Registry` composes profiles and the verifier resolves the one a bundle names (`internal/profile`, `internal/wire/verifier.go`)
- [x] Everything Context can do is delegated to Context: no exact-phrase, budget, rejection, chunk-id, or pack/snapshot-id recomputation in LeX; the verifier has Context rebuild the frozen state and compares by identity and canonical hash (`internal/evidence/frozen.go`, `internal/wire/evidence.go`); withdrawn finding codes `chunk_identity`, `pack_envelope`, `pack_request_identity`, `query_mismatch`, `rejection_mismatch`
- [x] Two evidence inputs behind one frozen state: `sources` (LeX freezes through Context) and `frozen_context` (caller-frozen state that Context must reproduce), disclosed in `GET /v1/capabilities` `context.inputs`, one-of enforced by the published schema (`internal/evidence`, `internal/httpapi/request.go`, `internal/httpapi/frozen_context_test.go`, request example `context-account-access-frozen.json`)
- [x] `wire/bundle.go` and `httpapi/evaluation.go` split by responsibility: `document.go`, `verifier.go`, `evidence.go`, `answers.go`; `request.go`, `evaluation.go`, `response.go`, `trace.go`; stage traces rendered through the lifecycle machine
- [x] Typed replay bundle: `wire.Bundle` decoded after schema and self-hash validation; `TestBundleTypeMirrorsSchema` fails on drift from `replay-bundle.schema.json`
- [x] Docs aligned: `checks.md`, `integration-stack.md`, `scope.md`, `architecture.md`, ADR-0003, ADR-0004, `conformance-report.md`, `README.md`, `AGENTS.md`

## Deferred

- [ ] Race evidence on a runner with gcc
- [ ] SLO measurements, including the 28-day window, published as measurements
- [ ] Conformance report pinned to a deployment revision, with exclusions and rollback to the previous immutable build
- [ ] Vulnerability review named by the release checklist

## Still open

- [ ] Hosted adapter proof on the deployment. Local direct and hosted fixtures pass (`internal/adapters/typesafe/conformance_test.go`, `internal/adapters/openrouter/conformance_test.go`). Live evaluations resolve `direct-systemone`. `LEX_OPENROUTER_API_KEY` and `LEX_HOSTED_RESOLVED_MODEL` are not in the local environment, so this deployment proof is not run.
- [ ] ADR-0001 acceptance: deployed toolchain identity is not recorded
- [ ] ADR-0002 acceptance: region, duration, bundle size, and cold start are not recorded
- [ ] ADR-0003 acceptance: blocked by the deferred race evidence
- [ ] ADR-0009 acceptance: local adapter fixtures pass; deployment proof and a per-path calibration report are still open
- [ ] ADR-0010 acceptance: blocked by the deferred vulnerability review
- [ ] ADR-0011 acceptance: blocked by the deferred SLO measurements
- [ ] ADR-0012 acceptance: blocked by the deferred revision-pinned conformance report

## Outside this slice

- [ ] Executor, operation receipt, and `execution_error`
- [ ] Score inside `claim-validation` `0.2.0`
- [ ] TypeScript SDK, database, durable queue, or server-side history
