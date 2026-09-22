# ADR-0005: Canonical hashes and immutable input binding

Status: accepted. Date: 2026-09-22.
Owner: LeX maintainers. Independent vectors come from `scripts/jcs_vectors.py`
using pinned `rfc8785`. `internal/canonical/vectors_test.go` checks those
vectors. `internal/wire/golden_test.go` checks bundle hashes and verdicts
against that script and `scripts/verdict_gate.py`.

## Context

Equivalent JSON can have different bytes; a checksum without a defined scope cannot support replay.

## Decision

Define JCS canonical JSON and SHA-256, with an explicit content projection per object excluding its own digest and separately classified volatile metadata. Hash original source bytes without normalization. Freeze entity, pack, profiles, questions, policy, adapter, resolved model, and verifier versions. Use bounded numeric domains compatible with the canonicalization profile.

## Consequences and alternatives

Hashes detect content changes but do not authenticate issuers. Context v0.1.0 snapshot/request identities and ADR-0020 pack checksums are separate upstream contracts, not JCS; preserve them and additionally bind the complete frozen representation under the LeX hash profile. Do not include credentials in hashable payloads or normalize provider answers by rewriting values. Question definition changes require a new version/hash.

## Acceptance evidence

Independent standard hash vectors plus LeX object vectors; Unicode/key order/numeric edges, tampering, self-hash exclusion, source-byte changes, and changed criteria. The implementation must match vectors independently of Go's ordinary JSON serializer.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
