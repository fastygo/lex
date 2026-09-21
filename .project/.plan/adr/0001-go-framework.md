# ADR-0001: Go, Framework, and dependency direction

Status: proposed. Date: 2026-09-21.
Scope constraints in [the plan](../README.md) are fixed; implementation details
require acceptance evidence. Owner: LeX maintainers. No implementation claimed.

## Context

LeX needs a small Go REST service while keeping deterministic verification independent of hosting.

## Decision

Use github.com/fastygo/framework for HTTP composition and github.com/fastygo/context only through public adapter packages. Keep core semantics free of framework/provider SDK types. No GoBackend, TypeScript SDK, UI runtime, or JavaScript service. Pin Context to v0.1.0 as recorded in [context-version.md](../context-version.md); select Framework's exact revision and the deployed toolchain separately. Upstream Go 1.25 directives are compatibility inputs, not proof of the deployed toolchain.

## Consequences and alternatives

Transitive modules may appear in go.sum without being runtime requirements. Review the actual import graph and binary; absence of a database means no database configuration, connection, or persistence path, not deleting upstream module declarations.

## Acceptance evidence

Compile on the chosen Vercel toolchain; inspect go list -deps and module versions; test core without an HTTP server. Reject local replace paths in release builds.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
