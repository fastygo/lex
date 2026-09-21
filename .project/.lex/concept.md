# LeX concept

Status: conceptual architecture. Normative object semantics and lifecycle are
owned by [protocol.md](protocol.md); implementation priorities by [analysis.md](analysis.md).

## Definition

**LeX is an operational infrastructure for verification.**

It transforms uncertain, heterogeneous, and potentially conflicting inputs into governed, inspectable, and reproducible outcomes.

The protocol is one part of LeX, not the whole system. The complete concept includes:

- a shared information model;
- evidence acquisition and qualification;
- semantic interpretation;
- typed judgments;
- policy and authority;
- deterministic verification;
- controlled execution;
- receipts, traces, and replay;
- conformance and calibration.

LeX is independent of subject matter. It does not define what a valid fact, object, relationship, transformation, or action means in every field. It defines how a domain declares those meanings, binds them to evidence, evaluates uncertainty, authorizes consequences, and proves what occurred.

## Why X

The **X** in LeX marks a place where independent flows cross and exchange value.

In LeX, that place is not a centralized store of all information. It is a governed junction where context from different origins is:

- identified;
- normalized without replacing the original;
- compared without being silently collapsed;
- filtered by trust and relevance;
- exchanged through typed contracts;
- transformed into explicit predicates;
- admitted or rejected by policy;
- bound to a verifiable operation.

The underlying idea is to create a context crossroads and validation exchange: a common operational surface where heterogeneous evidence can meet without losing provenance, and where different judgment mechanisms can contribute without becoming the source of authority.

```text
source streams
      \
       X  → qualified context → verified operation
      /
policy and semantic constraints
```

### X as crossroads

Context rarely arrives as one coherent record. It is distributed across sources, versions, observations, derived artifacts, and competing interpretations.

Context Runtime and its evidence adapters bring those paths to a controlled intersection. LeX governs their handoff rather than implementing another retrieval engine. Across these components, the system performs:

1. identity resolution;
2. deduplication;
3. temporal and version alignment;
4. trust filtering;
5. semantic projection;
6. conflict preservation;
7. evidence ranking;
8. budget-aware packing;
9. decision binding;
10. verification.

The crossroads does not force every path into one conclusion. It records where paths agree, where they diverge, and whether the available intersection is sufficient for action.

### X as exchange

LeX defines an exchange between otherwise independent systems:

```text
evidence providers
  <-> semantic profiles
  <-> judgment mechanisms
  <-> policy authorities
  <-> execution mechanisms
  <-> verification and audit
```

The exchanged units are not unrestricted prose. They are typed, versioned, and addressable protocol objects.

This makes the exchange:

- inspectable rather than implicit;
- interoperable rather than implementation-specific;
- reversible where operations permit it;
- replayable rather than session-bound;
- measurable rather than based on asserted confidence.

### X as the unknown

X also represents the unresolved object under validation.

```text
X = the status, meaning, relationship, or operation not yet established
```

LeX does not assume that X must resolve to true or false. The result may be established, rejected, insufficient, conflicting, deferred, or blocked by policy.

The purpose of the system is not to manufacture certainty around X. It is to determine what can legitimately be concluded and what operation, if any, is authorized.

### X as crossing constraints

An outcome is valid only at the intersection of several independent conditions:

```text
relevant evidence
× admissible source
× semantic fit
× acceptable uncertainty
× policy authority
× verified postcondition
= governed outcome
```

Here the multiplication sign is conceptual: if a mandatory condition is absent, verified execution success cannot be claimed. A validation verdict concerns its declared obligations, not proof that an operation occurred.

### X in context governance

For context algorithms, X is the boundary between accumulation and use.

Before X, the system gathers and qualifies context. At X, it decides which projection is lawful for a specific intent. After X, only the frozen projection and its policy may influence the operation.

```text
context universe
  → retrieval
  → qualification
  → X: governed projection
  → judgment
  → operation
  → receipt
```

This prevents an unbounded context pool from silently becoming an unbounded source of authority.

In compact form:

> X is the governed crossroads where context becomes eligible for decision, and the exchange where a decision becomes accountable to evidence.

## Core purpose

LeX governs the path:

```text
observation
  → evidence
  → meaning
  → judgment
  → authority
  → verification
  → operation
  → receipt
```

The purpose is not to eliminate uncertainty. The purpose is to make uncertainty explicit and prevent it from silently becoming authority.

## What makes LeX infrastructure

A protocol only specifies how participants communicate. LeX also provides the operational conditions under which communication can become a verified result.

LeX therefore contains several coordinated layers.

### Evidence layer

Context Runtime and evidence adapters acquire, identify, version, classify, and preserve source material. LeX specifies and verifies the handoff obligations.

Every relevant item can carry:

- stable identity;
- origin;
- version;
- checksum;
- span or location;
- trust level;
- evidence class;
- derivation lineage;
- acceptance or rejection reason.

Evidence remains distinct from interpretations produced from it.

### Semantic layer

Defines the meanings used by a validation process:

- entities;
- claims;
- predicates;
- relations;
- criteria;
- ordered rubrics;
- domain constraints.

The semantic layer allows different domains to use the same validation machinery without forcing them into one universal ontology.

### Judgment layer

Evaluates explicit, typed questions over a frozen evidence state.

Judgments may express:

- whether a predicate is supported;
- whether a claim is established;
- whether evidence conflicts;
- where an entity lies on an ordered scale;
- which operational branch is preferred;
- how uncertain the result remains.

A judgment is a derived signal. It is never its own evidence.

### Policy layer

Defines when a judgment may influence an operation.

Policy includes:

- admissible evidence;
- required trust;
- confidence and uncertainty thresholds;
- risk classification;
- permissions;
- required approvals;
- conflict handling;
- escalation rules;
- version compatibility.

