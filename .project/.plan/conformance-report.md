# Conformance report

Status: local evidence on 2026-09-21. This is not an IETF, ISO, or vendor
certification, and it does not close the 28-day SLO window.

| Criterion | Result | Evidence |
|---|---|---|
| STD-02 JSON strictness | Local pass | Duplicate keys, trailing values, invalid UTF-8, evaluation-body duplicates, and queries longer than 4096 characters are rejected. |
| STD-03 canonical hashes | Partial | Key order, Unicode, and numeric edges (`-0`, `1.0`, `1e2`) match independent SHA-256. A one-byte source edit changes the pack hash. A second language consumer is not proven. |
| STD-05 HTTP contract | Partial | Method 405 with Allow, Accept 406, media type, auth, no-store, 413, 422, 502, 503, and 504 are tested locally. Unknown routes return 404 problem JSON. More than 128 sources is rejected before a provider call. The deployed revision does not include every later status. |
| STD-06 OpenAPI and problems | Partial | Draft OpenAPI 3.1.1 validates its problem example, a technical-verdict example, and the evaluation request. The handler rejects bodies that fail that request schema before building a pack. Optional metadata is bounded and is not sent to the provider or stored in the bundle. |
| LEX-01 evidence and authority | Partial | Project isolation, caller URL rejection, caller policy rejection, injection text staying out of questions, inference-only replay, compound or overlapping questions rejected after a matching self-hash, source checksums checked against frozen text, and secret redaction are tested. A calibration corpus is not. |
| LEX-02 decisions and replay | Local pass for the embedded profile | All six verdicts, precedence, contradictory evidence at the HTTP boundary, conflict retained beside a safety-gate finding, network-disabled replay, and both adapter fixtures pass. A repeated request with the same Idempotency-Key on a fresh instance calls the provider again. Replay recomputes the pack hash and entity checksum and rejects a foreign verifier or adapter version. A criteria edit changes the question-set hash. |
| SEC-01 deployment security | Partial | Bearer binding, cookie rejection, forged approval rejection, CORS denial, overload, depth, body limits, ambiguous provider JSON, and NUL source text pass locally. `govulncheck` found no called vulnerability. Race tests and a full telemetry inspection are not proven. |
| DEP-01 Context pin | Partial | Context v0.1.0 builds a pack through the handler, keeps a budget rejection, stores its checksum separately from the LeX pack hash, checks the snapshot identity against its sources, rejects a source that exceeds the focus character budget before a provider call, and rejects input above 256 KiB or a duplicate source id before a provider call. Race tests on a gcc-capable runner are not proven. |
| OPS-01 serverless | Partial | Evaluation leaves the working directory unchanged. One deployed sample exists for an earlier revision. Cold/warm platform labels and filesystem instrumentation on Vercel are absent. |
| OPS-02 SLO | Preliminary | `.project/.plan/measurements.md` records one local sample. It is not a 28-day SLI. |

Explicit exclusions in [conformance.md](conformance.md) remain excluded.
