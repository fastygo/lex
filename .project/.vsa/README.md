# Vertical Slice Architecture with Functional Control Contracts

Status: implementation architecture guidance, subordinate to the
[canonical protocol](../.lex/protocol.md). This is not a second protocol or
an implementation-completion claim. The first slice is validation-only.

This document specifies the architectural theses for autonomous verification and execution units using Vertical Slice Architecture (VSA) and Structured Analysis and Design Technique (SADT / IDEF0) functional decomposition.

## 1. Vertical Slice Architecture and Functional Decomposition

1. **System Partitioning by Vertical Slices**: The system is organized into decoupled vertical slices rather than horizontal technical tiers. Each slice represents an autonomous transaction boundary encapsulating its own input acquisition, context projection, semantic evaluation, policy enforcement, execution mechanisms, verification routines, and audit artifact generation.
2. **Functional Modeling via ICOM**: Within each vertical slice, the operational boundary is modeled using the four formal relationships of SADT / IDEF0:
   - **Input (I)**: The raw data, evidence items, and entity payloads undergoing processing or evaluation.
   - **Control (C)**: The governing rules, constraints, schema invariants, policy snapshots, risk parameters, and acceptance criteria (Done-iff conditions) that dictate the validity of the transformation.
   - **Mechanism (M)**: The execution resources utilized to execute the operation (retrieval adapters, language models, typed decision evaluators, tools, deterministic verifiers).
   - **Output (O)**: The produced structured artifacts, authorized state mutations, validation reports, and execution receipts.
3. **Immutability of Inter-Slice Interfaces**: Slices do not share mutable state. Communication between slices occurs strictly via immutable artifacts, content-addressed references, and explicit event signals.

## 2. Decoupling Mechanism from Control

1. **Autonomous Operation within Bound Invariants**: The Mechanism layer possesses internal execution latitude (such as selection of sub-paths, model sampling, parallel tool execution, or decomposition depth) provided it operates entirely within the boundaries established by Control.
2. **Invariance of Control Rules**: Control constraints are externally imposed and non-negotiable. A Mechanism cannot renegotiate, loosen, or override Control rules during execution.
3. **Invalidity of Non-Conformant Output**: Any operational output that violates an active Control constraint is invalid by definition. Incomplete satisfaction of Control rules does not yield a degraded or partial output; it constitutes an execution failure or an explicit abstention.
4. **Separation of Instructions from Data**: Control instructions and policy snapshots must remain in the control plane and must not be mixed into evidence items or data plane payloads.

## 3. Semantic Transduction versus Discrete Operational Gating

1. **Probabilistic Nature of Semantic Transducers**: Natural language models and typed decision models are probabilistic transducers, not deterministic switches. They evaluate semantic context against predefined answer spaces and emit continuous probability distributions over candidate sets.
2. **Non-Equivalence of Probability and Authority**: Probability, distribution concentration, and empirical calibration are distinct concepts. None is an authority credential. It does not constitute operational authorization.
3. **External Realization of the Discrete Gate**: A discrete operational gate (permitting or rejecting a state transition) requires an external, deterministic policy function:
   ```text
   Gate(Distribution, Policy, AuthoritativeEvidence) -> { ALLOW, DENY, ABSTAIN, ESCALATE }
   ```
   The semantic model supplies the measurement; the deterministic policy engine computes the transition.
4. **Decomposition of Hypotheses and Operations**: Semantic evaluation must separate proposition support from operational decisions:
   - Proposition support evaluates whether evidence aligns with a specific claim (modeled via independent boolean propositions).
   - Operational decisions select a mutually exclusive action branch (modeled via categorical choices with mandatory non-action alternatives).
   - Conflating hypothesis evaluation with operational action selection into a single categorical query causes probability mass distortion during evidence conflicts.

## 4. Multi-Tier Boundary Enforcement

In-prompt natural language instructions are inherently soft constraints prone to attention degradation, instruction conflict, and context loss. System integrity requires programmatic boundary enforcement structured into three tiers:

