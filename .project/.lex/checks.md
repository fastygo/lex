# LeX checks

Status: normative check catalog draft. Shared semantics belong to
[protocol.md](protocol.md); release blockers are in [analysis.md](analysis.md).

## Structural and preflight checks

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

## Semantic judgments

Use atomic Noul predicates for support, establishment, conflict, and semantic
safety. Use Choice for one mutually exclusive action recommendation and Score
only for an ordered rubric. Noul signals are independent; confidence and
probability remain model outputs, not external authority.

The verifier applies thresholds from PolicySnapshot. Thresholds require
calibration per entity type, resolved model, provider path, and policy; this
specification supplies no universal numeric default.

## Post-decision verification

The verifier MUST check:

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

## Verdict meanings

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
A review recommendation must not erase an established conflict. Exact primary
verdict precedence and policy-denial reason codes remain a release blocker;
these definitions are not a complete executable decision table.

## Failure stages

- `retrieval_error`: evidence retrieval fails.
- `pack_error`: pack construction, integrity, or contract validation fails.
- `question_error`: the QuestionSet is malformed or violates its contract.
- `decision_error`: provider execution or typed-answer validation fails.
- `policy_error`: policy resolution or evaluation fails, distinct from a valid denial.
- `verification_error`: deterministic verification cannot complete correctly.
- `execution_error`: an authorized operation or its postcondition verification fails.

Keep original stage and cause in the trace; do not describe every failure as
a model error. Wire error codes, retry eligibility, and transport mappings
must be finalized before release.

## Replay and conformance

Replay consumes saved DecisionSet and frozen inputs under a pinned verifier;
it compares deterministic findings and verdict, not fresh model outputs or
wall-clock trace identifiers. [analysis.md](analysis.md) owns the single release
checklist: real packs, schemas/hashes, persistence, verifier, fixtures,
adversarial cases, calibration, replay, and two-adapter interoperability.
