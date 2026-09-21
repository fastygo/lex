# ADR-0006: Synchronous REST surface and error mapping

Status: proposed. Date: 2026-09-21.
Scope constraints in [the plan](../README.md) are fixed; implementation details
require acceptance evidence. Owner: LeX maintainers. No implementation claimed.

## Context

RAM-only operation cannot promise a durable job resource or history lookup.

## Decision

Propose POST /v1/evaluations for live evaluation and POST /v1/replays for bundle verification, GET /v1/capabilities and GET /healthz for discovery and liveness. Publish OpenAPI separately as an immutable release artifact. POST returns 200 for completed non-error verdicts; malformed JSON 400, invalid contract 422, auth 401/403, unsupported media 415, body limit 413, overload 503, provider failure 502, deadline 504, internal failure 500. Specify 405/406 behavior and exact stage mapping in the HTTP binding.

## Consequences and alternatives

These routes/codes are proposals until normative promotion. Error responses use Problem Details with safe stage/cause extensions and available trace fragments; no 200 technical error. No GET run history, 201 persisted-resource claim, or 202 queue. Platform-generated failures may bypass the handler format.

## Acceptance evidence

Black-box method, content negotiation, status/body agreement, Cache-Control: no-store, and every verdict/stage mapping. Framework middleware must preserve the API error contract.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
