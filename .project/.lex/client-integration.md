# Client integration

Status: canary release, generic envelope `0.2`. This guide is for external
applications and agents that call LeX over HTTP. The contract itself is
[protocol.md](protocol.md) and [checks.md](checks.md); compatibility promises
are in [governance.md](governance.md). Request examples are in
`.project/examples/`.

## What a client gets

A client sends its own State and QuestionSet and receives raw typed answers
with the resolved model, hashes, a trace, and a sealed replay bundle. LeX
does not choose questions, interpret answers, apply thresholds, or perform
actions. The client owns meaning and the next step.

## Access

1. The operator issues a bearer token bound to one or more `project_id`
   values. Tokens live in the deployment secret `LEX_BEARER_TOKENS` as
   `{"<token>":["<project_id>", ...]}`; adding a client is a secret update
   and a redeploy, not a code change.
2. The client stores the token as its own secret, for example `LEX_TOKEN`,
   and the base URL as `LEX_BASE_URL` (canary: `https://lexproto.vercel.app`).
3. Call LeX from server-side code only. CORS is denied by design, and a
   token in a browser is a leaked token.

Every call sends `Authorization: Bearer $LEX_TOKEN` and
`Accept: application/json`; calls with a body also send
`Content-Type: application/json`. `GET /healthz` needs no token.

## Discover before calling

`GET /v1/capabilities` returns `protocol_status` (`canary`),
`operations.decision`, and `typed_decision` with primitives and limits:
64 questions, 255 Choice options, 10 Score levels, a 2 MiB body, and a 20 s
request deadline. Check `operations.decision` is `true` at startup.

## Minimal clients

curl:

```bash
curl -sS "$LEX_BASE_URL/v1/decisions" \
  -H "Authorization: Bearer $LEX_TOKEN" \
  -H "Accept: application/json" -H "Content-Type: application/json" \
  --data @decision.json
```

TypeScript (Node 18+ or any runtime with `fetch`):

```ts
export async function decide(body: unknown) {
  const response = await fetch(`${process.env.LEX_BASE_URL}/v1/decisions`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${process.env.LEX_TOKEN}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  const payload = await response.json();
  if (!response.ok) throw Object.assign(new Error(payload.detail), { problem: payload });
  return payload; // payload.decision_set.answers, payload.replay_bundle
}
```

Python (standard library):

```python
import json, os, urllib.request, urllib.error

def decide(body: dict) -> dict:
    request = urllib.request.Request(
        os.environ["LEX_BASE_URL"] + "/v1/decisions",
        data=json.dumps(body).encode(),
        method="POST",
        headers={
            "Authorization": "Bearer " + os.environ["LEX_TOKEN"],
            "Accept": "application/json",
            "Content-Type": "application/json",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        raise RuntimeError(json.load(error)) from None
```

Go:

```go
func Decide(ctx context.Context, body any) (map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, os.Getenv("LEX_BASE_URL")+"/v1/decisions", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LEX_TOKEN"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return payload, fmt.Errorf("lex %d %v: %v", resp.StatusCode, payload["reason"], payload["detail"])
	}
	return payload, nil
}
```

## Request and answers

A request carries `project_id`, `decision` (`id`, `version`), `state` (any
JSON object), and `question_set` (`id`, `version`, `questions`). Questions
are `noul` (`instructions`), `choice` (`instructions`, `options` map), or
`score` (`instructions`, ordered `levels`). Optional `context` binds a frozen
Context state; optional `metadata` is never sent to the adapter or sealed.

Answers in `decision_set.answers`:

```json
{
  "needs_human": {"type": "noul", "noul": 0.12},
  "route": {"type": "choice", "choice": "billing", "probabilities": {"billing": 0.8, "other": 0.2}},
  "urgency": {"type": "score", "score": 1.4}
}
```

A Score lies in `[0, len(levels) - 1]`. Probabilities are uncalibrated;
apply your own thresholds and keep a non-action path for risky steps.

## Errors and retries

Problems are RFC 9457 JSON with a stable `reason`:

- 400 `invalid_json`: the body is not JSON. Fix the encoding.
- 422 `question_error`: the body violates the request schema or QuestionSet
  rules; `detail` names the JSON pointer. Fix and resend. No provider call.
- 422 `pack_error`: the Context binding did not reproduce.
- 422 `decision_error`: the provider's answers failed structural checks; the
  sealed bundle is still returned.
- 422 `response_budget`: State, questions, or answers do not fit the body limit.
- 401/403: token missing, wrong, or not bound to `project_id`.
- 503 `admission_limited` or `provider_unavailable`, 502, 504: transient.
  LeX never retries. A client may retry with backoff; each retry is a new
  provider call and a new decision.

## Retention and replay

LeX stores nothing. Keep `replay_bundle` for any decision you may need to
prove. `POST /v1/replays` with the bundle as the body returns
`decision_reproduced` and the structural report without calling the
provider. Replay proves the recorded decision, not current authority.

## Compatibility

Within envelope `0.2`, fields, meanings, reason codes, and status mapping
do not change; new fields may appear, so ignore unknown response fields.
Check `protocol_version` in responses is `0.2`. A breaking change ships as a
new envelope version, announced in capabilities, with the previous verifier
kept for replay.

## Migrating a direct Jev call

Keep the Jev state as `state`, add a stable `project_id` and `decision`, and
wrap the question map in a versioned `question_set`. Copy a Choice `criteria`
object to `options` and a Score `criteria` array to `levels`; a Noul keeps
`type` and `instructions`. Do not send `model`, credentials, endpoints, or
retries: the deployment owns the adapter pin. Move client correlation values
to `metadata`.

## Agents

Copy [`.cursor/skills/lex-api/`](../../.cursor/skills/lex-api/SKILL.md) into
the application's `.cursor/skills/`. It needs only `LEX_BASE_URL` and
`LEX_TOKEN` and does not depend on this repository.
