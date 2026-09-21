# What LeX verification infrastructure includes

Status: canonical scope constraints. Detailed semantics belong to [protocol.md](protocol.md).
The full system scope below is broader than the first validation-only slice in [analysis.md](analysis.md).
The current [deployment plan](../.plan/README.md) is Go REST on Vercel using
Framework + Context, request RAM, and caller-owned replay bundles. TypeScript,
databases, durable server history, and side-effect execution are deferred.

LeX is a subject-neutral infrastructure for evidence, judgment, policy, verification, controlled execution, and proof of outcome.

The LeX Protocol standardizes the contracts between those layers. It is not the complete system by itself.

## Included in the LeX notion

### 1. Entity under validation

Anything that can be frozen, referenced, and checked, for example:

- documents and claims;
- source artifacts and chunks;
- events and observations;
- tool outputs and model outputs;
- code changes and structured artifacts;
- candidate actions (route, approve, refund, delete);
- taxonomies and criteria definitions.

Each run uses an **EntityEnvelope**: stable id, type, schema, version, checksum, and source refs.

### 2. Validation intent

What is being checked, without prescribing the answer:

- objective (e.g. route ticket, verify refund eligibility);
- FocusProfile and retrieval scope;
- QuestionSet id and version;
- PolicySnapshot id;
- risk class (reversible vs destructive).

### 3. Evidence plane (Context Runtime)

The Context integration contract requires the following capabilities, subject to verification against the pinned upstream API:

- project-scoped isolation;
- retrieval under the declared FocusProfile; exact, sparse, dense, and query-operator paths are upstream mechanisms, not mandatory LeX algorithms;
- **ContextPack** separating instructions, policy references, accepted evidence, and rejected material according to the pinned upstream schema;
- **evidence classes** (`source_text`, `attestation`, authoritative `tool_output`, `model_inference`, etc.);
- **trust levels** and citation spans with checksums;
- replayable index snapshots and lineage for derived artifacts.

LeX checks that only admissible evidence classes justify factual predicates.

### 4. Decision plane (provider-neutral adapters)

LeX expresses judgments with typed primitives; Jev is the reference model:

| Primitive | Use |
| --- | --- |
| **Noul** | Independent yes/no predicates (support, established, conflict, safe to act) |
| **Choice** | One operational switch (route, accept, reject, manual_review) |
| **Score** | Ordered rubrics (evidence sufficiency, severity, urgency) |

LeX stores the raw **DecisionSet** with exact typed answers, input bindings, adapter version, and resolved model identity; usage, timing, request id, and provider metadata are retained when available.

### 5. Separation of predicates

LeX explicitly models distinct concepts (do not collapse into one Choice):

- **supported** — eligible evidence points at a hypothesis;
- **established** — evidence meets the declared profile's sufficiency and coherence requirements;
- **safe_to_auto_act** — a semantic safety signal that does not grant authority;
- **action** — recommended operational branch, subject to deterministic policy and external authority.

Support for two routes at once is valid; automatic action when both conflict is not.

### 6. Preflight checks

Before invoking a decision adapter:

- exact duplicate definitions; semantic overlap requires reviewed definitions or explicit semantic assessment;
- missing `other` / `manual_review` when taxonomy is open or risky;
- Score levels ordered; Choice options mutually distinguishable;
- required state fields present;
- context budget respected;
- optional span-overlap diagnostics; shared spans alone do not prove conflicting criteria.

### 7. Verification

After decision output, deterministic verification:

- resolve required EvidenceBindings to frozen sources and spans; provider answers need not supply citations themselves;
- enforce required trust and evidence class rules;
- reject verdicts based only on `model_inference`;
- apply thresholds from PolicySnapshot (not universal constants);
- require external confirmation for destructive actions;
- detect structural conflicts in authoritative tool outputs.

### 8. Verdict and trace

Final **Verdict** statuses:

- `validated`
- `rejected`
- `insufficient`
- `conflict`
- `manual_review`
- `error`

**EvaluationTrace** pins: entity version, pack checksum/version, QuestionSet hash, profiles, adapter and verifier versions, resolved model id, policy, raw decisions, and verification report. A Verdict does not prove execution; an execution Receipt records postcondition verification separately.

### 9. Error taxonomy

LeX distinguishes:

- retrieval_error
- pack_error
- question_error
- decision_error
- policy_error
- verification_error
- execution_error

“Model wrong” is not a substitute for diagnosing the pipeline stage.

### 10. Optional consumer obligations (out of core)

Downstream products may require human-facing rules (source for claims, authority for actions, receipt for outcomes). LeX maps those to ContextPack + PolicySnapshot + VerificationReport + trace. It does not define CMS taxonomies or public publishing graphs.

## Explicitly excluded

- Prose generation as a validation outcome.
- Model-only `validated` without verification.
- Using Jev confidence as proof of source truth.
- Treating retrieval score as calibrated probability of a claim.
- Hidden merge of conflicting authoritative sources.
- A single overlapping Choice instead of independent support Noul + action Choice.
- Universal thresholds without domain calibration.

## Maturity note

The **v0.1 working draft** has research observations on **hand-authored** states in `.project/.jev/`. These do not prove implemented LeX behavior. Release requires the proof gates in [analysis.md](analysis.md), including real ContextPack API runs, conformance fixtures, calibration, replay, and interoperability.
