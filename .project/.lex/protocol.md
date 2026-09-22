# LeX protocol

Status: normative, canary release, envelope `0.2`. Requirement keywords and
compatibility promises follow [governance.md](governance.md). Checks, finding
codes, and HTTP mapping are in [checks.md](checks.md).

## Operation

LeX has one operation and its replay:

```text
project_id + DecisionIdentity + State + QuestionSet + optional frozen Context
  -> pinned typed-decision adapter
  -> raw DecisionSet
  -> deterministic structural verification
  -> structural report + trace + sealed replay bundle
```

The caller owns State, question meaning, thresholds, interpretation,
sequencing, and every next action. LeX owns request validation, project
authorization, canonical hashes, adapter and model pins, structural answer
checks, optional Context binding verification, trace, and replay. Context
Runtime owns retrieval and ContextPack construction before the call. The
typed-decision model supplies answers.

## Invariants

1. State and QuestionSet MUST reach the adapter semantically unchanged; LeX does not rewrite, enrich, or translate them.
2. Questions MUST be typed and atomic: Noul, Choice, or Score with a declared answer domain.
3. LeX MUST NOT derive a domain verdict, apply a threshold, or choose a next action.
4. Evidence, judgment, authority, execution, and verified success MUST remain distinct; a decision grants no authority and performs no action.
5. A supplied Context binding MUST be reproduced by Context before the adapter is called.
6. Context MUST NOT be merged into State; the pack is a replay binding, not adapter input rewritten by LeX.
7. Runs MUST pin project, DecisionIdentity, State, QuestionSet, optional Context binding, adapter, resolved model, and verifier versions.
8. A criteria change MUST produce a new QuestionSet hash; callers also bump the QuestionSet version.
9. A model alias such as `latest` MUST NOT substitute for a resolved model identifier.
10. Transport normalization MUST NOT rewrite the semantic answers.
11. Trace data MUST exclude credentials and provider secrets.
12. Request `metadata` MUST NOT reach the adapter or the replay bundle.

## Entities

**DecisionIdentity**: caller-owned `id` and `version` of one decision step.

**State**: a caller-owned JSON object. LeX validates its transport form and
hashes it. It is not evidence by default and is never populated from Context.

**QuestionSet**: caller-owned `id`, `version`, and one through 64 questions
keyed by stable identifiers:

- `noul`: `instructions`; the answer is the probability of yes.
- `choice`: `instructions` and two through 255 named `options`; the answer is
  one option and a probability per option.
- `score`: `instructions` and two through ten unique ordered `levels`; the
  answer is a position in `[0, len(levels) - 1]`.

LeX converts `options` and `levels` to the adapter-neutral criteria shape. A
raw provider `criteria` field is not part of the request. LeX validates
structure; it does not judge whether a taxonomy suits a product.

**Context binding**: an optional frozen Context `pack`, `snapshot`, and
`pack_request` from the pinned runtime, bound by `pack_hash`.

**DecisionSet**: raw typed answers with `adapter_id`, `adapter_version`, and
`resolved_model`. Provider request id, timing, and usage are returned as
`adapter_metadata` when the provider supplies them, never fabricated, and are
not sealed.

**Structural report**: `structural_status` (`valid` or `invalid`) and
`findings`. A finding is a code and a detail. It carries no domain meaning.

**Replay bundle**: `bundle_kind: typed_decision`, `protocol_version`,
`verifier_version`, `project_id`, `decision`, `state`, `question_set`,
optional `context`, `decision_set`, and `bundle_hash`, the canonical hash of
everything else.

## Lifecycle

```text
decision: receive -> decide -> verify
replay:   receive -> verify
```

Context binding verification runs inside `decide`, before the adapter call.
A request-scoped state machine emits the trace; illegal sequences cannot be
rendered. Request refusals happen before a stage trace exists. A failed stage
ends the trace at that stage. A transport failure is not a DecisionSet.

A structurally invalid answer MUST be reported as findings together with its sealed bundle.

## Adapters

An adapter MUST: accept exact State and QuestionSet; declare supported primitives before a call; preserve raw typed answers; bind the resolved model; record provider metadata when available; distinguish retryable transport failure from decision output; never fall back silently to another model or provider; and pass the common conformance fixtures.

Endpoints, credentials, SDK types, aliases, and billing stay inside the
adapter. Provider additions stay metadata. An adapter that cannot preserve
Noul, Choice, and Score semantics is not a conforming typed-decision adapter.

## Replay

Replay takes a bundle the caller retained and rechecks it under the pinned
verifier. It reproduces structural findings, not a fresh model judgment.

- Replay MUST NOT contact a retrieval service, call a decision provider, or execute side effects.
- A Context binding is rebuilt by Context from its own snapshot and pack request; the saved pack MUST NOT be replaced.
- Replay MUST refuse a bundle whose project is outside the authenticated principal's projects.

A fresh model call is a new decision even with the same resolved model. Trace
timestamps and request ids are not sealed; byte-identical traces are not
promised. Replay proves the recorded decision, not current authority.

## Retention

The service keeps nothing after the response. The caller owns the replay
bundle. There is no server history, run lookup, durable job, or idempotency
store; a lost response is not recoverable.

## Deferred

Controlled execution (authorization, operation contract, postcondition check,
receipt) is outside this release. No response authorizes or reports an
operation.
