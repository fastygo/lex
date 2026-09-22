# Conformance report

Status: canary evidence, 2026-09-22. This is not a certification, a
calibration report, or an SLO measurement.

## Canary

Revision `d3b7e84` on `https://lexproto.vercel.app` (GitHub deployment
`6599093830`, Production, branch `beta`) is the canary of the typed-decision
switch, envelope `0.2`. It is the first revision without the removed
claim-validation route; bundles sealed by earlier revisions are not accepted.

Rollback: deployment `6598362708` (commit `c83d77b`), the previous production
build, reported `success` at the time of the samples. Rolling back restores
the removed route.

| Sample | Decisions valid + replayed | Negative probes | Removed route |
|---|---:|---|---|
| `canary/20260922T194012Z` | 3/3 | 422 `question_error`, 400 `invalid_json` | 404 `not_found` |
| `canary/20260922T194038Z` | 3/3 | 422 `question_error`, 400 `invalid_json` | not sampled |

Each sample sent the three `.project/examples/` requests (Noul + Choice,
Noul + Score, and Noul + Choice with a frozen Context binding). Every
decision returned HTTP 200 `structural_status: valid`, `protocol_version: 0.2`,
through `direct-systemone` / `jev-1.13.0`; every bundle replayed HTTP 200
`decision_reproduced` and valid. No bundle retained request `metadata`. The
raw-Jev `criteria` probe named `/question_set/questions/route` in `detail`;
neither negative probe reached the provider. Summaries contain no credential,
raw answer, or source text.

Probe: `LEX_REVISION=<sha> LEX_DEPLOYMENT_ID=<id> node scripts/probe-canary.mjs`.

## Local evidence

At this revision `go test ./... -count=1`, `go vet ./...`, and
`npm run check:english` pass. They cover:

- every MUST line in `protocol.md` and `checks.md` mapped to a test;
- schemas and OpenAPI validated against live handler responses;
- JCS hashes agreeing with independent `rfc8785` vectors;
- network-free replay, tampering, pin, and project-isolation cases;
- Context rebuild, malformed answers, budgets, deadlines, and adapter
  conformance fixtures for the direct and hosted paths.

Race tests, hosted-adapter proof on the deployment, calibration, and the SLO
window remain open in [progress.md](progress.md).
