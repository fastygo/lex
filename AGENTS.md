# Agent notes

LeX is subject-neutral infrastructure that turns uncertain judgments into
inspectable, reproducible records. It is a **typed-decision switch**: the
caller supplies State and a QuestionSet, LeX returns raw typed answers from a
pinned adapter, a structural report, a trace, and a sealed replay bundle.

Release: **canary**, envelope `0.2`. Compatibility promises are in
`.project/.lex/governance.md`; stable follows after several products use LeX
and the gates in `.project/.plan/progress.md` close. Version identifiers carry
no pre-release suffix; a breaking wire change needs a new envelope version.

## Central principle

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

LeX owns the judgment record and its proof. Evidence comes from the caller or
Context Runtime. Meaning, thresholds, authority, and action belong to the
caller.

## Normative documents

Read in this order; `.project/.lex/` wins on any disagreement.

1. `.project/.lex/README.md` - index.
2. `.project/.lex/protocol.md` - operation, invariants, entities, lifecycle, replay.
3. `.project/.lex/checks.md` - request checks, finding codes, problem reasons, HTTP status.
4. `.project/.lex/scope.md` - built, open, and refused capabilities.
5. `.project/.lex/client-integration.md` - external clients.
6. `.project/.plan/architecture.md` and `decisions.md` - implementation and durable decisions.

Every MUST line in `protocol.md` and `checks.md` is mapped to a test in
`internal/conformance/obligations_test.go`. A new MUST line needs a mapped test.

Agent tooling:

- `.cursor/skills/lex-api/SKILL.md` - calling the API; portable to other apps.
- `.cursor/rules/lex-code-structure.mdc` - package map and how to add features.

`.project/.jev/` is non-normative research. `.project/examples/` holds request
bodies validated by tests.

## HTTP surface

`GET /healthz`, `GET /v1/capabilities`, `POST /v1/decisions`,
`POST /v1/replays`. A valid decision is HTTP 200 `structural_status: valid`.
Structurally invalid answers are HTTP 422 `decision_error` with the sealed
bundle. Request errors never call a provider: non-JSON is 400 `invalid_json`;
a schema or QuestionSet violation is 422 `question_error` whose `detail` names
the JSON pointer and keyword and never echoes caller values. A raw Jev
`criteria` field is refused: Choice uses `options`, Score uses `levels`.
Replay never calls a provider or retrieval service and refuses bundles outside
the principal's projects.

## Non-negotiable invariants

1. State and QuestionSet reach the adapter unchanged.
2. Questions are typed and atomic: Noul, Choice, or Score.
3. LeX derives no domain verdict, applies no threshold, and picks no next action.
4. A decision grants no authority and performs no action.
5. A Context binding is rebuilt by Context before the provider call and is
   never merged into State.
6. Runs pin project, DecisionIdentity, State, QuestionSet, optional Context,
   adapter, resolved model, and verifier versions.
7. A criteria change produces a new QuestionSet hash.
8. Aliases such as `latest` are not model identifiers.
9. Transport normalization never rewrites answers.
10. Credentials never enter responses, traces, bundles, or logs.

## Question design (caller guidance)

- Independent Noul questions for support, conflict, and safety predicates;
  their probabilities need not sum to one.
- One Choice for a mutually exclusive selection; add `other` for open
  taxonomies and a non-action option such as `manual_review` for risky actions.
- Score only for a genuinely ordered scale.
- One atomic claim per question; do not use one Choice both to discover
  hypotheses and to authorize an action.
- Confidence is not authority or calibration.

## Boundaries

- Context Runtime owns retrieval, selection, budgeting, rejection, and pack
  construction. LeX uses the embedded `memory-exact-v1` runtime only to rebuild
  a frozen state and compare; it never retrieves.
- Adapters own provider endpoints, credentials, aliases, and billing. Provider
  names never appear in protocol objects or answer semantics. Every adapter
  preserves raw typed answers, records resolved model and available metadata,
  distinguishes retryable transport failure, never falls back silently, and
  passes the common conformance fixtures.
- Do not add domain profiles, thresholds, verdicts, a question catalogue,
  automatic question generation, retrieval, provider orchestration, server
  history, or an executor. New use cases are caller data.

## Functional contract (ICOM)

- Input: caller State and optional frozen Context.
- Control: QuestionSet, schemas, pins, budgets, project authorization.
- Mechanism: adapter, Context rebuild, structural verifier.
- Output: DecisionSet, structural report, trace, replay bundle.

Mechanisms may change; they cannot weaken Control.

## Failure classes

`question_error`, `pack_error`, `decision_error`, `verification_error`, plus
transport reasons listed in `checks.md`. A retryable provider failure is
`provider_unavailable` and is not retried. Do not explain every pipeline
failure as "the model was wrong".

## Testing and deployment

```bash
go test ./... -count=1
go vet ./...
npm run check:english
```

`scripts/probe-canary.mjs` exercises a deployment with the three examples,
two negative probes, and replay of every bundle; `scripts/lex-request.mjs`
sends one authenticated request with the token from `.env` and never prints
it. Record revision-pinned outcomes and the rollback point in
`.project/.plan/conformance-report.md`.

## Repository rules

- Code comments and repository text are English (`.manual/` excluded).
- Keep credentials, API keys, and personal data out of the repository.
- Update `.project/.lex/` when behavior changes; examples never define law.
