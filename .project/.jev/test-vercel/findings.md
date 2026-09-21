# LeX protocol run on Vercel

Status: research observation against `https://lexproto.vercel.app` at
`2026-09-21T22:14:57Z`. No credentials are recorded. This run does not call
TypeSafe or OpenRouter directly. The deployment adapter already holds the
decision credential.

The live profile was `claim-validation` `0.2.0`, calibration `uncalibrated`,
retrieval `exact_phrase`, runtime `memory-exact-v1`. The wire envelope was
`0.1-draft`. The resolved model was `jev-1.13.0` and the adapter was
`direct-systemone` `0.1.0`.

Request bodies are in [requests/](requests/). Compact verdicts are in
[evaluations/](evaluations/). The index is [results.json](results.json).
The JSON maps under [../examples/](../examples/) were not sent as evaluations,
except one negative check of `playground/questions-jev.json`.

## Formulation

Each original example question becomes one LeX `EvaluationRequest`:

```text
POST /v1/evaluations
Authorization: Bearer <deployment token>
Content-Type: application/json

{
  "project_id": "example-project",
  "entity": {
    "id": "<claim-id>",
    "type": "claim",
    "schema_version": "0.1",
    "version": "1"
  },
  "query": "<exact claim phrase>",
  "sources": [{"id": "<source-id>", "version": "v1", "text": "<case or note>"}],
  "metadata": {"example": "<playground|llm|context|chaos>", "client_ref": "<claim-id>"}
}
```

The caller does not send the question set. The deployment asks `support`,
`established`, `refuted`, `conflict`, `safe_to_auto_act`, and `action`.
The claim is the `query` phrase. Exact retrieval keeps a source only when that
phrase appears in its text. Chaos and context cases are one source each, so
the model sees the full narrative.

## Live verdicts

All 18 evaluations returned HTTP 200. Each sealed bundle replayed as HTTP 200
`verdict_reproduced` with the same verdict. `playground/questions-jev.json`
returned HTTP 400 `invalid_json`.

| Request | Query | support | established | refuted | conflict | safe | action | Verdict |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- | --- |
| playground-structured-decisions | structured decisions | 0.92 | 0.59 | 0.03 | 0.05 | 0.48 | proceed | insufficient |
| playground-cannot-generate-strings | cannot generate string responses | 0.97 | 0.88 | 0.06 | 0.04 | 0.68 | proceed | manual_review |
| playground-parallel-questions | evaluated independently and in parallel | 0.98 | 0.92 | 0.03 | 0.04 | 0.77 | proceed | manual_review |
| playground-noul-no-confidence | no separate confidence field | 0.95 | 0.81 | 0.22 | 0.12 | 0.55 | proceed | manual_review |
| llm-next-token | next token | 0.85 | 0.59 | 0.03 | 0.04 | 0.62 | proceed | insufficient |
| llm-temperature-zero | Temperature 0 | 0.86 | 0.59 | 0.10 | 0.07 | 0.47 | manual_review | insufficient |
| llm-proposal-not-authority | treat the model output as a proposal | 0.98 | 0.92 | 0.02 | 0.04 | 0.51 | manual_review | manual_review |
| llm-hallucination-structural | Hallucination is structural | 0.98 | 0.83 | 0.02 | 0.04 | 0.57 | proceed | manual_review |
| context-restore-access | restore access | 0.96 | 0.89 | 0.03 | 0.10 | 0.60 | proceed | manual_review |
| context-refund | refund | 0.32 | 0.04 | 0.47 | 0.23 | 0.13 | reject | insufficient |
| context-account-access | account access | 0.90 | 0.93 | 0.03 | 0.14 | 0.62 | proceed | manual_review |
| context-duplicate-charge | duplicate charge | 0.12 | 0.04 | 0.90 | 0.20 | 0.23 | reject | rejected |
| chaos-raw-sign-in | cannot sign in | 0.98 | 0.90 | 0.03 | 0.04 | 0.67 | proceed | manual_review |
| chaos-llm-augmented-sign-in | cannot sign in | 0.96 | 0.79 | 0.03 | 0.14 | 0.32 | manual_review | insufficient |
| chaos-resolved-sign-in | cannot sign in | 0.97 | 0.90 | 0.04 | 0.11 | 0.58 | proceed | manual_review |
| chaos-resolved-account-access | account access | 0.74 | 0.79 | 0.07 | 0.10 | 0.52 | proceed | insufficient |
| chaos-conflicted-sign-in | cannot sign in | 0.97 | 0.87 | 0.04 | 0.17 | 0.31 | manual_review | manual_review |
| chaos-conflicted-duplicate-charge | duplicate charge | 0.91 | 0.42 | 0.06 | 0.69 | 0.10 | manual_review | conflict |

## Observations

### 1. The protocol path is the evaluation route

The deployment accepted the 18 LeX evaluation bodies and asked the embedded
`0.2.0` question set. Posting the raw Jev question map
`playground/questions-jev.json` was HTTP 400 `invalid_json`.

### 2. A proceed choice did not validate any claim in this run

`evaluated independently and in parallel` had support 0.98 and establishment
0.92, with safety 0.77. The verdict was `manual_review` on `safety_gate`.
No request in this sample cleared the safety gate of 0.8, so none returned
`validated`.

### 3. Refutation is separate from conflict and from a reject choice

`duplicate charge` in the context scenario had refuted 0.90, conflict 0.20,
and action `reject`. The verdict was `rejected` with finding `negative_result`.
`refund` also chose `reject`, but refuted was 0.47. The verdict stayed
`insufficient`. The report kept `action_inconsistent` beside the support and
establishment findings; that review finding did not outrank `insufficient`.

### 4. Conflict still outranks a review action

`chaos-conflicted-duplicate-charge` had conflict 0.69 and action
`manual_review`. The verdict was `conflict`. The findings kept both
`evidence_conflict` and `review_required`.

### 5. The same note still supports different atomic claims

On the playground note, `evaluated independently and in parallel` was
established and blocked only by safety. `structured decisions` stayed at
establishment 0.59 and finished `insufficient`. `cannot generate string
responses` was established at 0.88 and stayed `manual_review` because safety
was 0.68.

### 6. Chaos cases remain distinguishable when the whole case is one source

- raw `cannot sign in`: support 0.98, establishment 0.90, safety 0.67, action `proceed`, verdict `manual_review`
- LLM-augmented: establishment 0.79, verdict `insufficient`
- resolved `cannot sign in`: establishment 0.90, safety 0.58, verdict `manual_review`
- resolved `account access`: establishment 0.79, verdict `insufficient`
- conflicted `cannot sign in`: action `manual_review`, conflict 0.17, verdict `manual_review`
- conflicted `duplicate charge`: conflict 0.69, verdict `conflict`

The LLM summary did not establish the sign-in claim. The conflicted billing
claim remained a conflict rather than a refutation: refuted was 0.06.

## Limits

- One live sample. Probabilities moved relative to the earlier `0.1` profile run. This file records the `0.2.0` sample only.
- Embedded claim-validation questions only. Original Noul, Choice, and Score maps were not sent.
- Exact-phrase retrieval, not hybrid Context retrieval.
- Caller sources are labeled `source_text` / `project` by the deployment.
- Policy is explicitly uncalibrated. This is not a calibration or SLO measurement. The calibration procedure is [calibration.md](calibration.md).
