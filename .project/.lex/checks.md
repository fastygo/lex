# LeX checks

Status: normative, canary release, envelope `0.2`. Shared semantics are in
[protocol.md](protocol.md).

## Request checks

Before any adapter call, LeX checks the bearer token and its project binding,
method, `Accept`, `Content-Type`, body size, JSON well-formedness (duplicate
keys, invalid UTF-8, non-finite numbers, depth above 32), the request schema,
QuestionSet semantics, and the response budget. A failed check never reaches a
provider. Caller-supplied policy, approval, or cookie authority is refused.

## Structural verification

The verifier MUST check: bundle self-hash; decision, State, QuestionSet, and Context bindings; adapter, resolved-model, and verifier pins; answer completeness, types, and domains; Context reproduction; and project authorization at replay.

Finding codes are stable. Codes without a suffix:

| Code | Meaning |
|------|---------|
| `binding_mismatch` | a sealed State or Context hash, or an identifier, does not match its content |
| `decision_checksum_mismatch` | the DecisionIdentity checksum does not match project, id, and version |
| `invalid_answers` | the answer envelope is not a typed answer map |
| `pack_rebuild` | Context does not rebuild the saved pack |
| `pack_shape` | the Context binding is not Context's frozen form |
| `project_binding` | the Context binding names another project or runtime |
| `snapshot_identity` | Context does not reproduce the saved snapshot |
| `unknown_adapter` | the adapter is not configured on this deployment |
| `unpinned_adapter` | the adapter version differs from the deployment pin |
| `unpinned_question_set` | the QuestionSet hash is missing or differs |
| `unpinned_verifier` | the bundle kind, envelope, or verifier version is not this verifier |
| `unresolved_model` | the resolved model differs from the pin or is an alias |

Answer codes append `:` and the question id: `answer_type_mismatch`,
`invalid_choice`, `invalid_noul`, `invalid_score`, `missing_answer`,
`unexpected_answer`, `unsupported_question_type`.

A Choice selects one of its maximum-probability options; exact ties are
allowed. Reserved answer fields are case-sensitive: `NOUL` is
`invalid_answers`, not an alias. Invalid UTF-8 and unpaired surrogates are
rejected before decoding can repair them. New codes are additive.

LeX does not recompute Context selection, budgeting, or rejection. It decodes
the frozen state strictly into Context's public types, checks project and
runtime identity, asks the pinned runtime to rebuild, and compares identities
and canonical hashes.

## Problems and HTTP status

Problems are RFC 9457 JSON with a stable `reason`. The HTTP status equals the
problem `status`.

| Status | Reason | When |
|-------:|--------|------|
| 400 | `invalid_json`, `invalid_body`, `json_too_deep` | body is not usable JSON |
| 401 | `authentication_required` | no bearer token |
| 403 | `authentication_denied`, `project_forbidden` | wrong token, or token lacks the project |
| 404 | `not_found` | unknown route |
| 405 | `method_not_allowed` | wrong method; `Allow` is set |
| 406 | `not_acceptable` | `Accept` refuses `application/json` |
| 413 | `body_too_large` | body above the limit |
| 415 | `unsupported_media_type` | body is not `application/json` |
| 422 | `question_error` | schema or QuestionSet violation; `detail` names the JSON pointer and keyword, never caller values |
| 422 | `pack_error` | the Context binding does not reproduce; `findings` listed |
| 422 | `decision_error` | answers fail structural checks; sealed bundle returned |
| 422 | `response_budget` | the response would not fit the body limit |
| 422 | `invalid_replay_bundle` | the replay body fails the bundle schema or its self-hash |
| 499 | `client_canceled` | the client disconnected |
| 500 | `verification_error`, `internal_error` | LeX could not complete a deterministic step |
| 502 | `decision_error` | the adapter returned an unusable response |
| 503 | `admission_limited`, `provider_unavailable`, `decision_provider_unavailable` | overload, retryable provider failure, or no adapter configured |
| 504 | `deadline_exceeded` | the request deadline elapsed |

A retryable provider failure MUST NOT be retried by LeX; it is `provider_unavailable`, and each client retry is a new decision.

A response that does not fit the budget MUST be refused with `response_budget`, never truncated; the provider is not called when the inputs already fill the budget.

`Accept` is matched by specificity: an explicit refusal of `application/json`
outweighs `application/*` and `*/*`. Responses are `Cache-Control: no-store`.
A refused provider redirect is a decision failure and the target is not
returned. A provider response that echoes the credential, including a
Unicode-escaped copy, is dropped.

Keep the original stage in the trace. Do not report a pipeline failure as
"the model was wrong".
