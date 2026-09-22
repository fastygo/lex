---
name: lex-api
description: Calls the LeX typed-decision API (POST /v1/decisions, /v1/replays, /v1/capabilities), builds State and QuestionSet requests with Noul, Choice, and Score questions, reads DecisionSets and problem responses, and replays bundles. Use when calling LeX from an agent, probing the lexproto deployment, translating a direct Jev call into a LeX request, debugging 400/422 responses, or asking what LeX can and cannot do.
---

# Calling the LeX API

LeX is a typed-decision switch. The caller owns meaning: it writes the State
and the questions, and it decides what to do with the answers. LeX validates,
hashes, pins the model, calls the adapter, checks answer structure, and returns
a replayable bundle. It never returns a domain verdict or performs an action.

## Endpoints

| Route | Use |
|-------|-----|
| `POST /v1/decisions` | canonical: State + QuestionSet -> DecisionSet |
| `POST /v1/replays` | post a returned `replay_bundle` unchanged |
| `GET /v1/capabilities` | primitives, limits, adapter contract, model policy |
| `GET /healthz` | liveness, no auth |
| `POST /v1/evaluations` | deprecated claim validation; do not use for new work |

Every authenticated call needs `Authorization: Bearer <token>`,
`Accept: application/json`, and, with a body, `Content-Type: application/json`.
The token must belong to the request `project_id`.

## Quick start

Use the repository script. It reads the token from `.env` and never prints it.
`LEX_BASE_URL` overrides the default `https://lexproto.vercel.app`.

```bash
node scripts/lex-request.mjs v1/capabilities
node scripts/lex-request.mjs v1/decisions POST .project/.jev/generic/intent-decision-request.json
node scripts/lex-request.mjs v1/replays POST path/to/replay_bundle.json
```

Never print, log, or commit the bearer token or provider keys.

## Request shape

```json
{
  "project_id": "example-project",
  "decision": {"id": "route-step", "version": "1"},
  "state": {"message": "Where is my order 1042?"},
  "question_set": {
    "id": "support.routing",
    "version": "1",
    "questions": {
      "mentions_order": {"type": "noul", "instructions": "Does the message reference a specific order?"},
      "route": {
        "type": "choice",
        "instructions": "Which queue should handle the message?",
        "options": {"shipping": "Delivery status.", "billing": "Payments.", "other": "None of these.", "manual_review": "A human must decide."}
      },
      "urgency": {"type": "score", "instructions": "How urgent is the message?", "levels": ["low", "medium", "high"]}
    }
  },
  "metadata": {"trace_tag": "optional"}
}
```

Rules enforced by the schema (unknown fields are rejected):

- `project_id`, `decision.id`, `question_set.id`: `^[A-Za-z][A-Za-z0-9._:-]{0,255}$`.
- Question ids and option keys: `^[A-Za-z][A-Za-z0-9_-]{0,127}$`.
- 1-64 questions; Choice 2-255 `options`; Score 2-10 unique ordered `levels`.
- `state` is any JSON object and is passed unchanged. `metadata` is at most 16
  string values, is not sent to the adapter, and is not part of the bundle.
- Optional `context` is a frozen Context `pack`, `snapshot`, and `pack_request`
  from the pinned runtime; LeX verifies it and never merges it into `state`.
- Never send a raw Jev `criteria` field. Choice uses `options`; Score uses
  `levels`.

Change `question_set.version` whenever any question or option changes; the
hash changes anyway, and the version keeps replays explainable.

## Question design

- Noul: one independent yes/no predicate. Probabilities across Nouls need not
  sum to one.
- Choice: one mutually exclusive selection. Add `other` for open taxonomies
  and `manual_review` or another non-action option for risky actions.
- Score: only for a genuinely ordered scale.
- One atomic claim per question. Do not use one Choice both to discover
  hypotheses and to authorize an action.
- Put facts in `state` or in a frozen `context`, not in `instructions`.

## Reading a response

HTTP 200 body fields: `structural_status` (`valid`), `decision_set.answers`,
`decision_set.adapter_id`, `adapter_version`, `resolved_model`, `state_hash`, `question_set`,
`findings`, `trace`, `replay_bundle`, optional `adapter_metadata`.

Answer shapes:

```json
{
  "mentions_order": {"type": "noul", "noul": 0.93},
  "route": {"type": "choice", "choice": "shipping", "probabilities": {"shipping": 0.81, "billing": 0.04, "other": 0.1, "manual_review": 0.05}},
  "urgency": {"type": "score", "score": 1.4}
}
```

A Score answer lies in `[0, len(levels) - 1]`. Probabilities are uncalibrated
model output, not authority. The caller applies its own thresholds.

## Errors

Problems use RFC 9457 with a stable `reason`. No request error calls a provider.

| Status and reason | Meaning and fix |
|-------------------|-----------------|
| 400 `invalid_json` | body is not JSON; fix the encoding |
| 422 `question_error` | schema or QuestionSet violation; `detail` names the JSON pointer and keyword |
| 422 `pack_error` | supplied Context binding did not reproduce; see `findings` |
| 422 `decision_error` | provider answers failed structural checks; sealed bundle still returned |
| 422 `response_budget` | State, questions, or answers exceed the body budget |
| 401 / 403 | missing or wrong token, or token lacks `project_id` |
| 406 / 415 | wrong `Accept` or `Content-Type` |
| 502 / 503 / 504 | provider failure or no adapter; not retried by LeX |

## Agent loop

```text
build State + QuestionSet
 -> POST /v1/decisions
 -> 422 question_error: fix the field at the reported pointer, resend
 -> 200: apply caller-owned thresholds to the answers
    -> confident: act through the caller's own tools and approvals
    -> uncertain: rephrase, add evidence to state or context, bump version, resend
    -> risky or unresolved: stop or escalate to a human
 -> keep replay_bundle; POST /v1/replays to prove the recorded decision
```

## What LeX refuses

- choosing questions, thresholds, or next actions for the caller;
- returning a domain verdict from `/v1/decisions`;
- retrieval, crawling, or building a Context pack from raw text in the generic route;
- executing tools, side effects, or issuing receipts;
- model aliases such as `latest`, silent model or provider fallback, or retries;
- server-side history: the caller keeps the bundle.

## References

- Contract: `.project/.lex/generic-decision-api.md`
- Checks and error classes: `.project/.lex/checks.md`
- Capability map: `.project/.jev/capability.md`
- Schemas: `internal/wire/schema/decision-request.schema.json`, `openapi.json`
- Fixtures: `.project/.jev/generic/`
