# Sources and standards applicability

Status: reference index. External documents govern their own contracts, not
LeX verdict semantics. Research and brainstorm notes are informative.

## Requirement language

[RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and
[RFC 8174](https://www.rfc-editor.org/rfc/rfc8174) define requirement keywords
used under [governance.md](governance.md).

## Intended serialization and API baseline

The following are inputs to the release work in [analysis.md](analysis.md).
They do not define a finished LeX wire profile merely by being listed.

- [JSON, RFC 8259](https://www.rfc-editor.org/rfc/rfc8259): JSON message syntax.
- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12): intended schema dialect and source for generated types.
- [JCS, RFC 8785](https://www.rfc-editor.org/rfc/rfc8785): candidate JSON canonicalization for SHA-256 content hashes. Specify numeric limits, duplicate-key rejection, hash scope, and test vectors before adoption.
- [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339): timestamp baseline. The v0.1 profile carries no audit timestamps on evaluation requests, evaluation responses, replay bundles, replay responses, or problem bodies. Replay does not compare wall-clock time. A preserved Context pack is not a LeX clock. This is narrower than RFC 3339 and is not date parsing.
- [HTTP semantics, RFC 9110](https://www.rfc-editor.org/rfc/rfc9110): intended first remote transport.
- [Problem Details, RFC 9457](https://www.rfc-editor.org/rfc/rfc9457): candidate HTTP error format; stage errors still need explicit mapping.
- [OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1.html): candidate HTTP interface description, not a replacement for semantics or lifecycle rules.

A hash detects changes relative to an expected digest; it does not authenticate
an authority by itself. Signature formats, authentication, and distributed event
standards require a concrete threat model and are not implicitly mandatory here.

## Context Runtime

Use the sibling repository's shipped documentation:

- [Documentation index](../../../@Context/docs/README.md).
- [HTTP API v1](../../../@Context/docs/api/v1.md).
- [API changelog](../../../@Context/docs/api/v1-changelog.md).
- [Lab gate and frozen integration boundary](../../../@Context/docs/lab-gate.md).
- [ADR-0020: ContextPack budget and evidence](../../../@Context/docs/decisions/0020-contextpack-budget-and-evidence.md).

These relative links assume the documented sibling checkout layout.
The public Go integration is `github.com/fastygo/context/pkg/contextkit`.
Pin upstream versions during integration. Upstream plans are not shipped
capabilities, and LeX documentation cannot redefine the Context v1 wire format.

## Decision providers and local research

Jev is the reference typed-decision model; direct TypeSafe and OpenRouter are
adapter access paths. Pin endpoint/API capabilities and resolved model versions
in adapter implementation evidence rather than core object names.

[Research findings](../.jev/research-findings.md) and
[example captures](../.jev/README.md) document local observations. They do not
prove provider conformance, calibration, or current endpoint availability.
The [historical protocol notes](../.jev/lex-protocol-history.md) are not a normative source.
