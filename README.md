# LeX

LeX is subject-neutral operational infrastructure for turning evidence and
uncertain judgments into governed, inspectable, reproducible outcomes.

Its central rule is:

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

The canonical operation is a **typed-decision switch**: an agent sends its own
State and QuestionSet, LeX runs them through a pinned typed-decision adapter,
checks the raw answers structurally, and returns a sealed, replayable
DecisionSet. LeX does not interpret the answers. The agent decides what to do
next: rephrase and ask again, retrieve, call an MCP or web tool, or stop.

LeX combines:

- provider-neutral typed decisions (Noul, Choice, Score) as the judgment plane;
- canonical hashes, exact model pins, and a deterministic structural verifier;
- traces and caller-owned replay bundles;
- an optional frozen **Context Runtime** binding as the evidence plane;
- a deprecated claim-validation compatibility route with policy and verdicts.

Jev is the current reference typed-decision model, reachable through the
direct TypeSafe System One API and OpenRouter. Provider APIs are adapters, not
protocol identities.

## Status

Release: **canary**. External applications can call it today; the contract
is frozen within each envelope version and changes additively
([compatibility promises](.project/.lex/governance.md)).

- Generic envelope `0.2`: `POST /v1/decisions`, generic replay.
  Revision-pinned canary proof is in
  [conformance-report.md](.project/.plan/conformance-report.md).
- Legacy envelope `0.1`: deprecated `POST /v1/evaluations` with the
  embedded `claim-validation` `0.2.0` profile and an explicitly
  `uncalibrated` policy.
- Stable follows after several products use LeX; the remaining gates
  (calibration, SLO window, race evidence, vulnerability review, hosted
  adapter proof) are listed in [progress.md](.project/.plan/progress.md).
  Conformance certification is not claimed.

Running profile:

- Go REST service using Framework and
  [`github.com/fastygo/context@v0.1.0`](.project/.plan/context-version.md);
- embedded Context capability `memory-exact-v1`;
- deployment target: Vercel Go runtime;
- request-scoped RAM only; no database, disk persistence, queue, server-side
  run history, or side-effect execution.

## Architecture

```text
Agent: State + QuestionSet + optional frozen Context binding
  -> LeX: schema + canonical hash + exact model pin
  -> adapter (Jev): raw typed answers
  -> LeX: structural answer verification
  -> LeX: DecisionSet + structural report + trace + replay bundle
  -> Agent: interpret, clarify, retrieve, call a tool, or stop
```

Compatibility route (deprecated):

```text
ValidationIntent + sources or frozen Context state
  -> claim-validation profile + Context Runtime -> frozen ContextPack
  -> adapter -> raw DecisionSet
  -> structural verifier + policy verifier
  -> Verdict + VerificationReport + EvaluationTrace + replay bundle
```

Replay consumes a caller-retained bundle. It never contacts a retrieval service
or a decision provider. When a bundle carries a Context binding, Context
rebuilds the pack from the frozen snapshot and LeX compares it; LeX does not
recompute selection.

Controlled execution and operation receipts are deliberately out of scope. A
decision or verdict never proves that an operation occurred.

## Protocol model

Generic objects:

- `DecisionIdentity` — caller id and version for one decision step;
- `State` — arbitrary caller JSON, hashed and passed unchanged;
- `QuestionSet` — caller-owned id, version, and typed questions;
- optional Context binding — frozen `pack`, `snapshot`, `pack_request`;
- `DecisionSet` — raw typed answers with adapter and resolved model identity;
- structural report — `structural_status` plus findings; never a verdict;
- decision bundle — self-hashed record that replays without a provider.

Compatibility-only objects: `EntityEnvelope`, `ValidationIntent`,
`SemanticProfile`, `PolicySnapshot`, `VerificationReport`, `Verdict`, and
`EvaluationTrace`. Their verdicts stay distinct:

```text
validated | rejected | insufficient | conflict | manual_review | error
```

## Typed questions

- **Noul** — one independent yes/no predicate; answer is a probability of yes.
- **Choice** — one mutually exclusive selection among named `options`.
- **Score** — a position along ordered `levels`; answer is in
  `[0, len(levels) - 1]`.

Keep one atomic claim per question, add `other` to incomplete taxonomies, and
add an explicit non-action option such as `manual_review` for risky actions.
Confidence never grants authority.

## HTTP profile

```text
POST /v1/decisions      canonical generic operation
POST /v1/replays        generic and legacy bundles
GET  /v1/capabilities   primitives, limits, adapter contract, model policy
GET  /healthz
POST /v1/evaluations    deprecated claim-validation compatibility
```

The wire profile uses JSON Schema 2020-12, RFC 8785 JCS with SHA-256, and
RFC 9457 Problem Details. Schemas live in `internal/wire/schema/`, including
the OpenAPI 3.1.1 document (`openapi.json`, version `0.2`).

