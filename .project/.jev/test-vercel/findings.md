# LeX protocol run on Vercel

Status: research observation against `https://lexproto.vercel.app` on 2026-09-21.
No credentials are recorded. This run does not call TypeSafe or OpenRouter
directly. The deployment adapter already holds the decision credential.

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
| playground-structured-decisions | structured decisions | 0.80 | 0.62 | 0.11 | 0.54 | proceed | insufficient |
| playground-cannot-generate-strings | cannot generate string responses | 0.97 | 0.92 | 0.05 | 0.58 | proceed | manual_review |
| playground-parallel-questions | evaluated independently and in parallel | 0.97 | 0.91 | 0.07 | 0.65 | proceed | manual_review |
| playground-noul-no-confidence | no separate confidence field | 0.89 | 0.85 | 0.41 | 0.63 | proceed | manual_review |
| llm-next-token | next token | 0.81 | 0.65 | 0.09 | 0.64 | proceed | insufficient |
| llm-temperature-zero | Temperature 0 | 0.74 | 0.51 | 0.29 | 0.54 | manual_review | insufficient |
| llm-proposal-not-authority | treat the model output as a proposal | 0.97 | 0.88 | 0.05 | 0.42 | proceed | manual_review |
| llm-hallucination-structural | Hallucination is structural | 0.95 | 0.78 | 0.06 | 0.54 | proceed | insufficient |
| context-restore-access | restore access | 0.94 | 0.90 | 0.09 | 0.54 | proceed | manual_review |
| context-refund | refund | 0.05 | 0.06 | 0.92 | 0.25 | reject | conflict |
| context-account-access | account access | 0.95 | 0.93 | 0.08 | 0.57 | proceed | manual_review |
| context-duplicate-charge | duplicate charge | 0.04 | 0.04 | 0.96 | 0.39 | reject | conflict |
| chaos-raw-sign-in | cannot sign in | 0.93 | 0.77 | 0.07 | 0.39 | proceed | insufficient |
| chaos-llm-augmented-sign-in | cannot sign in | 0.83 | 0.65 | 0.14 | 0.13 | manual_review | insufficient |
| chaos-resolved-sign-in | cannot sign in | 0.90 | 0.81 | 0.13 | 0.50 | proceed | manual_review |
| chaos-resolved-account-access | account access | 0.85 | 0.77 | 0.13 | 0.54 | proceed | insufficient |
| chaos-conflicted-sign-in | cannot sign in | 0.88 | 0.76 | 0.14 | 0.10 | manual_review | insufficient |
| chaos-conflicted-duplicate-charge | duplicate charge | 0.66 | 0.44 | 0.34 | 0.09 | manual_review | insufficient |

## Observations

### 1. The protocol path is the evaluation route, not a Jev question map

The deployment accepted these bodies because they are LeX evaluation objects.
The adapter then called Jev with the frozen pack and the pinned question set.
Posting `questions-*.json` from the examples is still not a valid evaluation.

### 2. Atomic claims over the same note still diverge

On the playground note, `cannot generate string responses` and
`evaluated independently and in parallel` reached establishment above 0.90.
`structured decisions` stayed at 0.62 and finished `insufficient`. Safety
stayed below 0.80 in every playground and LLM case, so even strong support
became `manual_review` rather than `validated`.

### 3. Context refund and duplicate-charge claims were rejected as conflict

With the full test scenario as one source, claim `refund` produced support
0.05, conflict 0.92, and action `reject`. Claim `duplicate charge` was the
same shape. Claims `restore access` and `account access` had high support and
establishment, but safety ~0.55 kept the verdict at `manual_review`. That is
the expected LeX split: access is supported; refund is not.

### 4. Chaos cases are distinguishable once the whole case is one source

Unlike the earlier split-source probe, Jev saw payment, lock, and policy text
together.

- raw `cannot sign in`: high support, establishment 0.77, safety 0.39
- LLM-augmented: safety dropped to 0.13 and action became `manual_review`
- resolved `cannot sign in`: establishment 0.81, still gated by safety 0.50
- conflicted `cannot sign in`: safety 0.10
- conflicted `duplicate charge`: support 0.66, conflict 0.34, action
  `manual_review` with confidence 0.99

The LLM summary still did not establish a billing fact, but it lowered the
safety signal. Unresolved ledgers kept automatic action unsafe.

### 5. A proceed Choice is not a validated verdict

Several cases chose `proceed` while safety or establishment failed the
embedded thresholds. The verifier recorded `safety_gate` and sometimes
`action_inconsistent`. No request in this sample returned `validated`.

## Limits

- Embedded claim-validation questions only; original Noul/Choice/Score maps
  were not sent.
- Exact-phrase retrieval, not hybrid Context retrieval.
- Caller sources are labeled `source_text` / `project` by the deployment.
- Policy is explicitly uncalibrated.
- One request per claim; not a calibration or SLO measurement.
