# LeX Protocol

Status: historical research draft `0.1`, non-normative and not maintained as a mirror.

This file preserves an earlier design, including examples and readiness ideas
that may conflict with the current protocol. Do not implement from this file.
Use the [canonical protocol](../.lex/protocol.md) and [release plan](../.lex/analysis.md).
The historical body below is retained for provenance.

## Purpose

LeX is a protocol for checking and validating entities using:

- Context Runtime for evidence, provenance, policy, retrieval, and replay;
- Jev for fast typed probabilistic judgments;
- a deterministic verifier to bind decisions to evidence and emit the final verdict.

Entities under validation may include:

- a document or claim;
- a source artifact;
- an event or observation;
- tool output;
- model output;
- a code change;
- a candidate action;
- a route, category, or policy decision;
- a derived structured artifact.

LeX does not define domain ontology. The consumer supplies entity schema, policy, criteria, and risk model.

## Core invariants

1. **Decision is not evidence.** A Jev answer never becomes the source evidence for its own verdict.
2. **Evidence is addressable.** Every piece of evidence has identity, provenance, version, and checksum.
3. **Questions are typed.** Every model judgment compiles to `Noul`, `Choice`, or `Score`.
4. **Support is not action.** Hypothesis support, fact establishment, safety, and action are modeled separately.
5. **Uncertainty is a valid result.** Insufficient data and conflict are not forced into a positive or negative answer.
6. **Policy is external to the model.** Risk thresholds, permissions, and destructive approvals are applied in code.
7. **Verification is mandatory for final verdicts.** A model-only run may yield a recommendation, not `validated`.
8. **Runs are replayable.** Entity, pack, question set, model, and policy versions are fixed in the trace.
9. **Criteria are versioned.** Changing definitions or Score order creates a new QuestionSet version.
10. **Model aliases are not reproducible identifiers.** The trace stores the actually resolved model version.

## Protocol entities

### EntityEnvelope

Describes the object under validation.

```json
{
  "entity_id": "ticket:A-104",
  "entity_type": "support_ticket",
  "project_id": "demo",
  "schema_id": "lex.support-ticket.v1",
  "version": "7",
  "checksum": "sha256:...",
  "created_at": "2026-09-21T00:00:00Z",
  "source_refs": ["source:ticket-A-104"]
}
```

### ValidationIntent

Defines what is being validated without prescribing the answer.

```json
{
  "intent_id": "route-support-ticket",
  "entity_ref": "ticket:A-104@7",
  "objective": "Select a safe operational route",
  "focus_profile_id": "support-routing-v2",
  "question_set_id": "support-routing-questions-v3",
  "policy_id": "support-routing-policy-v4",
  "risk_class": "reversible"
}
```

### EvidenceBinding

Links evidence to the validation.

Minimum fields:

- `evidence_id`;
- `source_id` or `artifact_id`;
- `evidence_class`;
- `trust_level`;
- `span_start` and `span_end` when applicable;
- `checksum`;
- `snapshot_id`;
- `lineage`;
- `accepted` or `rejected`;
- reason codes for inclusion or rejection.

### QuestionSet

Versioned set of Jev questions. It must store:

- exact question JSON;
- hash;
- model compatibility;
- semantic purpose of each question;
- expected post-processing;
- abstention criteria;
- links to predicates and actions.

Question IDs are for code. Full semantics belong in `instructions` and `criteria`.

### DecisionSet

Unmodified typed Jev output plus execution metadata:

```json
{
  "model": "jev-1.13.0",
  "question_set_id": "support-routing-questions-v3",
  "answers": {},
  "input_pack_id": "pack:...",
  "request_id": "...",
  "evaluation_time_ms": 98.2,
  "usage": {
    "input_tokens": 2183,
    "output_tokens": 543
  }
}
```

### VerificationReport

Checks contract compliance, not model reasoning:

- all required questions were answered;
- answer types match QuestionSet;
- cited evidence is present in the pack;
- required trust was satisfied;
- factual predicates are not based only on `model_inference`;
- structural conflicts were handled;
- thresholds were applied correctly;
- destructive approvals are present;
- model version is allowed by policy.

### Verdict

Minimum final statuses:

- `validated`: claim or action passed evidence and policy verification;
- `rejected`: evidence suffices for a negative decision;
- `insufficient`: evidence is not enough;
- `conflict`: admissible evidence contradict each other;
- `manual_review`: automation forbidden by policy or uncertainty;
- `error`: the protocol did not complete technically.

`manual_review` is an action status. `insufficient` and `conflict` are epistemic statuses. They must not collapse into one boolean.

## Standard predicates

LeX recommends the same decomposition across entity types:

### Evidence predicates

- `has_eligible_evidence`
- `has_authoritative_evidence`
- `has_conflicting_evidence`
- `has_required_provenance`

### Semantic predicates

