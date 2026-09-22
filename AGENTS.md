# Agent notes

LeX is subject-neutral operational infrastructure for turning evidence and
uncertain judgments into governed, inspectable, reproducible outcomes.

The protocol is one part of LeX. The concept includes schemas, optional
evidence contracts, typed judgments, compatibility policy, deterministic
verification, traces, and conformance. The current slice makes generic typed
decisions and replays them. Controlled execution and operation receipts are
deferred.

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
Context Runtime     optional evidence plane        current
Typed decision      judgment plane                 current
LeX verifier        structural verification        current
Trace + replay      caller-owned bundle            current
Executor            authorized operation           deferred
Receipt             postcondition proof            deferred
```

Current reference decision model: **Jev**. Current access paths include the
direct TypeSafe System One API and OpenRouter's Jev/System One APIs. These are
provider adapters, not protocol identities. More providers may be added without
changing LeX core semantics.

LeX binds to a provider-neutral contract:

```text
Caller State + QuestionSet + optional frozen Context binding
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
8. Generic runs pin project, DecisionIdentity, State, QuestionSet, optional
   Context binding, adapter, and resolved model versions. Compatibility claim
   evaluations additionally pin entity, ContextPack, policy, and profile.
9. Criteria changes produce a new QuestionSet version/hash.
10. Aliases such as `latest` are not reproducible model identifiers.

Do not collapse these verdicts:

```text
validated | rejected | insufficient | conflict | manual_review | error
```

`insufficient` and `conflict` are epistemic. `manual_review` is operational.

## Question design

- Use independent **Noul** questions for support, establishment, refutation,
  conflict, and safety predicates. Their probabilities do not need to sum to one.
- Use one **Choice** for a mutually exclusive operational action.
- Use **Score** only for a genuinely ordered rubric.
- Keep one atomic claim per question.
- Add `other` for incomplete taxonomies.
- Add an explicit non-action path such as `manual_review` for risky actions.
- Do not use one Choice both to discover supported hypotheses and authorize an
  action.
- Do not treat confidence as authority or domain calibration.

Preferred fan-out for the claim-validation compatibility profile:

```text
support Noul x N
established/conflict Noul
safe_to_auto_act Noul
one action Choice
optional ordered Score
```

The embedded `claim-validation` profile `0.2.0` asks `support`, `established`,
`refuted`, `conflict`, `safe_to_auto_act`, and one action Choice. `refuted`
means the evidence establishes that the claim is false. Coherent refutation is
not conflict. Score remains a conforming adapter type and is not part of this
profile. The policy is explicitly `uncalibrated`.

## VSA and functional contracts

Build LeX behavior as vertical slices. Each slice owns one transaction boundary
and its proof artifacts.

Use ICOM to describe the functional contract:

- **Input**: entity and evidence being transformed.
- **Control**: schema, policy, risk, and Done-iff conditions.
- **Mechanism**: Context Runtime, Jev adapter, tools, verifier.
- **Output**: verdict, authorized mutation, report, trace, or receipt.

The generic current slice returns a DecisionSet, structural report, trace, and
replay bundle. The caller interprets it and chooses any next action. The
deprecated claim-validation compatibility route additionally returns a Verdict.
Neither route authorizes a mutation or emits a receipt. Mechanisms may choose
internal paths, but they cannot weaken Control.

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

- accept exact caller State and QuestionSet, plus an optional frozen Context
  binding;
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

## Current slice

Keep LeX thin. Context Runtime owns retrieval, selection, budgeting, rejection,
and ContextPack construction. Generic decisions do not invoke it automatically:
they optionally verify a frozen Context binding supplied by the caller. The
service uses the embedded `memory-exact-v1` runtime. It does not call the
Context HTTP API, it does not contain a second retrieval engine, and it does
not recompute Context's selection: the verifier asks Context to rebuild a
supplied frozen state and compares. Decision adapters own provider transport.
LeX owns:

- protocol schemas and hashes;
- generic State, QuestionSet, DecisionSet, hashes, pins, structural verifier,
  lifecycle trace, and typed replay bundle;
- optional Context binding checks: project/runtime identity, hashes, and
  reproduction;
- legacy compatibility profiles (`internal/profile/claimvalidation`), evidence
  source freezing, policy gate, Verdict, and legacy replay bundle only behind
  `/v1/evaluations`.

The implemented path is:

```text
Generic: State + QuestionSet + optional frozen Context
  -> provider-neutral DecisionSet
  -> structural verifier
  -> structural report + trace + replay bundle

Compatibility: ValidationIntent
  -> claim-validation profile + frozen Context state
  -> provider-neutral DecisionSet
  -> policy verifier
  -> Verdict + trace + legacy replay bundle
```

The HTTP surface is `GET /healthz`, `GET /v1/capabilities`,
`POST /v1/decisions`, deprecated `POST /v1/evaluations`, and
`POST /v1/replays`. A generic valid decision is HTTP 200
`structural_status: valid`; structural answer failure is HTTP 422 with its
sealed generic bundle. Non-error legacy Verdicts are HTTP 200. A technical
legacy `error` verdict is HTTP 422 and still returns the sealed bundle. An
empty legacy exact selection is HTTP 200 `insufficient`, does not call a
provider, and still returns a replay bundle. Replay refuses a bundle outside
the authenticated principal's projects. The generic wire envelope is
`0.2-draft`; the legacy envelope remains `0.1-draft`.

Do not add a universal ontology, automatic question generation, a new retrieval
engine, provider orchestration, server-side history, or an executor inside this
slice.

## Testing and proof

`.project/.jev/examples/` holds research question and response maps. They are
not evaluation requests and do not prove retrieval quality, calibration, or
production safety. Protocol requests for those scenarios live in
`.project/.jev/test-vercel/requests/` and go through legacy
`POST /v1/evaluations`. Generic cross-domain request fixtures live in
`.project/.jev/generic/` and go through `POST /v1/decisions`.

Local tests already cover a real embedded ContextPack, versioned schemas and
canonical hashes, raw answers retained only in the response bundle, the
verifier, golden and invalid fixtures, adversarial evidence and provider
failures, network-free replay, and direct plus hosted adapter fixtures.

Still open: a calibration report, race evidence on a gcc-capable runner, a
28-day SLO, hosted-adapter proof on a deployment, and proof of the latest
deployment revision. No conformance certification is claimed.

Classify failures by stage. The HTTP mapping is in `.project/.lex/checks.md`:

```text
retrieval_error | pack_error | question_error | decision_error
policy_error | verification_error | execution_error
```

A policy denial is a finding, not `policy_error`. A retryable provider failure
is `provider_unavailable` and is not retried. This slice has no execution
route, so it does not emit `execution_error`. Do not explain every pipeline
failure as "the model was wrong."

## Repository rules

- Human-readable repository content is English only outside `.manual/`.
- Code comments must be English.
- Keep credentials, API keys, personal data, and provider secrets out of the
  repository and traces.
- Keep `.project/.lex/` normative and `.project/.jev/` experimental.
- Update canonical docs when behavior changes; do not let examples define law.
