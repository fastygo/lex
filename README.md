# LeX

LeX is subject-neutral infrastructure that turns uncertain judgments into
inspectable, reproducible records.

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

LeX is a **typed-decision switch**. An agent sends its own State and
QuestionSet; LeX runs them through a pinned typed-decision adapter, checks the
raw answers structurally, and returns a sealed, replayable DecisionSet. LeX
does not interpret the answers. The agent decides what happens next: ask
again, fetch context, call a tool, escalate, or stop.

Jev is the reference decision model, reachable through the direct TypeSafe
System One API and through OpenRouter. Providers are adapters, not protocol
identities.

## Status

Release: **canary**, envelope `0.2`. Fields, meanings, reason codes, and
status mapping are fixed within the envelope and change only additively
([governance](.project/.lex/governance.md)). The revision-pinned canary and
its rollback point are in the
[conformance report](.project/.plan/conformance-report.md). Stable follows
after several products use LeX and the gates in
[progress.md](.project/.plan/progress.md) close.

Running profile: Go REST service on Framework `v0.3.0` and Context `v0.1.0`
(embedded runtime `memory-exact-v1`), deployed on Vercel, request RAM only; no
database, disk, queue, server history, or side-effect execution.

## How it works

```text
agent: project_id + decision + state + question_set + optional frozen context
  -> LeX: schema, canonical hashes, exact model pin, optional Context rebuild
  -> adapter (Jev): raw typed answers
  -> LeX: structural verification
  -> LeX: DecisionSet + structural report + trace + replay bundle
  -> agent: interpret and choose the next step
```

Questions are typed:

- **Noul**: one independent yes/no predicate; the answer is a probability of yes.
- **Choice**: one mutually exclusive selection among named `options`.
- **Score**: a position along ordered `levels`, in `[0, len(levels) - 1]`.

Keep one atomic claim per question, add `other` to open taxonomies, and add a
non-action option such as `manual_review` to risky choices. Probabilities are
uncalibrated and never grant authority.

## HTTP

```text
POST /v1/decisions      State + QuestionSet -> DecisionSet + replay bundle
POST /v1/replays        recheck a returned bundle without calling a provider
GET  /v1/capabilities   primitives, limits, adapter contract, model policy
GET  /healthz
```

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
      "has_purchase_flow": {"type": "noul", "instructions": "Does the supplied state require a purchase flow?"}
    }
  }
}
```

A valid decision is HTTP 200 with `structural_status: "valid"`. Post the
`replay_bundle` unchanged to `/v1/replays`. Request errors never reach a
provider: non-JSON is 400 `invalid_json`; a schema or QuestionSet violation is
422 `question_error` with the failing JSON pointer. Structurally invalid
provider answers are 422 `decision_error` and still return the sealed bundle.
Full mapping: [checks.md](.project/.lex/checks.md). Examples:
[`.project/examples/`](.project/examples/).

The wire profile is JSON Schema 2020-12, RFC 8785 JCS with SHA-256, and
RFC 9457 problems; schemas and OpenAPI 3.1.1 live in `internal/wire/schema/`.

## Using LeX from another application

1. Get a bearer token bound to your `project_id` from the operator.
2. Call `https://lexproto.vercel.app` (or your deployment) from server-side
   code only; CORS is refused.
3. Keep every `replay_bundle` you may need to prove later; LeX stores nothing.

[client-integration.md](.project/.lex/client-integration.md) has curl,
TypeScript, Python, and Go clients. For agents, copy
[`.cursor/skills/lex-api/`](.cursor/skills/lex-api/SKILL.md) into the
application's `.cursor/skills/`.

## Running locally

```bash
export LEX_BEARER_TOKENS='{"development-token":["example-project"]}'
export LEX_TYPESAFE_API_KEY='replace-with-secret'
go run ./cmd/api
```

The hosted adapter uses `LEX_OPENROUTER_API_KEY` and requires
`LEX_HOSTED_RESOLVED_MODEL`, an exact immutable model identity; aliases are
refused. Set `LEX_DECISION_ADAPTER` to `direct` or `hosted` when both keys are
present. Secrets never go into source, requests, traces, bundles, or logs.

Call a deployment with the token from `.env` without printing it:

```bash
node scripts/lex-request.mjs v1/capabilities
node scripts/lex-request.mjs v1/decisions POST .project/examples/intent-decision-request.json
LEX_REVISION=<sha> LEX_DEPLOYMENT_ID=<id> node scripts/probe-canary.mjs
```

## Checks

```bash
python -m pip install -r scripts/requirements-conformance.txt
go test ./... -count=1
go vet ./...
npm run check:english
```

These enforce the schemas, OpenAPI, and the mapping from every MUST line to a
test. They are not a conformance certification.

## Documentation

- [Specification](.project/.lex/README.md): concept, protocol, checks, scope,
  client guide, governance.
- [Plan](.project/.plan/README.md): architecture, decisions, progress,
  conformance report.
- [Jev research](.project/.jev/README.md) (non-normative).

## License

Licensed under the [Apache License 2.0](LICENSE).