1. **Deterministic Boundaries**:
   - Enforced by strict software logic: schema validation, cryptographic checksums, token budgets, byte-span bounds, and tool allowlists.
   - Stage-local enforcement: checks run when their inputs are available. Preflight violations stop model invocation; later violations stop authorization or success claims.
2. **Semantic Boundaries**:
   - Evaluated by typed probabilistic models against explicit atomic definitions. Operational Choice alternatives are mutually exclusive; independent support predicates may overlap. Calibration requires measurement.
   - Uncertainty-aware: results falling within bounded ambiguity ranges trigger automated abstention rather than forced transitions.
3. **Authoritative Boundaries**:
   - Governed by external credentials, explicit multi-party approvals, or human verification.
   - Irreversible or destructive operations (such as asset transfers, permanent record deletions, or external state modifications) are programmatically isolated and cannot be authorized by semantic models alone.

## 5. Separation of Functional Contracts and Dynamic Execution Flow

1. **Functional Incompleteness of Static SADT**: SADT / IDEF0 specifies declarative functional requirements (what transforms what, under what constraints, using what resources, to produce what result). It lacks primitives for dynamic temporal execution, such as:
   - Polling loops and timeouts;
   - Retry strategies and backoff intervals;
   - Compensation and rollback transactions;
   - Asynchronous branch-and-join execution;
   - Process cancellation and crash recovery.
2. **Orthogonal State Machine Architecture**: Complete automation requires an orthogonal separation:
   - The **Functional Contract** defines the transformation invariants and the acceptance conditions (Done-iff).
   - The **Deterministic State Machine** executes the temporal flow, coordinates adapters, evaluates transitions, and manages failure states.
3. **Functional decomposition of an execution-capable slice (not wire states)**:
   ```text
   RECEIVE -> QUALIFY -> PREFLIGHT -> EVALUATE -> RESOLVE -> VERIFY -> COMMIT -> RECEIPT
   ```
   - **RECEIVE**: Ingest entity envelope and validation intent.
   - **QUALIFY**: Retrieve context, enforce trust filters, and build immutable evidence pack.
   - **PREFLIGHT**: Validate schemas, detect exact duplicates, check reviewed criteria, and enforce budgets; arbitrary semantic disjointness is not deterministically proven.
   - **EVALUATE**: Execute typed semantic queries in parallel across frozen context.
   - **RESOLVE**: Evaluate deterministic policy over probabilities, evidence classes, and conflict status.
   - **VERIFY**: Check validation obligations and preconditions against bound evidence. Execution postconditions can only be checked after execution.
   - **COMMIT**: Execute authorized operations or trigger fallback escalation.
   - **RECEIPT**: Verify execution postconditions and record the result. The trace accumulates throughout every stage, including failures. The validation-only slice stops at Verdict + EvaluationTrace; its canonical stage order is defined in protocol.md.

## 6. Formal Invariants and Fault Semantics

1. **Decision Is Not Evidence**: A model judgment generated within a slice cannot serve as authoritative evidence for its own verification or downstream justifications within the same transaction.
2. **Addressability and Provenance**: Every factual claim relied upon by a slice must resolve to immutable evidence identity, version, provenance, and checksum, with spans where required by the profile.
3. **Epistemic States vs. Operational States**: System outcomes must not be collapsed into binary success or failure. The system must explicitly distinguish:
   - `validated`: Declared validation obligations pass deterministic verification; execution success requires separate postcondition proof.
   - `rejected`: Admissible evidence actively refutes the proposition.
   - `insufficient`: Evidence is missing or falls below required trust thresholds.
   - `conflict`: Admissible evidence supports incompatible conclusions.
   - `manual_review`: An operational review requirement prevents automated disposition; epistemic findings remain in the report.
   - `error`: A technical or contract failure prevents correct evaluation.
4. **Deterministic Replay Guarantee**: Replay uses identical frozen entity, pack, profiles, QuestionSet, policy, saved raw DecisionSet, approval evidence, and a pinned verifier version. It reproduces deterministic findings and verdict without retrieval, model calls, or side effects. A pinned model alone cannot reproduce probabilistic output; timestamps and trace identifiers need not be identical.
