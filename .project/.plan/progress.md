# Progress

Updated: 2026-09-22. Release: **canary**, envelope `0.2`.

## Canary baseline

- [x] One operation, `POST /v1/decisions`, with replay; routes `GET /healthz`,
  `GET /v1/capabilities`, `POST /v1/decisions`, `POST /v1/replays`
- [x] Caller-owned State and QuestionSet; Noul, Choice, and Score; no profiles,
  thresholds, or verdicts in the core
- [x] Optional frozen Context binding rebuilt by Context before the provider call
- [x] Structural verifier, stable finding codes, self-hashed bundle, provider-free replay
- [x] Direct adapter live as `direct-systemone` / `jev-1.13.0`; hosted adapter
  passes local fixtures
- [x] Schemas, OpenAPI 3.1.1, and live responses validated in tests; every MUST
  line mapped to a test
- [x] Client guide and portable `lex-api` skill
- [x] Revision-pinned canary with replay and rollback point
  ([conformance-report.md](conformance-report.md))

## Open until stable

Stable follows after several independent products use LeX in production and
these gates close.

- [ ] Use by several independent products, with integration findings recorded
- [ ] Race evidence on a gcc-capable runner (0003)
- [ ] SLO measurements over the 28-day window against the targets in
  [architecture.md](architecture.md) (0011)
- [ ] Stable conformance report (0012)
- [ ] Vulnerability review repeated per release (0010)
- [ ] Hosted adapter proof on the deployment (0009)
- [ ] Per-adapter calibration report (0009)
- [ ] Deployed toolchain identity recorded (0001)
- [ ] Region, duration, bundle size, and cold start recorded (0002)

Numbers refer to [decisions.md](decisions.md).
