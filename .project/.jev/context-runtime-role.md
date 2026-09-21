# Context Runtime’s role for Jev

Status: non-normative research integration sketch, not a claim of shipped upstream behavior.
The current [integration contract](../.lex/integration-stack.md) takes precedence.
Shared spans are only overlap diagnostics; policy mismatch, missing evidence,
and low trust are distinct from contradictory admissible evidence.
Model-reported safety never grants authority.

## Position in the architecture

Context Runtime is not a prompt wrapper for Jev and not a competitor to Jev. It is the **evidence plane** of the LeX protocol. Jev is the **decision plane**. LeX defines the contract between them and the rules for the final verdict.

```text
Context Runtime
  evidence selection + provenance + policy
        ↓ ContextPack
Jev
  typed probabilistic judgments
        ↓ DecisionSet
LeX verifier
  evidence binding + thresholds + risk policy
        ↓ Verdict + Trace
```

Without Context Runtime, Jev gets unbounded state where facts, instructions, hypotheses, and noise mix. Without Jev, Context Runtime can find and pack evidence but does not perform semantic probabilistic judgments. Without LeX, the two components share no validation contract.

## Context Runtime responsibilities

### 1. Entity and project isolation

Each entity under validation belongs to a project and has stable identity, version, and checksum. Context Runtime must not let evidence from another project enter validation unnoticed.

### 2. Evidence retrieval

Context Runtime performs:

- exact retrieval for identifiers, quotes, and known values;
- sparse retrieval for terms;
- dense retrieval for semantic similarity;
- morphology-aware retrieval for word forms;
- merge, deduplication, and ranking;
- filtering by trust, evidence class, policy, and FocusProfile.

Jev must not search for facts across an unbounded project on its own. It evaluates a pre-selected state.

### 3. Building a narrow ContextPack

`ContextPack` should contain only evidence needed for a specific validation intent. One pack need not answer every question in the project.

The pack separates:

- `instructions[]`;
- `policy_refs[]`;
- `evidence_items[]`;
- `rejected_items[]`;
- `verification_requirements[]`.

That separation prevents data from posing as instructions and model-generated hypotheses from posing as source truth.

### 4. Provenance and evidence classes

Each significant fragment keeps:

- `source_id`;
- byte spans;
- checksum;
- snapshot/version;
- trust level;
- evidence class;
- lineage for derived artifacts.

For factual validation, prefer:

- `source_text`;
- `attestation`;
- explicitly authoritative `tool_output`.

`model_inference`, `lexical_analysis`, and `concept_mapping` may guide search or interpretation but do not establish facts by themselves.

### 5. Preparing Choice criteria

Context Runtime and lexicon adapters can help build criteria from:

- `Sense.definition`;
- `Concept.preferred_label`;
- broader/narrower relations;
- attestations;
- controlled vocabularies;
- project-specific terminology.

Before calling Jev, the consumer should check:

- definitions are distinguishable;
- shared spans are reviewed as diagnostics, without assuming they make options invalid;
- taxonomy covers expected cases;
- `other` or `manual_review` exist when coverage is incomplete;
- Choice does not replace independent multi-label diagnostics.

Core Context Runtime provides evidence contracts; the LeX consumer compiles the concrete taxonomy into Jev questions.

### 6. Detecting conflicts before action

Context Runtime must preserve contradicting evidence, not silently pick a convenient source. Conflict may appear via:

- incompatible authoritative tool outputs;
- different versions or snapshots;
- identical spans supporting multiple criteria;
- disagreement between policy and source;
- insufficient trust;
- missing required evidence class.

Jev may assess conflict, but the LeX verifier should check structural conflicts deterministically when possible.

### 7. Verification and replay

After Jev, Context Runtime supplies material to verify:

- the decision is actually supported by cited spans;
- an allowed snapshot was used;
- required trust was met;
- the verdict is not based only on `model_inference`;
- policy and approval requirements were satisfied;
- the run can be replayed.

High probability does not replace this check.

## What Context Runtime must not do

- Treat model summary as source truth.
- Hide rejected or conflicting evidence.
- Turn retrieval score into claim truth probability.
- Choose business action from passage similarity alone.
- Treat one winning Choice as proof that criteria are mutually exclusive.
- Set universal thresholds instead of LeX domain policy.
- Allow destructive action from Jev confidence alone.

## Recommended Jev integration

Connect Jev as a replaceable typed decision adapter after `ContextPack` is built:

```text
ValidationIntent
  → FocusProfile
  → RetrievalPlan
  → ContextPack
  → LeX QuestionSet
  → Jev DecisionSet
  → deterministic evidence binding
  → policy/risk gate
  → Verdict
```

Different tasks may use different adapter roles:

- classifier over a narrow pack;
- semantic predicate evaluator;
- evidence sufficiency scorer;
- reranker when the result remains only a ranking signal;
- action recommender before mandatory verifier.

Jev must not be:

- the context store;
- the provenance source;
- the sole factual verifier;
- the prose completer;
- the sole authorization gate.

## Practical effect on accuracy

Context Runtime improves accuracy not by changing Jev’s weights but by improving the task:

1. removes irrelevant state;
2. separates facts from inference;
3. supplies definitions and attestations for criteria;
4. preserves contradictions;
5. scopes questions to one FocusProfile;
6. enables post-model verification.

The goal is not maximum Jev confidence but minimum wrong automatic actions while still honestly preserving uncertainty as `insufficient` or `conflict`, with `manual_review` as a distinct operational disposition.
