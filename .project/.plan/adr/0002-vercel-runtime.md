# ADR-0002: Vercel Go deployment and handler lifecycle

Status: proposed. Date: 2026-09-21. The handler is deployed.
Toolchain, region, duration, and cold-start evidence are still open. See [progress.md](../progress.md).

## Context

Serverless instances may be replaced and cannot own durable protocol state. Framework already exposes App.Handler().

## Decision

Use one net/http handler, preferably hosted by the Vercel Go preset with cmd/api/main.go and PORT. Keep a thin function-entry wrapper as a deployment alternative, not a second implementation. No scheduled workers, background persistence, or correctness-critical post-response work. Secrets come from deployment configuration.

## Consequences and alternatives

The runtime choice is a deployment concern. Current Go support is documented as Beta. Pin and test the actual entry mode, plan, region, duration, bundle size, and toolchain before claiming support.

## Acceptance evidence

Deploy a minimal preview; prove routes, cold start, no static/HTML middleware interference, deadline cancellation, and fresh-instance operation. Re-run the probe after dependency/runtime changes.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
