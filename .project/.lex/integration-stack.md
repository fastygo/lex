# Integration stack

Status: normative integration contract, canary release; examples and
provider access-path notes are informative. Shared semantics belong to [protocol.md](protocol.md).

## Roles

```text
Context Runtime         optional evidence plane
Typed-decision adapter  judgment plane
LeX                     transport binding + structural verification + replay
Consumer / agent        question meaning, interpretation, and next action
Executor                explicitly authorized operation only
Trace / Receipt         audit + replay / execution postconditions
```

Jev is the reference decision model. Direct TypeSafe System One and OpenRouter
are provider access paths, not protocol identities. Endpoint availability and
capabilities must be checked by the adapter before execution.

## Generic decision boundary

The canonical operation is caller-owned `State + QuestionSet -> DecisionSet`.
The caller chooses the state, defines all Noul, Choice, and Score questions,
and interprets the answers. LeX preserves that state semantically, hashes the
input, pins the adapter and resolved model, validates answer shape and domains,
and returns a caller-owned replay bundle.

LeX does not generate questions, route between profiles, infer product
meaning, choose a threshold, retrieve evidence, or invoke a later tool. An
agent can use the returned answers to clarify a request, construct another
state, call Context Runtime, perform a web or MCP lookup, or stop.

`/v1/evaluations` is a claim-validation compatibility operation. It owns the
current fixed profile and semantic Verdict; it is not the generic core API.

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

## Optional Context binding

A generic decision may carry a frozen Context state as an optional binding. The
agent obtains it from Context before calling LeX. LeX validates project/runtime
identity and asks Context to reproduce the frozen state; it never merges pack
content into caller State or recomputes selection.

The compatibility evaluation operation additionally supports `sources` and
`frozen_context`: it freezes caller text through its fixed Context focus or
accepts a state obtained from Context. Those convenience inputs are not part of
the generic decision State contract.

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

1. Accept the exact State and QuestionSet, plus an optional frozen Context
   binding when the caller supplied one.
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

Deployment supplies exact adapter-version/model bindings to the verifier.
The verifier freezes that map and performs equality checks; it does not infer
immutable model identity from a suffix. Provider-specific pin configuration
belongs to the HTTP composition and adapters. The direct default is pinned;
the hosted path requires an explicitly configured, provider-confirmed identity.
An echoed selector or a different resolved identity fails without fallback.

## Caller-owned question design

Use Noul for one independent condition, Choice for mutually exclusive
categories, and Score for a genuinely ordered rubric. A Choice has exact
unique option keys; a Score has two through ten ordered levels. Callers should
include an `other` or clarification option when their taxonomy is incomplete.
LeX validates these contracts but does not decide whether an option set is
semantically adequate for the caller's product.

Illustrative interpretation:

```text
retain raw answers in request RAM and the response bundle
validate answer types, domains, pins, and optional Context binding
return structural findings and DecisionSet to the caller
let the caller apply domain threshold, policy, authority, and next-step logic
```

The claim-validation compatibility adapter separately derives support,
conflict, safety, and `manual_review`. A high model probability never grants
authority in either API.

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
