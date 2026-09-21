# Deployed scenario run

Status: research observation against `https://lexproto.vercel.app` on 2026-09-21.
Raw statuses are in [results.json](results.json). No credentials are recorded.

The example files are TypeSafe question maps and captured DecisionSets. The
deployment accepted none of them as a LeX evaluation or replay. `POST /v1/evaluations`
returned 404. `POST /v1/replays` returned 422 `invalid_replay_bundle` for every
example body. Health, bearer authentication, method rejection, and media-type
rejection behaved as separate HTTP gates.

These captures remain evidence about Jev question design. They are not evidence
that Context Runtime built the packs or that LeX verified the answers.

## Later evaluation probe

A subsequent deployment of the evaluation and replay routes was probed on
2026-09-21 against `https://lexproto.vercel.app` for `example-project`.
No credentials are recorded.

- `GET /healthz` returned 200.
- Authenticated `GET /v1/capabilities` reported `memory-exact-v1`,
  `evaluation: true`, `replay: true`, and `execution: false`.
- A query absent from the source returned 200 `insufficient` at stage
  `retrieval` with `replay_available: false`.
- Query `account` over the sentence `The account is locked.` called resolved
  model `jev-1.13.0` and returned `insufficient`, retaining a `manual_review`
  finding. The embedded policy is `uncalibrated`.
- Resubmitting that bundle to `POST /v1/replays` returned
  `verdict_reproduced` with the same findings.
- Another project returned 403 `project_forbidden`.
- A source `url` field returned 400 `invalid_json`.
- A request without a bearer token returned 401 `authentication_required`.

This probe is one deployment sample. It is not a 28-day SLO measurement,
calibration study, or conformance certification.
