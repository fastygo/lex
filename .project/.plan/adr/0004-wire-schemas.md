# ADR-0004: Wire schemas, versions, and extensions

Status: accepted. Date: 2026-09-22.
Owner: LeX maintainers. JSON Schema 2020-12 is pinned through
`github.com/santhosh-tekuri/jsonschema/v6`. Positive and negative fixtures,
duplicate-key rejection, OpenAPI examples, and protocol-version rejection are
covered by `internal/wire/openapi_test.go`, `internal/wire/request_test.go`,
`internal/wire/bundle_test.go`, and `internal/httpapi/handler_test.go`.

## Context

Go structs alone do not define a language-neutral protocol. Unknown fields and changing criteria can alter semantics.

## Decision

Use versioned JSON Schema 2020-12 as the wire-shape source; generate Go transport types where tooling preserves semantics. Keep normative meaning in .lex. Require explicit protocol/profile versions. Reject unsupported versions and unknown normative fields/enums in v0.1; reserve a bounded namespaced metadata extension area that cannot change decisions.

Replay bundles are decoded into one typed `wire.Bundle` after schema and
self-hash validation, not walked as untyped maps. No generator is adopted: the
typed form is a hand-maintained mirror and `TestBundleTypeMirrorsSchema` fails
when the type and `replay-bundle.schema.json` name different properties at any
level, so the mirror cannot drift silently. Context's own documents (pack,
snapshot, pack request) stay Context JSON inside the bundle and are decoded
through the evidence plane into Context's public types with unknown fields
refused.

## Consequences and alternatives

Additive fields are not automatically compatible with strict validation or hashes. Immutable schema identifiers and an explicit compatibility decision are required. Authoring tools and SDKs are deferred.

## Acceptance evidence

Positive and negative schema fixtures, required-field and numeric-domain checks, duplicate-key rejection before decoding, and examples validated against the same schemas. Explicit format assertions and version-negotiation tests. Select and pin a Go validator after running dialect tests.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
