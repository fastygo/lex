# ADR-0011: Budgets, measurement, and overload

Status: proposed. Date: 2026-09-21. Budgets are implemented.
SLO measurement is deferred. See [progress.md](../progress.md).

## Context

Long calls and unbounded bundles can violate serverless duration, memory, and payload limits. Local counters disappear on instance replacement.

## Decision

Adopt proposed SLOs and admission budgets in ../slo.md, subject to benchmark evidence. Bound input/output, provider time, concurrency, and schema work. Return controlled overload errors. Emit bounded metadata events and use external/platform aggregation; no in-service metrics database. Include upstream and platform failures in user-visible availability.

## Consequences and alternatives

Targets are not achieved SLOs until measured over the declared window. Logs are not evidence storage. Platform telemetry can persist metadata outside the RAM-only application and its retention must be disclosed. Lack of observability yields unknown, not success.

## Acceptance evidence

Warm/cold/load benchmarks with exact environment and percentile counts; full-window SLI report; oversize/slow-provider/OOM pressure tests; canceled goroutine/connection cleanup; telemetry redaction and error-budget response drill.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
