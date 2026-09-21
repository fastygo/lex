# Conformance report

Status: local evidence on 2026-09-21. This is not an IETF, ISO, or vendor
certification, and it does not close the 28-day SLO window.

| Criterion | Result | Evidence |
|---|---|---|
| STD-02 JSON strictness | Local pass | Duplicate keys, trailing values, invalid UTF-8, and evaluation-body duplicates are rejected. |
| STD-03 canonical hashes | Partial | Key order, Unicode, and numeric edges (`-0`, `1.0`, `1e2`) match independent SHA-256. A one-byte source edit changes the pack hash. A second language consumer is not proven. |
| STD-05 HTTP contract | Partial | Method 405 with Allow, Accept 406, media type, auth, no-store, 413, 422, 502, 503, and 504 are tested locally. More than 128 sources is rejected before a provider call. The deployed revision does not include every later status. |
| STD-06 OpenAPI and problems | Partial | Draft OpenAPI 3.1.1 validates its problem example and a technical-verdict example. |
| LEX-01 evidence and authority | Partial | Project isolation, caller URL rejection, caller policy rejection, injection text staying out of questions, inference-only replay, compound or overlapping questions rejected after a matching self-hash, source checksums checked against frozen text, and secret redaction are tested. A calibration corpus is not. |
| LEX-02 decisions and replay | Local pass for the embedded profile | All six verdicts, precedence, contradictory evidence at the HTTP boundary, network-disabled replay, and both adapter fixtures pass. Replay recomputes the pack hash and entity checksum. A criteria edit changes the question-set hash. |
| SEC-01 deployment security | Partial | Bearer binding, overload, depth, and body limits pass locally. `govulncheck` found no called vulnerability. Race tests and a full telemetry inspection are not proven. |
| DEP-01 Context pin | Partial | Context v0.1.0 builds a pack through the handler, keeps a budget rejection, and stores its checksum separately from the LeX pack hash. Race tests on a gcc-capable runner are not proven. |
| OPS-01 serverless | Partial | Evaluation leaves the working directory unchanged. One deployed sample exists for an earlier revision. Cold/warm platform labels and filesystem instrumentation on Vercel are absent. |
| OPS-02 SLO | Preliminary | `.project/.plan/measurements.md` records one local sample. It is not a 28-day SLI. |

Explicit exclusions in [conformance.md](conformance.md) remain excluded.
