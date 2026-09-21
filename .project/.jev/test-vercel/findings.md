# LeX protocol run on Vercel

Status: research observation against `https://lexproto.vercel.app` on 2026-09-21,
deployment revision `1c9cc90`. No credentials are recorded. This run does not
call TypeSafe or OpenRouter directly. The deployment adapter already holds the
decision credential. Files under `evaluations/` and `results.json` keep the
earlier sample; the table below is the rerun.

Request bodies are in [requests/](requests/). Compact verdicts are in
[evaluations/](evaluations/). The index is [results.json](results.json).

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

The wire schema accepts only this shape. Caller policy, custom TypeSafe
question maps, and extra entity types are rejected. The embedded profile asks
Jev `support`, `established`, `conflict`, `safe_to_auto_act`, and `action`.
Policy is `claim-validation` 0.1.0, calibration `uncalibrated`.

The claim is the `query` phrase. Exact retrieval keeps a source only when that
phrase appears in its text. Chaos and context cases are therefore one source
each (the case section), so Jev sees the full narrative the way the original
example state did.

## Live verdicts

All 18 evaluations returned HTTP 200, resolved model `jev-1.13.0`, adapter
`direct-systemone`. Every sealed bundle replayed to the same verdict.

| Request | Query | support | established | conflict | safe | action | Verdict |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| playground-structured-decisions | structured decisions | 0.93 | 0.60 | 0.09 | 0.49 | proceed | insufficient |
| playground-cannot-generate-strings | cannot generate string responses | 0.97 | 0.91 | 0.07 | 0.70 | proceed | manual_review |
| playground-parallel-questions | evaluated independently and in parallel | 0.98 | 0.92 | 0.06 | 0.80 | proceed | validated |
| playground-noul-no-confidence | no separate confidence field | 0.95 | 0.80 | 0.44 | 0.55 | proceed | manual_review |
| llm-next-token | next token | 0.88 | 0.66 | 0.07 | 0.64 | proceed | insufficient |
| llm-temperature-zero | Temperature 0 | 0.85 | 0.55 | 0.20 | 0.50 | manual_review | insufficient |
| llm-proposal-not-authority | treat the model output as a proposal | 0.98 | 0.92 | 0.05 | 0.53 | manual_review | manual_review |
| llm-hallucination-structural | Hallucination is structural | 0.99 | 0.81 | 0.05 | 0.56 | proceed | manual_review |
| context-restore-access | restore access | 0.97 | 0.90 | 0.14 | 0.63 | proceed | manual_review |
| context-refund | refund | 0.35 | 0.04 | 0.91 | 0.15 | reject | conflict |
| context-account-access | account access | 0.90 | 0.93 | 0.13 | 0.60 | proceed | manual_review |
| context-duplicate-charge | duplicate charge | 0.11 | 0.03 | 0.97 | 0.22 | reject | conflict |
| chaos-raw-sign-in | cannot sign in | 0.98 | 0.91 | 0.06 | 0.66 | proceed | manual_review |
| chaos-llm-augmented-sign-in | cannot sign in | 0.96 | 0.82 | 0.17 | 0.29 | manual_review | manual_review |
| chaos-resolved-sign-in | cannot sign in | 0.97 | 0.90 | 0.11 | 0.60 | proceed | manual_review |
| chaos-resolved-account-access | account access | 0.78 | 0.78 | 0.12 | 0.55 | proceed | insufficient |
| chaos-conflicted-sign-in | cannot sign in | 0.97 | 0.88 | 0.11 | 0.29 | manual_review | manual_review |
| chaos-conflicted-duplicate-charge | duplicate charge | 0.91 | 0.43 | 0.50 | 0.11 | manual_review | conflict |

## Observations

### 1. The protocol path is the evaluation route, not a Jev question map

The deployment accepted these bodies because they are LeX evaluation objects.
The adapter then called Jev with the frozen pack and the pinned question set.
Posting `questions-*.json` from the examples is still not a valid evaluation.

### 2. Atomic claims over the same note still diverge

On the playground note, `evaluated independently and in parallel` reached
support 0.98, establishment 0.92, and safety 0.80, so the verdict was
`validated`. `cannot generate string responses` had the same shape of support
but safety 0.70, so it stayed `manual_review`. `structured decisions` stayed
at establishment 0.60 and finished `insufficient`.

### 3. Context refund and duplicate-charge claims were rejected as conflict

With the full test scenario as one source, claim `refund` produced support
0.35, conflict 0.91, and action `reject`. Claim `duplicate charge` produced
support 0.11, conflict 0.97, and action `reject`. Claims `restore access` and
`account access` had high support and establishment, but safety 0.63 and 0.60
kept those verdicts at `manual_review`. Access is supported; refund is not.

### 4. Chaos cases are distinguishable once the whole case is one source

Unlike the earlier split-source probe, Jev saw payment, lock, and policy text
together.

- raw `cannot sign in`: support 0.98, establishment 0.91, safety 0.66, action `proceed`, verdict `manual_review`
- LLM-augmented: safety 0.29 and action `manual_review`
- resolved `cannot sign in`: establishment 0.90, safety 0.60, verdict `manual_review`
- resolved `account access`: establishment 0.78, verdict `insufficient`
- conflicted `cannot sign in`: safety 0.29, action `manual_review`
- conflicted `duplicate charge`: support 0.91, conflict 0.50, action `manual_review`, verdict `conflict`

The LLM summary still did not establish a billing fact. A conflict probability
at the 0.50 threshold outranked the review action.

### 5. A proceed Choice is not a validated verdict

Several cases chose `proceed` while safety or establishment failed the
embedded thresholds, and the verdict stayed `manual_review` or `insufficient`.
One playground claim cleared every gate and returned `validated`. Posting
`questions-jev.json` itself returned HTTP 400 `invalid_json`.

## Limits

- Embedded claim-validation questions only; original Noul/Choice/Score maps
  were not sent.
- Exact-phrase retrieval, not hybrid Context retrieval.
- Caller sources are labeled `source_text` / `project` by the deployment.
- Policy is explicitly uncalibrated.
- One request per claim; not a calibration or SLO measurement.
