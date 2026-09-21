# Local latency sample

Status: one local measurement on 2026-09-21. This is not a 28-day SLO,
not a cold-start study, and not a live-provider availability result.

Command:

```text
go test ./internal/httpapi -run TestLatencySampleAgreesOnVerdicts -count=1
```

Toolchain: go1.25.5 windows/amd64. Modules: `github.com/fastygo/context v0.1.0`,
`github.com/fastygo/framework v0.3.0`.

The sample ran in one process through the HTTP handler. Replay used one sealed
bundle. Evaluation used an in-process stub decider, not a network provider.
Every sampled response in this run was HTTP 200 with verdict `validated`.

| Path | N | P50 | P95 | P99 | Max |
|---|---:|---:|---:|---:|---:|
| Warm `POST /v1/replays` | 10000 | 1.05 ms | 2.99 ms | 5.60 ms | 172.38 ms |
| Stub `POST /v1/evaluations` | 1000 | 2.75 ms | 4.40 ms | 6.49 ms | 110.82 ms |

The proposed warm-replay targets are 250 ms at P95 and 1 s at P99. This sample
is inside those numbers and does not establish them. Stub evaluations do not
measure SLO-02. No cold/warm platform label was available. No deployment region,
CPU, or RSS figure is claimed from this run.

## Vulnerability review

`govulncheck` on 2026-09-21 reported no vulnerabilities in called code. One
vulnerability exists in a required module on a path this module does not call.
The module stayed on Go 1.25.0.
