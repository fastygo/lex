# LeX Protocol

Status: canonical normative specification, canary release. Wire envelopes:
generic `0.2`, legacy compatibility `0.1`. Requirement keywords and canary
compatibility promises follow [governance.md](governance.md). Stable-release
gates are tracked in [analysis.md](analysis.md).

## Purpose

LeX defines provider-neutral contracts for typed decisions, optional evidence
bindings, deterministic structural verification, controlled operations, and
proof artifacts. Consumers supply domain schemas, criteria, risk rules,
thresholds, and authority.

The canonical decision operation is:

```text
caller State + caller QuestionSet + optional frozen Context binding
  -> typed-decision adapter
  -> raw DecisionSet
  -> structural verification
  -> trace + replay bundle
```

The caller owns question meaning, interpretation, sequencing, and any later
tool call or action. LeX does not select a domain profile, retrieve evidence,
rewrite state, derive a semantic domain verdict, or choose the next step.
Claim-validation is a compatibility consumer of this operation, not the
identity of the core protocol.

## Participants

- Consumer: supplies decision identity, state, QuestionSet, and any
  interpretation or action policy.
- Context Runtime: optionally retrieves evidence and builds a ContextPack
  before the consumer calls LeX.
- Profile author: compatibility-only author of versioned meanings and atomic
  questions.
- Policy authority: compatibility-only supplier of rules and any required
  external approvals.
- Decision adapter: obtains typed answers over frozen inputs.
- Verifier: checks bindings and policy deterministically.
- Trace custodian: preserves inputs, outputs, and verification artifacts.
  In the current RAM-only profile, the service returns a bundle and the caller
  owns retention; no durable server-side trace store is implied.
- Human reviewer: supplies external review when required.
- Executor: performs an explicitly authorized operation when execution is supported.

## Core invariants

1. A decision MUST NOT be evidence for its own verdict.
2. Evidence MUST be addressable by identity, provenance, version, and checksum.
3. Questions MUST be typed and atomic: Noul, Choice, or Score.
4. Support, establishment, authority, action, and verified success MUST remain distinct.
5. Implementations MUST preserve `insufficient` and `conflict` as valid outcomes.
6. Policy and thresholds MUST be external to the judgment model.
7. A compatibility `validated` verdict MUST require deterministic verification.
8. Generic runs pin project, decision identity, State, QuestionSet, adapter,
   resolved model, and verifier versions. An optional Context binding is pinned
   when present. Runs MUST pin entity, ContextPack, QuestionSet, policy, adapter, and resolved model versions for compatibility validation. Profile and verifier versions MUST also be recorded.
9. A criteria change MUST produce a new QuestionSet version and hash.
10. A model alias such as `latest` MUST NOT substitute for a resolved model identifier.

These requirements apply to the first slice, not only a future release.

## Protocol entities

These are logical contracts, not final JSON field definitions. Schemas must
specify required fields, reference resolution, and hash scope before release.

### DecisionIdentity and State

DecisionIdentity is caller-owned stable id and version for one decision step.
State is a caller-owned JSON object. LeX validates its transport form and
canonical hash, then supplies its semantics unchanged to the adapter. State is
not evidence by default and it is not implicitly populated from Context.

### EntityEnvelope

Identifies the evaluated entity, its project, type, schema, version, checksum,
and source references. It is a compatibility-validation contract; generic
decisions use DecisionIdentity instead.

### ValidationIntent

Identifies the objective, entity, FocusProfile, SemanticProfile, QuestionSet,
PolicySnapshot, and risk class. It requests evaluation, not a predetermined result.
It is a compatibility-validation contract, not required by generic decisions.

### QuestionSet

Exact versioned typed questions, hash, per-question purpose, answer domains,
and capability requirements. Generic QuestionSets are caller-owned and contain
Noul, Choice, or Score questions only. A Choice declares exact unique options;
a Score declares two through ten ordered levels. LeX validates this structure
but does not decide whether a taxonomy is useful for a particular product.

### FocusProfile and ContextPack

Context-owned contracts. FocusProfile constrains evidence selection, trust,
classes, and budget. ContextPack is an optional frozen evidence handoff. LeX
records and verifies an optional upstream binding rather than redefining or
automatically merging it into State.

### EvidenceSet and EvidenceBinding

EvidenceSet is the logical view of accepted and rejected pack material, not a
second retrieval format. EvidenceBinding connects required predicates to
addressable evidence, including source identity, class, trust, checksum,
version/snapshot, applicable spans, lineage, and acceptance/rejection reasons.
A provider answer need not contain citations; bindings must independently exist
and pass verification when required by the profile.

### SemanticProfile

Versioned domain definitions, atomic predicates, criteria, ordered rubrics,
evidence requirements, and mappings to a QuestionSet. It is a compatibility
consumer artifact, not a universal ontology or a generic-decision requirement.

### PolicySnapshot

Frozen evidence eligibility, trust requirements, thresholds, risk rules,
permissions, approval requirements, and allowed component/model versions.
Credentials themselves are not persisted in this object or in traces. It is
required only when a compatibility consumer derives semantic findings or a
Verdict.

