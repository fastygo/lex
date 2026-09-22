# LeX

LeX is subject-neutral operational infrastructure for turning evidence and
uncertain judgments into governed, inspectable, reproducible outcomes.

Its central rule is:

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

LeX combines:

- **Context Runtime** as the evidence plane;
- provider-neutral typed decisions as the judgment plane;
- deterministic policy and verification gates;
- verdicts, traces, replay bundles, and conformance evidence.

Jev is the current reference typed-decision model. Provider APIs are adapters,
not protocol identities.

## Status

LeX is a working draft. The service implements an additive generic typed-
decision API, legacy synchronous claim validation, and caller-owned replay. No
released protocol, conformance certification, achieved SLO, or production
deployment is claimed.

The running profile is:

- Go REST service using Framework and
  [`github.com/fastygo/context@v0.1.0`](.project/.plan/context-version.md);
- embedded Context capability `memory-exact-v1`;
- deployment target: Vercel Go runtime;
- request-scoped RAM only for mutable protocol data;
- synchronous validation and deterministic replay;
- caller-owned replay bundles;
- no database, disk persistence, durable queue, server-side run history, or
  side-effect execution;
- no TypeScript SDK or JavaScript application runtime in v0.1.

Context v0.1.0 and Framework v0.3.0 are pinned. The generic `0.2-draft`
operation accepts caller-owned JSON state and a caller-owned QuestionSet,
optionally binds a frozen Context state, calls a pinned typed-decision adapter,
validates raw answer structure, and returns a replayable DecisionSet. The
agent owns semantic interpretation and subsequent tools or actions.

The compatibility evaluation operation freezes a Context
pack with embedded `memory-exact-v1` from caller text, or accepts a state the
caller already froze through that runtime and has Context reproduce it. It then
asks the embedded `claim-validation` `0.2.0` question set through a direct or
hosted System One adapter, applies the uncalibrated `0.2.0` policy, and returns
a verdict plus a caller-owned replay bundle. Replay reproduces that verdict
without contacting a retrieval service or a provider. It has Context rebuild
the pack from the frozen snapshot already in the bundle and does not replace
the saved pack; LeX itself does not recompute selection. The wire envelope
remains `0.1-draft`. Bundles pinned to the `0.1` profile are rejected by this
verifier.

A draft OpenAPI document and local adversarial tests exist. The open proof
list is [progress.md](.project/.plan/progress.md): hosted-adapter proof on a
deployment, ADR acceptance that is still proposed, and the deferred race, SLO,
revision-pinned conformance report, and vulnerability review. Calibration of
`0.2.0` stays `uncalibrated`. No conformance certification is claimed.

## Architecture

```text
Agent: State + QuestionSet + optional Context binding
  -> LeX: schema + hash + model pin + structural answer verification
  -> Jev: raw typed answers
  -> LeX: DecisionSet + trace + replay bundle
  -> Agent: interpret, clarify, retrieve, call MCP/web/tool, or stop
```

```text
EntityEnvelope + ValidationIntent
  -> frozen PolicySnapshot + profiles + QuestionSet
  -> Context Runtime -> frozen ContextPack + EvidenceBinding
  -> preflight checks
  -> typed-decision adapter -> raw DecisionSet
  -> deterministic verifier
  -> Verdict + VerificationReport + EvaluationTrace + replay bundle
```

Replay consumes a caller-retained frozen bundle. It does not contact a retrieval
service or invoke a decision provider. Context's rebuild from the frozen
snapshot checks the saved pack and does not replace it.

Profiles are legacy compatibility objects: each pins a question set, a policy
document and gate, an entity kind, and a Context focus. The deployment composes
a registry only for claim evaluation and its legacy replay bundles. Generic
decisions have no profile registry; they use a caller-defined QuestionSet.

Controlled execution is deliberately outside the first slice. A validation
verdict does not prove that an operation occurred.

## Protocol model

Core protocol objects:

- `EntityEnvelope` — the versioned object under validation;
- `ValidationIntent` — the question being resolved;
- `ContextPack` and `EvidenceBinding` — frozen admissible evidence;
- `SemanticProfile` and `QuestionSet` — versioned meanings and typed questions;
- `DecisionSet` — unmodified provider output with resolved identity;
- `PolicySnapshot` — frozen thresholds, risk, and authority rules;
- `VerificationReport` — deterministic contract checks;
- `Verdict` — final validation disposition;
- `EvaluationTrace` — replayable record;
- `Receipt` — proof of a separately authorized and verified operation.

LeX preserves these verdicts:

```text
validated | rejected | insufficient | conflict | manual_review | error
```

`insufficient` and `conflict` are epistemic outcomes.
`manual_review` is an operational disposition.

## Typed question design

The reference decision contract uses:

- independent **Noul** questions for support, establishment, refutation,
  conflict, and safety predicates;
- one **Choice** for a mutually exclusive operational recommendation;
- **Score** only for a genuinely ordered rubric.

The embedded `0.2.0` profile asks `support`, `established`, `refuted`,
`conflict`, `safe_to_auto_act`, and one action Choice. `refuted` means the
evidence establishes that the claim is false. Coherent refutation is not
conflict. Score is a conforming adapter type and is not asked by this profile.

A decision is never evidence for its own verdict, and confidence never grants
authority.

## HTTP profile

The handler serves:

```text
POST /v1/decisions
POST /v1/evaluations
POST /v1/replays
GET  /v1/capabilities
GET  /healthz
```

`POST /v1/decisions` is the canonical generic operation. It accepts State,
QuestionSet, and an optional frozen Context binding and returns no semantic
verdict. `POST /v1/evaluations` is deprecated claim-validation compatibility
behavior.

