# Agent notes

LeX is subject-neutral operational infrastructure for turning evidence and
uncertain judgments into governed, inspectable, reproducible outcomes.

The protocol is one part of LeX. The complete system includes schemas, evidence
contracts, semantic profiles, typed judgments, policy, deterministic
verification, controlled execution, traces, receipts, and conformance.

## Central principle

No uncertain interpretation becomes an authorized operation without explicit
evidence, policy, and verification.

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

## Canonical architecture

```text
Context Runtime     evidence plane
Typed decision      judgment plane
LeX verifier        policy binding + deterministic gates
Executor            authorized operation only
Trace/receipt       replay + proof of postconditions
```

Current reference decision model: **Jev**. Current access paths include the
direct TypeSafe System One API and OpenRouter's Jev/System One APIs. These are
provider adapters, not protocol identities. More providers may be added without
changing LeX core semantics.

LeX binds to a provider-neutral contract:

```text
Frozen ContextPack + QuestionSet
  -> typed Noul / Choice / Score answers
  -> raw DecisionSet with resolved model and provider metadata
  -> deterministic verification
```

Keep provider endpoints, credentials, model aliases, SDK types, retries, and
billing fields inside adapters. Never put TypeSafe, OpenRouter, or future
provider names into core protocol object names or verdict semantics.

## Normative documents

Read in this order:

1. `.project/.lex/README.md` - canonical index and core path.
2. `.project/.lex/concept.md` - subject-neutral concept and boundaries.
3. `.project/.lex/protocol.md` - canonical entities, lifecycle, invariants.
4. `.project/.lex/checks.md` - checks, verdicts, and error classes.
5. `.project/.lex/integration-stack.md` - Context and decision integration.
6. `.project/.lex/scope.md` - complete in/out boundary.
7. `.project/.lex/analysis.md` - v0.1 priorities and deliberate deferrals.
8. `.project/.vsa/README.md` - vertical slices and ICOM control contracts.

`.project/.jev/` is research evidence, not normative protocol text:

- `research-findings.md` records observed behavior and limitations.
- `examples/` contains hand-authored fixtures and captured responses.
- `lex-protocol-draft.md` is a historical extended draft and may lag
  `.project/.lex/protocol.md`.

When documents disagree, `.project/.lex/` wins. Do not create another complete
copy of the protocol.

## Non-negotiable invariants

1. A decision is not evidence for its own verdict.
2. Evidence is addressable: identity, provenance, version, and checksum.
3. Questions are typed and atomic.
4. Support, establishment, authority, action, and verified success are distinct.
5. `insufficient` and `conflict` are valid outcomes, not errors to hide.
6. Policy and thresholds are external to the judgment model.
7. `validated` requires deterministic verification.
8. Runs pin entity, ContextPack, QuestionSet, policy, adapter, and resolved
   model versions.
9. Criteria changes produce a new QuestionSet version/hash.
10. Aliases such as `latest` are not reproducible model identifiers.

Do not collapse these verdicts:

```text
validated | rejected | insufficient | conflict | manual_review | error
```

`insufficient` and `conflict` are epistemic. `manual_review` is operational.

## Question design

- Use independent **Noul** questions for support, establishment, conflict, and
  safety predicates. Their probabilities do not need to sum to one.
- Use one **Choice** for a mutually exclusive operational action.
- Use **Score** only for a genuinely ordered rubric.
- Keep one atomic claim per question.
- Add `other` for incomplete taxonomies.
- Add an explicit non-action path such as `manual_review` for risky actions.
- Do not use one Choice both to discover supported hypotheses and authorize an
  action.
- Do not treat confidence as authority or domain calibration.

Preferred fan-out:

```text
support Noul x N
established/conflict Noul
safe_to_auto_act Noul
one action Choice
optional ordered Score
```

## VSA and functional contracts

Build LeX behavior as vertical slices. Each slice owns one transaction boundary
and its proof artifacts.

Use ICOM to describe the functional contract:

- **Input**: entity and evidence being transformed.
- **Control**: schema, policy, risk, and Done-iff conditions.
- **Mechanism**: Context Runtime, Jev adapter, tools, verifier.
- **Output**: verdict, authorized mutation, report, trace, or receipt.

Mechanisms may choose internal paths, but they cannot weaken Control.

Enforce three boundary tiers:

1. **Deterministic** - schema, checksum, budget, spans, allowlists.
2. **Semantic** - typed probabilistic judgments over frozen evidence.
3. **Authoritative** - credentials, explicit approvals, human authority.

SADT/ICOM describes what makes a transformation lawful. A deterministic state
machine owns retries, timeouts, cancellation, compensation, and temporal flow.

## Provider adapter rules

Direct TypeSafe and OpenRouter may expose compatible Jev question/answer
semantics through different endpoints and metadata. Adapters must normalize
transport details into the same LeX `DecisionSet`.

Every adapter must:

- accept a frozen state and exact QuestionSet;
- preserve raw typed answers without semantic rewriting;
- record provider, request id, resolved model id, timing, and usage when
  available;
- distinguish retryable transport failure from decision output;
- never silently fall back to a different model or provider;
- expose capability differences before execution;
- pass common conformance fixtures.

Provider-specific response additions remain trace metadata. They must not alter
verdict rules. If a provider cannot preserve Noul/Choice/Score semantics, it is
not a conforming typed-decision adapter.

## Implementation priorities

Keep LeX thin. Context Runtime owns retrieval and ContextPack construction.
Decision adapters own provider transport. LeX owns:

- protocol schemas and hashes;
- semantic profiles and QuestionSets;
- preflight checks;
- deterministic verifier and policy gates;
- verdict state machine;
- EvaluationTrace and Receipt;
- conformance fixtures.

Recommended v0.1 slice:

```text
ValidationIntent
  -> FocusProfile
  -> real ContextPack through Context API
  -> QuestionSet
  -> provider-neutral DecisionSet
  -> verifier
  -> Verdict + trace
```

Do not add a universal ontology, automatic question generation, a new retrieval
engine, or provider orchestration before this slice is proven.

## Testing and proof

Hand-authored `.project/.jev/examples/` prove contract shape only. They do not
prove retrieval quality, calibration, or production safety.

Required progression:

1. Real ContextPack from Context Runtime HTTP API or `contextkit`.
2. Versioned JSON schemas and canonical hashes.
3. Raw DecisionSet persistence.
4. Deterministic verifier reference implementation.
5. Golden and invalid protocol fixtures.
6. Adversarial cases: inference-only evidence, conflicts, missing evidence,
   overlapping criteria, compound questions, and provider failures.
7. Calibration report per entity type, model version, provider path, and policy.
8. Replay with pinned inputs and resolved versions.
9. Interoperability test across at least two decision-provider adapters.

Classify failures by stage:

```text
retrieval_error | pack_error | question_error | decision_error
policy_error | verification_error | execution_error
```

Do not explain every pipeline failure as "the model was wrong."

## Repository rules

- Human-readable repository content is English only outside `.manual/`.
- Code comments must be English.
- Keep credentials, API keys, personal data, and provider secrets out of the
  repository and traces.
- Keep `.project/.lex/` normative and `.project/.jev/` experimental.
- Update canonical docs when behavior changes; do not let examples define law.
