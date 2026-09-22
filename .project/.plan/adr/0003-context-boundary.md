# ADR-0003: Context public boundary and RAM integration

Status: selected dependency; the embedded runtime is called from the evaluation handler. Updated: 2026-09-22.
Owner: LeX maintainers. Full acceptance still requires the race evidence named in [progress.md](../progress.md).

## Context

The original proposal identified an HTTP-only public boundary and used
caller-supplied packs as a fallback. Context v0.1.0 now adds the separate
pkg/contextkit/runtime adapter under upstream ADR-0045 while preserving the
parent HTTP client. This update replaces the earlier proposed fallback.

## Decision

Pin github.com/fastygo/context v0.1.0 at commit
0715ad1ddbc1f749f8e0f0ac1864218eb09a2d13, capability memory-exact-v1.
The [dependency baseline](../context-version.md) owns version and proof details.

Construct one immutable Runtime per evaluation from authenticated,
project-scoped, versioned source texts. Map the reviewed LeX profile to the
supported Context Focus and PackRequest; unsupported profile requirements
must fail explicitly. Call ContextPack in process, then freeze the result
before invoking a decision provider. No HTTP service, disk, database, or
downstream import of Context internal packages is required.

The baseline evaluation path builds a real pack in RAM. Replay consumes a saved
pack and DecisionSet. External caller-pack evaluation is optional and not
required to work around a missing embedded interface anymore.

## Consequences and alternatives

The original embedded-interface blocker is closed for exact phrase retrieval
and pack construction. This is not full Context CLI parity: sparse/dense
retrieval, morphology, and parsing are unsupported by this profile.

Use deployment-owned mappings for trust/classes and policy references; source
labels from an untrusted caller do not confer authority. Export the source
Snapshot, exact PackRequest, pack, and component identities in the LeX bundle.
Preserve Context checksums separately from LeX JCS hashes.

## Acceptance evidence

Upstream package and full offline tests, vet, builder parity, and an external
Go consumer passed as recorded in the dependency baseline. Race instrumentation
and Vercel deployment were not verified there.

LeX acceptance still requires version pinning, real pack-to-EvidenceBinding
mapping, unsupported-capability rejection, project isolation, source version
and checksum checks, complete bounded bundles, race tests on a supported runner,
and Vercel composition. Do not mark all P0 or conformance gates complete.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