- `<claim>_supported`
- `<claim>_established`
- `<category>_supported`

### Safety predicates

- `safe_to_auto_act`
- `requires_human_review`
- `destructive_action_allowed`

### Operational decision

One `Choice` whose options are actions, not hidden facts:

- `route_*`;
- `accept`;
- `reject`;
- `request_more_evidence`;
- `manual_review`.

## Question compilation

### Noul for independent support

If several hypotheses may have evidence at once, each gets its own Noul. These values are independent and need not sum to 1.

```json
{
  "billing_supported": {
    "type": "noul",
    "instructions": "Does eligible evidence directly support the billing route?"
  },
  "account_access_supported": {
    "type": "noul",
    "instructions": "Does eligible evidence directly support the account access route?"
  }
}
```

### Choice for action

Choice is used after predicates:

```json
{
  "action": {
    "type": "choice",
    "instructions": "Which operational action should the system take?",
    "criteria": {
      "route_billing": "Coherent evidence supports billing alone",
      "route_account_access": "Coherent evidence supports account access alone",
      "manual_review": "Evidence is insufficient, ambiguous, or conflicting"
    }
  }
}
```

### Score for ordered quality

Score applies to an explicitly ordered rubric, for example:

1. no usable evidence;
2. weak or ambiguous evidence;
3. meaningful but incomplete/conflicting evidence;
4. strong coherent authoritative evidence.

The resulting decimal is the expected position on the distribution, not necessarily the chosen discrete level.

## Lifecycle

```text
1. RECEIVE
   EntityEnvelope + ValidationIntent

2. FREEZE
   PolicySnapshot + QuestionSet version + model version rule

3. RETRIEVE
   FocusProfile → RetrievalPlan → candidate evidence

4. PACK
   accept/reject evidence → ContextPack

5. PREFLIGHT
   schema checks + criteria overlap checks + required evidence checks

6. DECIDE
   execute Jev QuestionSet once against the frozen pack

7. INTERPRET
   derive support, established, conflict, safety and action signals

8. VERIFY
   bind decisions to evidence and apply deterministic policy

9. VERDICT
   validated | rejected | insufficient | conflict | manual_review | error

10. TRACE
    persist entity, pack, questions, raw answers, policy and verification
```

## Preflight criteria checks

Before calling Jev, LeX should detect:

- duplicate option labels;
- identical or near-identical definitions;
- Score levels without strict order;
- missing `other` for open taxonomy;
- missing `manual_review` for risky action;
- criteria supported by the same evidence spans;
- one question combining several independent predicates;
- references to missing state fields;
- context budget exceeded.

Some checks are deterministic. Semantic overlap may use a separate Jev/embedding/reranker signal, but that result remains diagnostic, not source truth.

## Interpreting uncertainty

LeX does not set a universal threshold.

- For Noul, a value near `0.5` means yes/no uncertainty.
- For Choice and Score, use `confidence` and the full distribution.
- Threshold depends on domain risk, error cost, and model version.
- High confidence does not waive required evidence.
- Low confidence does not always mean a bad question; it may honestly reflect source conflict.

Thresholds must be calibrated on labeled data and stored in PolicySnapshot.

## Minimal policy

```json
{
  "policy_id": "support-routing-policy-v4",
  "model_version": "jev-1.13.0",
  "required_trust_level": "project",
  "allowed_fact_evidence_classes": [
    "source_text",
    "attestation",
    "tool_output_authoritative"
  ],
  "auto_action": {
    "minimum_noul": 0.9,
    "minimum_choice_confidence": 0.9,
    "require_safe_to_auto_act": true,
    "deny_on_conflict": true
  },
  "destructive": {
    "require_external_confirmation": true
  }
}
```

Numbers illustrate policy shape, not recommended production thresholds.

## Protocol errors

LeX distinguishes:

- `retrieval_error`: required evidence not found;
- `pack_error`: evidence lost, misclassified, or truncated;
- `question_error`: criteria overlap or question merges predicates;
- `decision_error`: Jev gave a substantively wrong answer;
- `policy_error`: threshold or permission misconfigured;
- `verification_error`: decision incorrectly bound to evidence;
- `execution_error`: timeout, transport, schema, or provider failure.

This split matters: “Jev was wrong” must not explain every pipeline failure.

## LeX v1 readiness criteria

LeX v1 is ready when:

1. protocol entity schemas are frozen and versioned;
2. Context Runtime produces real replayable packs;
3. the Jev adapter stores raw typed answers without transformation;
4. the verifier emits all epistemic and action statuses;
5. a golden corpus reproduces expected verdicts;
6. an adversarial corpus covers injection, noisy inference, missing evidence, and conflicts;
7. a calibration report links thresholds to measured error rates;
8. destructive actions cannot run without external approval;
9. one trace reproduces the decision with pinned versions;
10. replacing Jev with another typed decision adapter does not change the core protocol.
