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
