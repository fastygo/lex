# ADR-0004: Wire schemas, versions, and extensions

Status: proposed. Date: 2026-09-21.
Scope constraints in [the plan](../README.md) are fixed; implementation details
require acceptance evidence. Owner: LeX maintainers. No implementation claimed.

## Context

Go structs alone do not define a language-neutral protocol. Unknown fields and changing criteria can alter semantics.

## Decision

Use versioned JSON Schema 2020-12 as the wire-shape source; generate Go transport types where tooling preserves semantics. Keep normative meaning in .lex. Require explicit protocol/profile versions. Reject unsupported versions and unknown normative fields/enums in v0.1; reserve a bounded namespaced metadata extension area that cannot change decisions.

## Consequences and alternatives

Additive fields are not automatically compatible with strict validation or hashes. Immutable schema identifiers and an explicit compatibility decision are required. Authoring tools and SDKs are deferred.

## Acceptance evidence

Positive and negative schema fixtures, required-field and numeric-domain checks, duplicate-key rejection before decoding, and examples validated against the same schemas. Explicit format assertions and version-negotiation tests. Select and pin a Go validator after running dialect tests.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
