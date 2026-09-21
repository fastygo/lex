# ADR-0009: Typed decisions and provider capabilities

Status: proposed. Date: 2026-09-21.
Scope constraints in [the plan](../README.md) are fixed; implementation details
require acceptance evidence. Owner: LeX maintainers. No implementation claimed.

## Context

Provider paths differ in metadata, limits, and failure behavior even for the same reference model.

## Decision

Use separate direct and hosted adapters behind a provider-neutral typed interface. Declare Noul/Choice/Score support and limits before execution; preserve raw answers, exact input bindings, resolved identity, and available metadata. Bound all network reads and respect request cancellation. Missing required capability fails explicitly. No silent fallback.

The initial direct adapter allowlists `https://api.typesafe.ai/v1/systemone`
and requests `jev-1.13.0`. The hosted adapter allowlists
`https://openrouter.ai/api/v1/systemone`, requests `typesafe/jev-1.13`, and
requires `LEX_HOSTED_RESOLVED_MODEL`, an exact immutable identity confirmed by
the provider for the deployment. Both adapter transport and replay compare
against that explicit pin. There is no default hosted resolved identity and
no suffix-based inference of immutability. An echoed selector, `latest`,
`stable`, `preview`, missing pin, or a different response identity is refused.
The dated identities in conformance fixtures are synthetic, not release evidence.
Neither adapter accepts caller-provided endpoints, credentials, or model pins.
Deployment supplies `LEX_TYPESAFE_API_KEY` and `LEX_OPENROUTER_API_KEY` as
separate secrets; neither value enters a request, bundle, trace, or log.

## Consequences and alternatives

Provider credentials and endpoint/billing details stay outside core semantics. A provider alias cannot support a reproducibility claim. Hosted routing must disclose sufficient resolved identity; inability to do so is a capability failure. The RAM-only path uses Context's immutable embedded runtime. If an optional HTTP evidence adapter is later added, contextkit clients with mutable LastAPIVersion must not be shared unsafely across requests.

## Acceptance evidence

Common valid/invalid fixtures on two adapter paths, unknown answer/domain checks, transport vs decision errors, alias resolution, bounded responses, cancellation, calibration per path, and trace secret scanning.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
