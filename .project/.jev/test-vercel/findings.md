# Deployed Jev scenario run

Status: research observation against `https://lexproto.vercel.app` on 2026-09-21.
Raw statuses and compact Jev answers are in [results.json](results.json). Per-case
captures are in [evaluations/](evaluations/). No credentials are recorded.

This run sends example-like claims through the deployed LeX evaluation route.
That route freezes exact-phrase evidence, calls Jev through
`direct-systemone`, and verifies the raw answers against the embedded
`claim-validation` policy. It is not a replay of the hand-authored TypeSafe
question maps in [`examples/`](../examples/). Those maps still are not a LeX
evaluation body.

Direct `POST https://api.typesafe.ai/v1/systemone` with the four example
question maps returned 401. Those unauthorized bodies are in [jev/](jev/).

## What Jev received

Each evaluation uses the pinned claim-validation questions:

| ID | Type | Role |
| --- | --- | --- |
| `support` | Noul | Does admissible evidence directly support the claim? |
| `established` | Noul | Is the claim sufficiently and coherently established? |
| `conflict` | Noul | Does admissible evidence support an incompatible conclusion? |
| `safe_to_auto_act` | Noul | Is automatic acceptance semantically safe? |
| `action` | Choice | `proceed`, `reject`, `manual_review`, or `other` |

The claim is the exact retrieval query. The deployment labels every source
`source_text` / `project`, so `model_inference` and `tool_output` distinctions
from the chaos and context notes are not preserved. `memory-exact-v1` keeps
only spans that contain the query phrase.

Uncalibrated thresholds: support 0.70, establishment 0.80, conflict 0.50,
safety 0.80.

## Live answers

Resolved model was `jev-1.13.0` for every evaluation that reached decide.
Every sealed bundle replayed to the same verdict.

| Case | Query | Evidence kept | support | established | conflict | safe | action | Verdict |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- | --- |
| playground-structured-decisions | structured decisions | 1 | 0.78 | 0.60 | 0.11 | 0.50 | proceed (0.49) | insufficient |
| llm-next-token | next token | 1 | 0.80 | 0.64 | 0.08 | 0.64 | proceed (0.41) | insufficient |
| context-restore-access | restore access | 1 | 0.90 | 0.67 | 0.07 | 0.49 | manual_review (0.74) | insufficient |
| context-refund | refund | 2 | 0.22 | 0.18 | 0.52 | 0.16 | manual_review (0.56) | conflict |
| chaos-raw-sign-in | cannot sign in | 1 | 0.93 | 0.77 | 0.07 | 0.45 | manual_review (0.49) | insufficient |
| chaos-llm-augmented-sign-in | cannot sign in | 1 | 0.93 | 0.78 | 0.07 | 0.46 | manual_review (0.40) | insufficient |
| chaos-resolved-sign-in | cannot sign in | 1 | 0.93 | 0.77 | 0.07 | 0.47 | manual_review (0.40) | insufficient |
| chaos-conflicted-sign-in | cannot sign in | 1 | 0.93 | 0.78 | 0.07 | 0.46 | manual_review (0.41) | insufficient |
| chaos-conflicted-duplicate-charge | duplicate charge | 1 | 0.63 | 0.34 | 0.09 | 0.42 | manual_review (0.90) | insufficient |

## Observations

### 1. Jev is reachable through the deployment

`GET /healthz` returned 200. Authenticated `GET /v1/capabilities` returned 200
with caller-owned replay and no server history. Nine example-like evaluations
called `jev-1.13.0` and returned typed Noul/Choice answers. This is one
deployment sample, not a calibration study.

### 2. Exact retrieval hid the extra chaos evidence

Queries `cannot sign in` only kept `customer_message`. Payment records,
authentication logs, runbooks, and ledger conflicts do not contain that phrase,
so Jev never saw them. `chaos-raw`, `chaos-llm-augmented`, `chaos-resolved`,
and `chaos-conflicted` therefore produced nearly the same distribution. That is
a retrieval-stage effect, not a repeat of the original chaos DecisionSets.

The original chaos capture sent the full case text as Jev state and could
distinguish resolved versus conflicted packs. This deployment cannot do that
until the query overlaps the distinguishing sources or retrieval is no longer
exact-phrase only.

### 3. A refund claim over mixed billing text produced conflict

`context-refund` kept `llm_report` and `refund_policy`. Jev returned low
support (0.22), low establishment (0.18), and conflict 0.52. The verifier
emitted `conflict` and retained `manual_review`. That matches the research
intent: an LLM refund hypothesis plus a policy that requires an explicit
refund request should not authorize a refund.

`context-restore-access` kept only the customer message. Support was high
(0.90) but establishment (0.67) and safety (0.49) stayed below threshold, so
the verdict was `insufficient` with `manual_review`.

### 4. Typed output still needs the verifier

On the playground and LLM notes Jev chose `proceed` while establishment and
safety were below threshold. The verifier recorded `action_inconsistent` and
kept the overall verdict `insufficient`. A winning Choice is not authority.

### 5. These results are not the original example question maps

The example files remain TypeSafe question maps and captured DecisionSets.
Posting them as LeX bodies is still invalid. This run asked the embedded
claim-validation questions over similar source text. It does not reproduce
playground primitives, LLM taxonomy Choices, or chaos support/action fan-out.

## Earlier HTTP-only probe

An earlier capture in this folder posted the raw example files to
`/v1/evaluations` and `/v1/replays`. Evaluations returned 404 at that time;
replays returned 422 `invalid_replay_bundle`. That probe is superseded by the
live Jev evaluations above. Health, bearer authentication, method rejection,
and media-type rejection remain separate HTTP gates.

## Limits

- No direct TypeSafe question-map reproduction in this folder.
- Exact-phrase retrieval, not Context hybrid retrieval.
- All sources coerced to `source_text` / `project`.
- Policy is explicitly `uncalibrated`.
- One request per case; no repeatability or calibration measurement.
- Not a 28-day SLO sample or conformance certification.
