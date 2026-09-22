# Delivery and proof sequence

Status: the validation and replay slice is running on the direct adapter.
[Context v0.1.0](context-version.md) satisfies the embedded-interface
prerequisite. Gates that still lack their acceptance evidence are unchecked
in [progress.md](progress.md). This file does not mark the release checklist
done.

## P0: dependency and deployment feasibility

Resolve ADR-0001 through ADR-0003. Build a Go-only Framework handler with pinned
Context v0.1.0 embedded dependency and deploy a preview to Vercel. Verify route mapping,
toolchain, middleware, request limits, cold starts, and absence of runtime
filesystem/database use. Exercise runtime.New and ContextPack through the actual
Framework handler. Upstream package tests establish the interface, not this
composition or deployment. Pin Framework separately.

Done when a real deployed probe and dependency report exist, and the supported
ContextPack acquisition path is declared. Upstream API changes require their own
measured blocker and ADR; this plan does not authorize edits to Context core.

## P1: wire contract and deterministic replay

Resolve ADR-0004 through ADR-0008. Define schemas and OpenAPI, canonical hash
vectors, all verdict precedence rules, stage/status mappings, and portable
bundles. Build the Go verifier and REST replay operation first.
Map every normative requirement to positive and negative vectors.

Done when independent producers/consumers agree on messages and hashes, every
verdict and invalid transition is covered, and replay runs with network disabled.
No TypeScript implementation is required: use a separate Go test consumer and
independently maintained standard vectors rather than one shared serializer.

## P2: real evidence and typed decisions

Build a real frozen pack with Context v0.1.0 memory-exact-v1 from versioned
source text under the reviewed focus, preserving raw source bindings.
Add direct and hosted decision adapters under common fixtures. Store raw
answers only in request RAM and the response bundle. Prove freeze-before-call,
resolved versions, no silent fallback, and bounded cancellation.
Verify ADR-0003's restricted capabilities, source admission, preserved upstream
hashes, complete source Snapshot export, and serialized bundle budgets.

Done when adversarial evidence and provider failures are classified correctly,
two adapter paths preserve typed semantics, and policy-specific calibration
results disclose limitations.

## P3: security and serverless reliability

Resolve ADR-0009 through ADR-0012. Exercise fresh-instance replay, concurrent
project isolation, retries, disconnects, oversized responses, memory pressure,
cold starts, and platform failures. Run the SLO benchmark and live probes.

Done when the conformance report links all mandatory checks and deployment
evidence to exact revisions. Publish performance measurements as measurements,
not an achieved long-window SLO.

## Release checklist

- Accepted ADRs and reviewed normative specification agree.
- Immutable schemas, OpenAPI, hash vectors, and verifier build are identified.
- Every mandatory conformance criterion passes; exclusions name unsupported roles.
- Vercel deployment manifest, secret handling, operational limits, and rollback
  to the prior immutable build are verified.
- No SDK, database, durable queue, or side-effect executor has entered scope.
- The response clearly discloses caller-owned replay retention and lack of history.
- Implementation checks include Go tests, race tests
  on a supported runner, vet, vulnerability review, and the conformance suite.

Acceptance of specification, runtime compatibility, security tests, calibration,
and SLO measurement are distinct evidence categories. One cannot replace another.
