# v0.1 implementation and release plan

Status: informative planning. No item below is evidence of completion.

## Baseline

The repository has a conceptual specification and Jev research on manual states.
All ten [protocol invariants](protocol.md#core-invariants) apply to the first
slice. Narrow the supported scenario, never the invariants.

## Current implementation profile

The [detailed plan](../.plan/README.md) owns ADRs, SLOs, conformance criteria,
and delivery gates. Current scope is Go REST on Vercel, using Framework and
Context public contracts, with request-local RAM and no database or TypeScript.
Raw answers and traces are exported in a complete response bundle; the caller
owns any retention. There is no durable server history or asynchronous job.
The selected evidence adapter is [Context v0.1.0](../.plan/context-version.md),
using public pkg/contextkit/runtime for in-process exact retrieval and packing.
Upstream capability is tested; LeX composition and Vercel deployment remain
pending under [ADR-0003](../.plan/adr/0003-context-boundary.md).

## First slice

One entity schema, one reviewed SemanticProfile, one fixed FocusProfile, one
versioned QuestionSet, and one explicit PolicySnapshot:

```text
ValidationIntent -> real ContextPack -> raw DecisionSet
  -> deterministic verifier -> Verdict + EvaluationTrace
```

Use independent support Noul questions, separate establishment/conflict/safety
questions, and one action Choice when an action recommendation is needed.
Score requires an ordered rubric. Execution is outside this first slice.

## Proof gates in dependency order

1. Build a real pack with Context v0.1.0's public embedded runtime from bounded,
   versioned source texts and reviewed focus controls. Retain the frozen pack,
   source Snapshot, exact PackRequest, and upstream identities in the replay
   bundle. Map fields to EvidenceBinding and verify the LeX composition; keep
   Context digests separate from LeX canonical hashes.
2. Define versioned JSON Schema 2020-12 artifacts, canonical hash inputs, and
   positive/negative hash vectors. Resolve object fields listed in protocol.md.
3. Preserve exact typed answers in request RAM and the response bundle with
   input bindings, adapter version, provider
   metadata, and resolved model identity. No silent fallback.
4. Implement the deterministic verifier, including evidence eligibility,
   conflicts, thresholds, authority, and all six verdicts.
5. Add golden and invalid fixtures linked to normative requirement identifiers.
6. Add adversarial cases: inference-only evidence, missing evidence, conflicting
   sources, compound questions, overlapping criteria, injection, and provider failures.
7. Calibrate by entity type, resolved model, provider path, and policy; report
   corpus limits and measured error rates.
8. Replay caller-supplied frozen inputs and DecisionSet without retrieval, model calls, or
   side effects; reproduce verifier results under a pinned verifier version.
9. Run common fixtures through at least two decision-provider adapters.
   Separately prove independent message producers and consumers agree on
   schemas, hashes, and verdict semantics.

## Specification blockers before release

- Exact wire fields, required/optional properties, references, numeric domains,
  limits, timestamps, and unknown-field/enum behavior.
- JSON canonicalization and SHA-256 profile: hash scope, self-hash exclusion,
  source-byte checksums, and cross-language vectors.
- Complete transitions, stage errors, timeouts, cancellation, duplicate
  requests, crash recovery, and idempotency scope and retention.
- Deterministic verdict precedence and reason codes for simultaneous conflict,
  missing evidence, policy denial, and human-review requirements.
- HTTP binding: media types, status codes, authentication, error bodies,
  version negotiation, and OpenAPI. HTTP is the intended first remote binding.
- Security considerations: untrusted evidence, isolation, credentials,
  authority binding, replay protection, retention, and trace redaction.
- Extension/version rules, conformance roles, testable requirements, and
  release evidence. Schemas and OpenAPI alone are insufficient.

Standards candidates are in [sources.md](sources.md). Listing a standard does
not implicitly adopt all of its requirements.

## Deliberate deferrals

Defer automatic question generation, universal ontologies, a new retrieval
engine, provider orchestration, binary transports, distributed execution, and
signature infrastructure, and TypeScript SDKs. Generate Go wire types from
chosen schemas; do not hand-maintain competing schema authorities.

Execution needs an OperationContract, external authority, idempotency and
recovery semantics, and verified postconditions. Unsupported capabilities must
be explicit; a successful validation verdict cannot simulate execution.