The wire profile uses JSON Schema 2020-12, RFC 8785 JCS with SHA-256 for LeX
canonical hashes, and RFC 9457 Problem Details. A draft OpenAPI 3.1.1 document
is at `internal/wire/schema/openapi.json`. `GET /v1/capabilities` reports
generic primitives and limits alongside legacy Context and policy capability.
Research maps under `.project/.jev/examples/` are not legacy evaluation
requests; agents translate their question/state material into generic decision
requests. These routes are the working draft surface. They
are not a released protocol while the remaining ADRs are proposed.

## Delivery path

The local service implements the evaluation and replay path: an embedded
Context pack, direct and hosted adapter fixtures, schemas, hashes, the
verifier, and network-free replay. What remains open is the checklist in
[progress.md](.project/.plan/progress.md). The delivery sequence still records
those proof gates in [delivery and proof](.project/.plan/delivery.md).

## Documentation

Start here:

1. [Canonical specification](.project/.lex/README.md)
2. [Concept and boundaries](.project/.lex/concept.md)
3. [Protocol entities and lifecycle](.project/.lex/protocol.md)
4. [Checks, verdicts, and errors](.project/.lex/checks.md)
5. [Implementation plan](.project/.plan/README.md)
6. [Architecture and deployment profile](.project/.plan/architecture.md)
7. [ADR register](.project/.plan/adr/README.md)
8. [Conformance gates](.project/.plan/conformance.md)
9. [SLOs and resource budgets](.project/.plan/slo.md)

Additional material:

- [VSA and ICOM guidance](.project/.vsa/README.md)
- [Jev research](.project/.jev/README.md) — non-normative captured evidence
- [Project documentation map](.project/README.md)

When documents disagree, `.project/.lex/` owns protocol semantics.
`.project/.plan/` owns implementation and deployment planning.

## Optional tooling

Human-readable content outside `.manual/` is English only. Optional:

```bash
npm run check:english
```

Current foundation checks (Python with the pinned conformance dependency is required):

```bash
python -m pip install -r scripts/requirements-conformance.txt
go test ./... -count=1
go vet ./...
```

The working-draft schemas, OpenAPI document, and local conformance tests are
enforced by `go test ./...`. They are not a conformance certification.

## Local API

The local handler exposes `GET /healthz`, authenticated `GET /v1/capabilities`,
`POST /v1/evaluations`, and `POST /v1/replays`. Evaluation accepts a project,
an entity of type `claim` with schema `0.1`, an exact-phrase query, and
versioned source texts. The caller does not send the question set. The server
assigns source trust and evidence class, freezes the pack, and applies the
embedded `claim-validation` `0.2.0` policy. That policy is explicitly
uncalibrated.

```json
{
  "project_id": "example-project",
  "entity": {
    "id": "claim-1",
    "type": "claim",
    "schema_version": "0.1",
    "version": "1"
  },
  "query": "account is locked",
  "sources": [
    {"id": "source-1", "version": "v1", "text": "The account is locked."}
  ]
}
```

The exact phrase in `query` must occur in a source text or the selection is
empty.

A caller that already holds a frozen state from the pinned Context runtime
sends it instead of `sources`. Exactly one of the two inputs is present.
`pack_request.query` must equal `query`, the pack request must carry the
profile focus with no caller controls, and Context must reproduce the snapshot
and pack before a provider is called; otherwise the response is HTTP 422
`pack_error` with the findings the verifier would emit. `GET /v1/capabilities`
lists the accepted inputs under `context.inputs`.

```json
{
  "project_id": "example-project",
  "entity": {"id": "claim-1", "type": "claim", "schema_version": "0.1", "version": "1"},
  "query": "account is locked",
  "context": {
    "pack": {"id": "pack_…", "evidence_items": ["…"], "…": "…"},
    "snapshot": {"id": "snapshot_…", "project_id": "example-project", "runtime_version": "memory-exact-v1", "sources": ["…"]},
    "pack_request": {"project_id": "example-project", "query": "account is locked", "focus": {"…": "…"}}
  }
}
```

The 19 scenario bodies derived from `.project/.jev/examples/`, including one
complete `frozen_context` body, are in
[`.project/.jev/test-vercel/requests/`](.project/.jev/test-vercel/requests/).
The JSON maps next to those research notes are Jev question and response
captures. Posting one of them to `/v1/evaluations` is rejected. Replay posts
the returned `replay_bundle` to `POST /v1/replays`.

An empty exact retrieval returns
`insufficient`, does not call a provider, and still returns a replay bundle.
A technical `error` verdict is HTTP 422 and still returns the sealed replay
bundle. Replay accepts that bundle only when its entity project is one of the
token's projects.

Configure bearer tokens outside version control. Set one decision credential,
or set `LEX_DECISION_ADAPTER` to `direct` or `hosted` when both are present:

```bash
export LEX_BEARER_TOKENS='{"development-token":["example-project"]}'
export LEX_TYPESAFE_API_KEY='replace-with-secret'
go run ./cmd/api
```

The embedded QuestionSet, policy, and verifier are version `0.2.0`.
The wire envelope remains `0.1-draft`. Bundles pinned to the `0.1` profile are
rejected by this verifier; retain the previous verifier for historical replay.

The hosted adapter additionally requires `LEX_HOSTED_RESOLVED_MODEL`: the exact
immutable identity confirmed by the provider for your deployment. The response
must match this pin. There is no inferred or default hosted resolved identity;
`latest`, `stable`, and `preview` are not accepted. Synthetic dates in tests are
fixtures, not verified provider releases. The same deployment pin governs replay.

`LEX_BEARER_TOKENS` and provider credentials are deployment secrets. Never add
them to source files, request payloads, traces, replay bundles, or logs.

## License

Licensed under the [Apache License 2.0](LICENSE).