Policy is external to the judgment mechanism and can be inspected independently.

### Verification layer

Checks that the declared result follows from the protocol state.

Verification binds:

- the entity;
- the evidence;
- the semantic definitions;
- the judgments;
- the policy;
- the requested operation.

It determines whether preconditions were satisfied, conflicts were handled, required authority existed, and the postcondition was reached.

### Execution layer

Performs only operations authorized by the verified state.

Execution is not implied by a model response or a winning classification. It requires an explicit operation contract and a policy decision.

Operations remain observable, cancellable when possible, and attributable to a specific validation run.

### Receipt and trace layer

Records what was evaluated, what was authorized, what was executed, and what was verified.

A trace preserves the complete path. A receipt summarizes the completed operation and its proof obligations.

This layer enables:

- audit;
- replay;
- comparison;
- regression analysis;
- calibration;
- accountability.

### Conformance layer

Determines whether an implementation follows LeX.

Conformance covers:

- schemas;
- state transitions;
- normative requirements;
- error behavior;
- compatibility;
- test vectors;
- invalid fixtures;
- replay guarantees.

No implementation becomes conformant merely by using the same field names.

## Universal validation model

LeX separates four questions that systems often collapse:

1. **Support** — does admissible evidence point toward the proposition?
2. **Establishment** — is the evidence sufficient and coherent enough to accept the proposition?
3. **Authority** — is the system permitted to act on that proposition?
4. **Execution** — did the authorized operation reach its declared postcondition?

These stages produce different outputs and failure modes.

```text
support without establishment
  → insufficient or conflict

establishment without authority
  → manual review or policy denial

authority without successful execution
  → execution error

execution without verified postcondition
  → incomplete receipt
```

## Core objects

LeX uses a small set of stable abstractions:

- **EntityEnvelope** — what is being evaluated;
- **ValidationIntent** — what must be determined;
- **EvidenceSet** — admissible and rejected material;
- **SemanticProfile** — definitions and constraints;
- **QuestionSet** — typed judgments to evaluate;
- **DecisionSet** — raw judgment outputs;
- **PolicySnapshot** — frozen authority and risk rules;
- **OperationContract** — preconditions, operation, and postconditions;
- **VerificationReport** — deterministic contract checks;
- **Verdict** — epistemic and operational disposition;
- **EvaluationTrace** — replayable record;
- **Receipt** — execution record with explicit postcondition verification status; only a successful verified result supports a success claim.

These are logical objects. Their transport and serialization may vary while their semantics remain stable.

## Verdict model

LeX does not reduce every outcome to pass or fail.

It preserves distinct states:

- `validated` — evidence, judgment, policy, and verification agree;
- `rejected` — sufficient evidence establishes a negative result against the declared criteria;
- `insufficient` — required evidence is absent or too weak;
- `conflict` — admissible evidence supports incompatible conclusions;
- `manual_review` — an operational review requirement prevents automated disposition;
- `error` — the validation lifecycle did not complete correctly.

Epistemic states and operational states must remain distinguishable.

## Scale and extensibility

LeX scales by keeping the core subject-neutral and moving specialized meaning into versioned profiles.

The core owns:

- lifecycle;
- evidence contracts;
- judgment contracts;
- policy binding;
- verification;
- verdict semantics;
- trace and receipt rules;
- conformance.

Domain profiles own:

- entity schemas;
- terminology;
- admissible sources;
- predicates;
- criteria;
- rubrics;
- risk models;
- operation contracts.

This boundary allows LeX to span radically different forms of validation without becoming a collection of unrelated special cases.

## System properties

LeX aims to provide:

- **inspectability** — every outcome can be decomposed;
- **traceability** — every claim points to its origin;
- **reproducibility** — frozen inputs and saved decisions can be replayed through a pinned verifier;
- **composability** — validation stages can be combined;
- **replaceability** — evidence and judgment providers are adapters;
- **calibration** — uncertainty can be measured against outcomes;
- **controlled autonomy** — operations remain policy-bound;
- **graceful abstention** — uncertainty is preserved rather than hidden;
- **interoperability** — independent implementations share semantics;
- **progressive assurance** — stronger proof requirements can be added without changing the conceptual core.

## What LeX is not

LeX is not:

- a universal source of truth;
- an ontology for every domain;
- a single model or provider;
- a replacement for expert judgment;
- a guarantee that all evidence is correct;
- a mechanism that converts confidence into authority;
- an execution engine without policy;
- a collection of prompts;
- a binary pass/fail classifier;
- a requirement to centralize all data.

## Architectural form

LeX should remain a thin universal core surrounded by replaceable and domain-specific components.

```text
domain profile
      ↓
evidence adapters → evidence plane
      ↓
semantic profile → judgment plane
      ↓
policy snapshot → verifier
      ↓
operation contract → execution
      ↓
verdict + trace + receipt
```

The core should standardize boundaries and proofs, not absorb the internal responsibilities of every component.

## Protocol within LeX

The LeX Protocol defines:

- canonical object semantics;
- lifecycle and state transitions;
- normative invariants;
- serialization profiles;
- compatibility rules;
- conformance requirements.

The LeX infrastructure implements and operates those contracts.

```text
LeX concept
  |-- protocol
  |-- schemas
  |-- verifier
  |-- policy engine
  |-- adapters
  |-- execution controls
  |-- trace and receipt store
  `-- conformance suite
```

## Central principle

The central principle of LeX is:

> No uncertain interpretation becomes an authorized operation without explicit evidence, policy, and verification.

In compact form:

```text
evidence does not equal judgment
judgment does not equal authority
authority does not equal execution
execution does not equal verified success
```

LeX makes every transition explicit.
