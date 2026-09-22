# ADR-0006: Synchronous REST surface and error mapping

Status: accepted. Date: 2026-09-22.
Owner: LeX maintainers. Method and Accept negotiation, `Cache-Control: no-store`,
verdict and stage status mapping, and problem JSON are covered by
`internal/httpapi/handler_test.go`.

## Context

RAM-only operation cannot promise a durable job resource or history lookup.

## Decision

Serve POST /v1/evaluations, POST /v1/replays, GET /v1/capabilities, and GET /healthz. Publish OpenAPI as a draft artifact of this working draft. POST returns 200 for completed non-error verdicts. A technical error verdict is HTTP 422 and still returns the sealed bundle. Malformed JSON is 400, auth is 401/403, unsupported media is 415, the body limit is 413, a retryable provider failure is 503, a provider contract failure including a refused redirect is 502, the caller deadline is 504, and an internal failure is 500. Method and Accept behavior are specified in the HTTP binding. These routes are the running surface. ADR acceptance still waits on the black-box evidence listed in progress.md.

## Consequences and alternatives

The route and status mapping above is the running HTTP binding in `.project/.lex/checks.md`. Error responses use Problem Details with safe stage/cause extensions and available trace fragments. No GET run history, 201 persisted-resource claim, or 202 queue. Platform-generated failures may bypass the handler format.

## Acceptance evidence

Black-box method, content negotiation, status/body agreement, Cache-Control: no-store, and every verdict/stage mapping. Framework middleware must preserve the API error contract.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
