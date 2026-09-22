# Architecture decision register

Status: decision records for the Go REST, Vercel, RAM-only profile. ADR-0004,
ADR-0005, ADR-0006, ADR-0007, and ADR-0008 are accepted. ADR-0003 selects
published Context v0.1.0; its race evidence remains pending. The other records
stay proposed until the evidence named in [progress.md](../progress.md) exists.
Accept only with recorded review and linked evidence; keep a supersession trail.
User constraints do not need reconfirmation. Unresolved engineering details
must not be presented as shipped behavior.

- [ADR-0001: Go, Framework, and dependency direction](0001-go-framework.md).
- [ADR-0002: Vercel Go deployment and handler lifecycle](0002-vercel-runtime.md).
- [ADR-0003: Context public boundary and RAM feasibility](0003-context-boundary.md).
- [ADR-0004: Wire schemas, versions, and extensions](0004-wire-schemas.md).
- [ADR-0005: Canonical hashes and immutable input binding](0005-hashes.md).
- [ADR-0006: Synchronous REST surface and error mapping](0006-rest-errors.md).
- [ADR-0007: Request RAM and caller-owned replay](0007-memory-replay.md).
- [ADR-0008: Lifecycle, verdict precedence, and retries](0008-state-verdict.md).
- [ADR-0009: Typed decisions and provider capabilities](0009-provider-adapters.md).
- [ADR-0010: Authority, isolation, and safe transport](0010-security.md).
- [ADR-0011: Budgets, measurement, and overload](0011-slo-observability.md).
- [ADR-0012: Standards conformance and release evidence](0012-conformance-release.md).

Resolve 0001-0003 in P0, 0004-0008 in P1, 0009 in P2, and 0010-0012
by P3. Security and budgets apply from the first implementation, even when
final evidence is collected later. SLO definitions live in [slo.md](../slo.md);
standards acceptance lives in [conformance.md](../conformance.md).
