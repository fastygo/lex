# Generic typed-decision API

Status: canonical API working draft for the additive `0.2-draft` wire
operation. This document defines `POST /v1/decisions`; it does not replace the
deprecated claim-validation compatibility operation during its support window.

## Purpose and ownership

The request is:

```text
project_id + DecisionIdentity + State + QuestionSet + optional Context binding
  -> raw typed DecisionSet + structural report + trace + replay bundle
```

The caller owns State, question semantics, domain thresholds, interpretation,
sequencing, and any next action. LeX owns request validation, project
authorization, canonical hashes, adapter/model pins, typed-answer structural
validation, optional Context binding verification, trace, and replay. Context
Runtime owns any retrieval and ContextPack construction before the call. Jev
supplies typed answers.

State is never implicitly populated from Context. Metadata is bounded,
non-normative, excluded from the adapter input, and excluded from replay
semantics.

## Request

`POST /v1/decisions` accepts JSON with:

- `project_id`: authenticated project scope;
- `decision`: caller id and version for this decision step;
- `state`: arbitrary JSON object passed semantically unchanged to the adapter;
- `question_set`: caller id, version, and one through 64 typed questions;
- optional `context`: Context `pack`, `snapshot`, and `pack_request`;
- optional `metadata`: up to 16 short string values.

Question ids and Choice option keys are stable machine identifiers. Questions
are one of:

- `noul`: `type` and `instructions`;
- `choice`: `type`, `instructions`, and two through 255 named `options`;
- `score`: `type`, `instructions`, and two through ten unique ordered `levels`.

LeX converts `options` and `levels` to the adapter-neutral criteria shape. It
does not alter State meaning or question instructions. A raw Jev `criteria`
field is not part of this request.

A body that is not JSON is HTTP 400 `invalid_json`. Well-formed JSON that
violates the request schema, or a QuestionSet that fails semantic checks, is
HTTP 422 `question_error`. The problem `detail` names the JSON pointer and
schema keyword of the most specific failure, never caller values. Neither
case calls a provider.

## Response

A valid response is HTTP 200 with `structural_status: "valid"`, raw
`decision_set`, hashes, adapter metadata when the provider supplied it, trace,
and caller-owned `replay_bundle`. It deliberately has no `verdict`,
`policy`, or action authorization.

If raw provider answers fail deterministic structural checks, LeX returns HTTP
422 `decision_error` with `structural_status: "invalid"`, findings, trace, and
the sealed bundle. A provider transport failure is not a DecisionSet and keeps
the existing 502/503 mapping.

## Optional Context binding

Context is optional. When supplied, LeX verifies its shape, project and runtime
identity, pack hash, and Context rebuild before calling the provider. The pack
is a replay binding only: it is neither retrieval input owned by LeX nor
automatically inserted into State.

## Replay

`POST /v1/replays` accepts a generic bundle identified by
`bundle_kind: "typed_decision"`. Replay checks self-hash, decision/state/
question bindings, adapter/model pins, answer domains, project authorization,
and optional Context reproducibility. It returns
`replay_status: "decision_reproduced"` and the structural report. It never
calls a provider, retrieval service, or executor.

## Compatibility

`POST /v1/evaluations` is deprecated compatibility behavior. It retains the
fixed `claim-validation` profile, source-freezing convenience, semantic policy
gate, Verdict, and legacy replay-bundle format. Consumers should adopt
`/v1/decisions` for new orchestration work. The removal version is not yet
selected; a compatibility notice will precede removal.

## Examples

An intent disambiguation Choice and database-selection Score are test fixtures,
not embedded profiles. See `internal/httpapi/decision_test.go` and
`internal/wire/decision_test.go`. Their product vocabulary never enters LeX
core code.

## Migration from a direct Jev call

A direct Jev client commonly sends a JSON `state` plus a map whose question
entries use `type`, `instructions`, and a provider `criteria` domain. To call
LeX, keep the state as `state`, supply a stable `project_id` and `decision`,
and wrap the map in a versioned `question_set`.

For a direct Choice `criteria` object, copy the same key-to-description entries
to `options`. For a direct Score `criteria` array, copy the ordered entries to
`levels`. A Noul keeps only `type: "noul"` and `instructions`. Do not send
provider `model`, credentials, endpoint fields, transport retries, or raw
answers: deployment configuration owns the adapter pin and LeX obtains the
answers. Move non-semantic client correlation values to `metadata`; they are
not sent to Jev or sealed into replay semantics.
