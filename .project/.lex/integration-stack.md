# Integration stack

Status: normative integration draft; examples and provider access-path notes
are informative. Shared semantics belong to [protocol.md](protocol.md).

## Roles

```text
Context Runtime      evidence plane
Typed-decision adapter  judgment plane
LeX verifier         policy binding + deterministic gates
Executor             explicitly authorized operation only
Trace / Receipt      audit + replay / execution postconditions
```

Jev is the reference decision model. Direct TypeSafe System One and OpenRouter
are provider access paths, not protocol identities. Endpoint availability and
capabilities must be checked by the adapter before execution.

## Context boundary

Integrate through Context HTTP API v1, public `contextkit`, or its embedded
`contextkit/runtime` package; do not import
Context internals or reopen its frozen API to add provider-specific types.
Context owns retrieval, ranking, FocusProfile, and ContextPack construction.
LeX checks the actual upstream contract and records its version.

Required handoff properties are project isolation, frozen evidence identity,
checksums and provenance, instruction/policy separation, accepted/rejected
material, and sufficient source references for verification. Preserve
conflicting evidence. Do not infer full LeX compatibility from a manual mock pack.

Context field definitions remain upstream; local EvidenceBinding maps those
fields without inventing an alternative ContextPack schema. See
[sources.md](sources.md) for the public API and evidence-contract references.

## Current deployment restriction

The [RAM-only plan](../.plan/README.md) pins [Context v0.1.0](../.plan/context-version.md)
and its separate public contextkit/runtime package, capability memory-exact-v1.
The parent contextkit.Client remains an HTTP client. The embedded path builds
real packs through exact phrase retrieval and the existing pack builder, without
an external service or persistence. Unsupported retrieval/profile features must
fail explicitly. LeX composition and deployed resource measurements remain
acceptance gates in [ADR-0003](../.plan/adr/0003-context-boundary.md).

## Decision-adapter obligations

An adapter MUST:

1. Accept the exact frozen pack and QuestionSet.
2. Declare supported typed primitives, limits, and metadata capabilities before a call.
3. Preserve raw typed answers without semantic rewriting.
4. Bind results to inputs, adapter version, and resolved model identity.
5. Record provider, request id, timing, and usage when available.
6. Distinguish retryable transport failures from decision output.
7. Never silently fall back to another model or provider.
8. Pass common conformance fixtures before claiming typed-decision compatibility.

Endpoints, credentials, SDK types, aliases, retry machinery, and billing fields
stay inside adapters. The lifecycle controller owns retry permission, budget,
and cancellation; adapter transport retries must respect these controls.
Provider additions remain trace metadata, never verdict rules.

An adapter that cannot preserve Noul/Choice/Score semantics is not a conforming
typed-decision adapter. Supporting one primitive does not imply all are available;
the requested QuestionSet must fit the declared capabilities.

## Question design

Use independent support Noul questions for each hypothesis; separate atomic
establishment, conflict, and safety questions; one action Choice for mutually
exclusive operational alternatives; optional Score for a genuinely ordered rubric.
Include `other` when the taxonomy is incomplete and an explicit non-action
path such as `manual_review` for risky action recommendations.

Illustrative interpretation:

```text
retain raw answers in request RAM and the response bundle
derive support, establishment, conflict, and safety signals
apply evidence, threshold, policy, and authority checks
preserve epistemic findings in VerificationReport
emit Verdict and any separately governed action disposition
```

A conflict can require human review operationally without being relabeled
`manual_review` epistemically. A high safety signal never grants authority.

## Evidence eligibility

Eligibility is controlled by PolicySnapshot and checked by the verifier.
Source text, attestations, and authoritative tool output can justify facts only
when their provenance, trust, and domain requirements pass. `model_inference`
cannot independently establish a fact. Lexical analysis and concept mappings
are aids to interpretation, not witnessed quotes or authority.

Shared spans are a diagnostic, not proof of semantic conflict. Deterministic
checks detect exact duplicates and structural violations; semantic overlap
requires reviewed definitions or an explicit semantic assessment.

## Proof boundary

[Research findings](../.jev/research-findings.md) concern hand-authored states.
Real Context API packs, calibration, and interoperability across two adapter
paths remain [proof gates](analysis.md). A provider accepting JSON does not
prove that it preserves the typed-decision contract.
