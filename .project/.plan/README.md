# LeX implementation plan

Status: canary release plan, updated 2026-09-22. Context v0.1.0 is
published. The generic decision, legacy validation, and replay slices run on
the canary deployment. SLO achievement and standards certification are not
claimed. Stable-release gates are listed in [progress.md](progress.md).

## Fixed scope

- Go implementation exposing a REST API; no TypeScript SDK or JavaScript application runtime.
- Required modules: `github.com/fastygo/framework@v0.3.0` and
  `github.com/fastygo/context@v0.1.0`.
- Vercel serverless deployment with request-scoped RAM only for mutable protocol data.
- No database, disk persistence, external cache, object store, durable queue, or server-side run history.
- Validation and replay only; side-effect execution is deferred.
- Provider-neutral typed decisions; no dependency on GoBackend.

These constraints supersede earlier suggestions to build a TypeScript SDK or
persist server-side replay files. Caller retention of a returned bundle is
optional consumer behavior, not a server storage dependency.

## Reading order

1. [Architecture and deployment profile](architecture.md).
2. [ADR register](adr/README.md), with a decision and proof obligation per topic.
3. [SLO and resource budgets](slo.md).
4. [Standards and conformance gates](conformance.md).
5. [Delivery sequence](delivery.md).
6. [Pinned Context version and capability](context-version.md).

The [canonical specification](../.lex/README.md) remains authoritative for core
semantics. This folder owns deployment planning, ADRs, and service targets,
not a second complete protocol.

## Decision status

User-selected constraints above are fixed. ADR-0001 through ADR-0013 are
accepted for the canary release; evidence still owed for stable is tracked in
[progress.md](progress.md). New ADRs stay proposed until their acceptance
tests and maintainer review are recorded. Each ADR carries
context, decision, consequences, and acceptance evidence. Supersede records
instead of silently changing an accepted decision.

## Context dependency readiness

Published Context v0.1.0 supplies pkg/contextkit/runtime, an immutable RAM
adapter for exact phrase retrieval and existing ContextPack construction.
The [version baseline](context-version.md) pins the tag, commit, capability,
limits, and proof. [ADR-0003](adr/0003-context-boundary.md) selects this path.

The missing embedded-interface blocker is closed for memory-exact-v1.
Framework composition, LeX evidence binding, and the direct-adapter Vercel
deployment are in place. Race testing, SLO measurement, hosted-adapter proof
on that deployment, and the conformance report remain open in
[progress.md](progress.md).
