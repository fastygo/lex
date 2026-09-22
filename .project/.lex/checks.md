# LeX checks

Status: normative check catalog draft. Shared semantics belong to
[protocol.md](protocol.md); release blockers are in [analysis.md](analysis.md).

## Generic structural checks

For `POST /v1/decisions`, LeX checks project authorization, bounded JSON State,
DecisionIdentity, QuestionSet shape and hash, primitive domains, adapter and
resolved-model pins, raw answer completeness/types/distributions, response
budget, optional frozen Context binding, bundle self-hash, and replay project
binding. A successful generic decision returns `structural_status: valid`; it
does not return a domain verdict.

The caller owns question meaning, semantic thresholds, policy findings, and
all next actions. A malformed QuestionSet is `question_error`; malformed or
incompatible typed answers are a structural decision error with a sealed
bundle when possible. Generic replay rechecks structure and returns
`decision_reproduced`; it does not contact Context retrieval or a provider.

## Compatibility structural and preflight checks

Before decision-provider execution, check:

- Entity/project/schema identity, versions, checksums, and immutable references.
- QuestionSet version/hash binding, required question identifiers, and types.
- Pack shape, instruction/evidence separation, evidence provenance, and budgets.
- Duplicate labels or definitions, declared ordered Score levels, and required
  `other` or non-action alternatives.
- Adapter capabilities, requested model restrictions, and available authority.

Exact duplicate detection is deterministic. Arbitrary natural-language overlap
and atomicity are not proven by a structural linter. Shared spans can support
different claims; overlap alone is not an invalid question or evidence conflict.
Reviewed semantic definitions and any unresolved ambiguity must be recorded.

Invalid packs or questions produce `pack_error` or `question_error` and skip
model execution. Insufficient eligible evidence in an otherwise valid pack is
an epistemic finding, not automatically a malformed-pack error.

## Compatibility semantic judgments

Use atomic Noul predicates for support, establishment, conflict, and semantic
safety. Use Choice for one mutually exclusive action recommendation and Score
only for an ordered rubric. Noul signals are independent; confidence and
probability remain model outputs, not external authority.

The verifier applies thresholds from PolicySnapshot. Thresholds require
calibration per entity type, resolved model, provider path, and policy; this
specification supplies no universal numeric default.

## Compatibility post-decision verification

The verifier MUST check for compatibility evaluation:

- Exact pack, QuestionSet, policy, adapter, and resolved-model bindings.
- Required answers, answer domains, numeric validity, and declared types.
- Evidence bindings and required spans against frozen sources and checksums.
- Evidence classes and trust; inference alone cannot justify factual claims.
- Structural contradictions and declared semantic conflict findings.
- Policy thresholds and consistency between predicates and recommended action.
- External approvals and operation restrictions where applicable.

Malformed answers are decision failures. A valid answer below a threshold is
not automatically a decision failure. A failed policy gate is not the same as
a broken policy evaluator. Verification cannot create missing provenance.

The evidence checks are the same whether the frozen state was built by LeX
from caller text or accepted from the caller as a Context frozen state, and
the same at evaluation time and at replay. LeX owns the controls: pack hash
binding, project and runtime binding, the profile focus with no caller
controls, per-item provenance, byte checksum, full-source surface, and
admissibility. Context owns the mechanism: the verifier asks the pinned
runtime to rebuild the snapshot and pack from the frozen sources and pack
request and compares identities and canonical hashes. It does not recompute
retrieval, selection, budgeting, or rejection itself. A snapshot Context
does not reproduce is `snapshot_identity`; a pack it does not reproduce is
`pack_rebuild`.

## Compatibility verdict meanings

- `validated`: the declared validation obligations pass deterministic checks;
  it does not imply an operation executed.
- `rejected`: sufficient admissible evidence establishes a negative result
  against the declared criteria.
- `insufficient`: evidence needed for a determination is absent or too weak.
- `conflict`: admissible evidence supports incompatible conclusions.
- `manual_review`: an operational review requirement prevents automated
  disposition; it is not a synonym for missing or conflicting evidence.
- `error`: a technical or contract failure prevents correct evaluation.

The report MUST preserve all material epistemic and policy findings.
A review recommendation must not erase an established conflict. The primary
verdict precedence is `error > conflict > insufficient > manual_review >
rejected > validated`. A policy denial is one of `safety_gate`,
`review_required`, or `action_inconsistent`. An evaluator that cannot apply
its thresholds is `policy_error`, not one of those denials.
The precedence order does not erase lower-priority findings.