### DecisionSet

Raw typed answers bound to exact State, QuestionSet, optional Context binding,
adapter version, and resolved model identity. Provider, request identifier,
timing, and usage are recorded when available as adapter metadata. Missing
metadata must not be fabricated. Missing reproducible model identity prevents a
conforming replay claim. Transport normalization MUST NOT rewrite the semantic answers.

### VerificationReport

Generic deterministic checks cover input integrity, question/answer types,
answer domains, optional Context binding, adapter/model pins, and supported
versions. A compatibility verifier may additionally check evidence eligibility,
conflicts, thresholds, and authority. Neither form proves every natural-language
proposition true.

### Verdict

The compatibility-validation disposition:
`validated | rejected | insufficient | conflict | manual_review | error`.

[checks.md](checks.md) owns the detailed definitions. Verdict is distinct from
a recommended action, operation progress, and execution success. All material
findings must remain in the report even when one primary verdict is selected.

### DecisionTrace and EvaluationTrace

DecisionTrace records generic stage outcomes and binds State, QuestionSet,
optional Context binding, raw DecisionSet, structural report, component
versions, and hashes. EvaluationTrace extends it with compatibility Verdict and
policy findings. Trace data MUST exclude credentials and provider secrets.
A replay record must retain the authorized evidence needed to resolve bindings;
a redacted export must disclose any resulting replay limitation. In the
[current deployment profile](../.plan/architecture.md), the service retains
artifacts only for the request and returns a complete replay bundle. Replay
requires the caller to resubmit it; response loss is not recoverable from server
history. Runtime persistence is not an invariant; complete bindings and honest
retention guarantees are.

### OperationContract and Receipt

Execution-extension contracts. OperationContract declares the authorized action,
preconditions, external authority, and postconditions. Receipt records what
actually executed and the postcondition verification result. It MUST NOT claim
verified success merely because a model recommended an action or validation passed.
Failures and incomplete postcondition checks remain explicit.

## Compatibility predicate meanings

- `supported`: eligible evidence supports the proposition.
- `established`: the profile's sufficiency and coherence requirements are met.
- `has_conflicting_evidence`: admissible evidence supports incompatible conclusions.
- `safe_to_auto_act`: a semantic safety signal, never permission by itself.
- Action Choice: a recommendation evaluated by deterministic policy.

Independent Noul probabilities need not sum to one. High confidence does not
establish source truth, domain calibration, or authority.

## Decision lifecycle

The generic decision stage order is:

```text
RECEIVE -> DECIDE -> VERIFY
```

An optional Context binding is verified before DECIDE but does not create or
alter State. The compatibility validation lifecycle is the following extended
baseline, not a complete remote operation state machine:

```text
RECEIVE -> FREEZE -> RETRIEVE -> PACK -> PREFLIGHT
  -> DECIDE -> INTERPRET -> VERIFY -> VERDICT
```

FREEZE pins the entity, intent, policy, profiles, and exact questions. PACK freezes
the retrieved ContextPack before DECIDE. Basic input and authorization checks
run before retrieval; pack-dependent preflight runs after packing.
INTERPRET derives signals without modifying retained raw answers.
VERIFY applies deterministic gates; VERDICT records their disposition.
EvaluationTrace accumulates throughout, including early failures; it is not a
success-only final stage.

A preflight failure skips DECIDE and records the stage error and available
inputs. Valid negative or uncertain evaluations are not infrastructure errors.
A transport failure is not a DecisionSet. Retry attempts retain their own
metadata and cannot silently change provider, model, or frozen input.

## Execution boundary

The first slice ends at Verdict + EvaluationTrace. An execution extension adds:

```text
explicit authorization + OperationContract
  -> execute -> verify postconditions -> Receipt
```

A validation verdict MUST NOT be treated as credentials or proof of execution.
Changed inputs or policy require a new evaluation; execution must check that
the authority and preconditions are still applicable. Exact execution,
cancellation, recovery, and idempotency rules remain release blockers for that
extension. Cancellation is operation status, not a seventh verdict.

## Replay

Generic replay uses saved decision identity, State, QuestionSet, optional
Context binding, raw DecisionSet, adapter/model pins, and verifier version. It
MUST NOT contact a retrieval service, call a decision provider, or execute side
effects. It reproduces structural findings, not a new semantic interpretation.
When a Context binding is present it MUST compare a recomputation from the
frozen snapshot and pack request with the saved pack. The saved pack MUST NOT be replaced.
Generic replay refuses requests outside the authenticated project.
Compatibility replay MUST refuse a replay whose entity project is outside the
authenticated principal's projects. Compatibility replay additionally reproduces
deterministic findings and Verdict.

A fresh model call is a new evaluation even with the same resolved model.
Audit timestamps, request identifiers, and replay-run identifiers may differ;
byte-identical traces are not promised. Source availability and retention
constraints must be stated in any replay claim.

## Release boundary

Wire schemas, canonical hashes, transport, full transition/error rules,
verdict precedence, security, compatibility, and executable conformance vectors
must be specified and verified before interoperability is claimed. See
[analysis.md](analysis.md) and [sources.md](sources.md).