### Generic decision request

```json
{
  "project_id": "example-project",
  "decision": {"id": "intent-step", "version": "1"},
  "state": {"message": "I need a site to show recent client work."},
  "question_set": {
    "id": "example.intent",
    "version": "1",
    "questions": {
      "intent": {
        "type": "choice",
        "instructions": "Which declared intent best matches the supplied state?",
        "options": {"portfolio": "Show a body of work.", "other": "None of these."}
      },
      "has_purchase_flow": {
        "type": "noul",
        "instructions": "Does the supplied state require a purchase flow?"
      }
    }
  }
}
```

A valid decision is HTTP 200 with `structural_status: "valid"`, `decision_set`,
`state_hash`, trace, and `replay_bundle`. Post the `replay_bundle` unchanged to
`POST /v1/replays`. Structurally invalid provider answers are HTTP 422
`decision_error` and still return the sealed bundle.

Request errors never call a provider. A body that is not JSON is 400
`invalid_json`. Well-formed JSON that violates the schema, such as a raw Jev
`criteria` field instead of `options` or `levels`, is 422 `question_error`
whose `detail` names the failing JSON pointer. See
[generic-decision-api.md](.project/.lex/generic-decision-api.md) for the full
contract and migration from direct Jev calls. Cross-domain fixtures are in
[`.project/.jev/generic/`](.project/.jev/generic/).

### Legacy evaluation request

The deprecated route accepts a `claim` entity, an exact-phrase `query`, and
either versioned `sources` or a caller-frozen Context `context`. The server
supplies the embedded QuestionSet and policy. Responses carry `Deprecation`
and `Link` headers. The 19 scenario bodies are in
[`.project/.jev/test-vercel/requests/`](.project/.jev/test-vercel/requests/).
An empty exact selection is HTTP 200 `insufficient` without a provider call; a
technical `error` verdict is HTTP 422 with the sealed bundle.

## Using LeX from another application

1. Ask the operator for a bearer token bound to your `project_id`.
2. Call `https://lexproto.vercel.app` (or your own deployment) from server-side
   code; never ship the token to a browser.
3. Keep every `replay_bundle` you may need to prove later; LeX stores nothing.

[client-integration.md](.project/.lex/client-integration.md) has curl,
TypeScript, Python, and Go clients. For agents, copy
[`.cursor/skills/lex-api/`](.cursor/skills/lex-api/SKILL.md) into the
application's `.cursor/skills/`; it works without this repository.

## Running locally

Configure bearer tokens and one decision credential outside version control.
Set `LEX_DECISION_ADAPTER` to `direct` or `hosted` when both are present:

```bash
export LEX_BEARER_TOKENS='{"development-token":["example-project"]}'
export LEX_TYPESAFE_API_KEY='replace-with-secret'
go run ./cmd/api
```

The hosted adapter uses `LEX_OPENROUTER_API_KEY` and requires
`LEX_HOSTED_RESOLVED_MODEL`, the exact immutable identity confirmed by the
provider. `latest`, `stable`, and `preview` are rejected, and the same pin
governs replay.

Call a deployment with the token from `.env` without printing it:

```bash
node scripts/lex-request.mjs v1/capabilities
node scripts/lex-request.mjs v1/decisions POST .project/.jev/generic/intent-decision-request.json
```

`LEX_BEARER_TOKENS` and provider credentials are deployment secrets. Never add
them to source files, request payloads, traces, replay bundles, or logs.

## Checks

```bash
python -m pip install -r scripts/requirements-conformance.txt
go test ./... -count=1
go vet ./...
npm run check:english
```

These enforce the canary schemas, OpenAPI document, and local conformance
tests. They are not a conformance certification.

## Documentation

1. [Canonical specification](.project/.lex/README.md)
2. [Generic typed-decision API](.project/.lex/generic-decision-api.md) and
   [client integration](.project/.lex/client-integration.md)
3. [Concept and boundaries](.project/.lex/concept.md)
4. [Protocol entities and lifecycle](.project/.lex/protocol.md)
5. [Checks, verdicts, and errors](.project/.lex/checks.md)
6. [Architecture and capability checklist](.project/.plan/architecture.md)
7. [ADR register](.project/.plan/adr/README.md)
8. [Conformance report](.project/.plan/conformance-report.md)
9. [Jev capability map](.project/.jev/capability.md)

Additional material: [VSA and ICOM guidance](.project/.vsa/README.md),
[Jev research](.project/.jev/README.md) (non-normative), and the
[project documentation map](.project/README.md). When documents disagree,
`.project/.lex/` owns protocol semantics.

## License

Licensed under the [Apache License 2.0](LICENSE).