## Embedded claim-validation profile 0.2.0

The QuestionSet, policy, and verifier are versioned together as `0.2.0`.
`established` concerns the positive claim. The separate atomic Noul `refuted`
concerns sufficient evidence for its negation, with policy threshold
`refute_min = 0.8`. Low support alone never establishes a negative result.
`conflict` concerns mutually incompatible evidence conclusions, not merely
an item contradicting the claim. Simultaneous support or establishment and
refutation above their thresholds also produces a conflict finding.

Established refutation produces `negative_result`; absent positive support or
establishment does not add insufficiency in that case. Operational review and
inconsistent action recommendations remain separate findings and retain the
existing precedence. The thresholds remain uncalibrated.

A Choice selects one of the maximum-probability options; exact ties are allowed.
A non-winning selection is `invalid_choice`, without rewriting the raw answer.
Reserved answer fields are case-sensitive. Case aliases such as `NOUL` are
`invalid_answers`; unrelated provider additions remain metadata. The loose raw
answer envelope allows malformed answer values to remain in an error bundle;
this is distinct from the verifier's strict typed-answer acceptance contract.
Invalid UTF-8 and unpaired escaped surrogates are rejected before decoding can
repair data or a bundle can seal changed values.

Previous profile/version bindings are refused rather than interpreted under
these new rules. Historical replay requires the historical verifier.

## Failure stages

- `retrieval_error`: a retrieval transport failed. This profile has no separate retrieval service. An empty exact selection is HTTP 200 `insufficient`, does not call a provider, and still returns a replay bundle. A failure while freezing sources is HTTP 422 `pack_error`. Cancellation remains HTTP 504 `deadline_exceeded` or HTTP 499 `client_canceled`. A canceled or expired pack rebuild stays that cancellation and is not reported as a pack mismatch.
- `pack_error`: pack construction, integrity, or contract validation fails. HTTP 422. A caller-frozen `context` that is not a Context frozen state, or that Context cannot reproduce, is this error before any provider call; the response lists the same findings the verifier would emit at replay.
- `question_error`: the QuestionSet or request shape violates its contract. HTTP 422 for a generic schema or semantic contract failure, with the failing JSON pointer in `detail`; generic bodies that are not JSON are 400 `invalid_json`; the legacy route returns its established request mapping. A legacy query that is empty after trimming is this error. A caller-frozen legacy `context` whose pack request query differs from `query` is this error. It is not an `insufficient` verdict.
- `decision_error`: provider execution or typed-answer validation fails. A provider contract failure is HTTP 502. A generic structural-answer failure is HTTP 422 and still returns the sealed decision bundle, but it does not manufacture a domain verdict. A legacy technical `error` verdict is HTTP 422 and still returns its sealed bundle. A retryable provider status is HTTP 503 `provider_unavailable` and is not retried. A refused provider redirect is a contract failure, not a temporary outage, and the endpoint address is not returned. A provider client timeout is `provider_unavailable` while the request context is still active. The request deadline stays `deadline_exceeded`. A response that contains the decision credential, including a Unicode-escaped copy, is rejected and that credential is not retained. Provider JSON deeper than 32 levels is a decision failure, and that body is not retained.
- `policy_error`: the evaluator cannot apply its thresholds. HTTP 500. A threshold denial stays a finding on the verdict response.
- `verification_error`: deterministic verification cannot complete correctly. HTTP 500.
- `execution_error`: an authorized operation or its postcondition verification fails. This slice has no execution route.

Accept is matched by specificity against `application/json`. An explicit refusal of that type outweighs `application/*` and `*/*`. A response that does not fit the budget is HTTP 422 `response_budget`. The provider is not called when the frozen pack already fills that budget, and a provider body larger than the remaining budget is not retained. No execution status is advertised as success.

Keep original stage and cause in the trace; do not describe every failure as
a model error. The reason codes above are the v0.1 wire mapping.

## Replay and conformance

Generic replay consumes saved DecisionSet, State, QuestionSet, optional Context
binding, and pins under a structural verifier. It compares structural findings,
not fresh model outputs or a domain verdict. Compatibility replay also compares
its legacy verdict. [analysis.md](analysis.md) owns the single release
checklist: schemas/hashes, verifier, fixtures, adversarial cases, replay, and
two-adapter interoperability.
