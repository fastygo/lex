# ADR-0008: Lifecycle, verdict precedence, and retries

Status: accepted. Date: 2026-09-22.
Owner: LeX maintainers. The verdict truth table, combined findings, deadline and
disconnect at pack, decide, verify, and replay, and a repeated request on a
fresh instance are covered by `internal/conformance/policy_gate_test.go`,
`internal/verify/verdict_test.go`, and `internal/httpapi/handler_test.go`.

## Context

One primary verdict must not erase conflicting evidence or become an authorization token.

## Decision

Implement the canonical validation stages as a deterministic request-scoped state machine. Resolve primary verdicts from explicit findings with the total order `error > conflict > insufficient > manual_review > rejected > validated`. Preserve all findings; selection of a primary verdict does not discard lower-priority findings. No execution extension. Automatic provider retries are disabled initially; cancellation stops new stages.

## Consequences and alternatives

Repeated live POST may re-evaluate and cost again; neither a run id nor an Idempotency-Key provides global deduplication without storage. Do not advertise exactly-once or durable idempotency. Replay is deterministic under the saved inputs, although audit ids/timestamps differ.

## Acceptance evidence

Approve a complete verdict truth table including combined failures. Test every legal and illegal transition, deadline and disconnect at each stage, simultaneous findings, repeated requests across fresh instances, and zero side effects during replay.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
