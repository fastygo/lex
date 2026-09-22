# ADR-0012: Standards conformance and release evidence

Status: accepted. Date: 2026-09-22. The requirement-to-proof register,
local conformance tests, and revision-pinned canary report are in use. The
stable conformance report is a stable-release gate in [progress.md](../progress.md).

## Context

Standards names and successful compilation do not prove interoperability or semantic correctness.

## Decision

Use ../conformance.md as a requirement-to-proof register. Freeze schemas, normative semantics, HTTP binding, hash vectors, verifier build, and accepted ADRs together. Test independent producers/consumers without requiring a TypeScript SDK. Publish declared roles and exclusions; retain evidence in release/CI artifacts, not in the runtime.

## Consequences and alternatives

Deployment compatibility, protocol conformance, calibration, and SLOs are separate claims. A deployment-specific restriction must not silently redefine the provider-neutral protocol. No blanket industry-standard certification. Immutable releases support rollback; breaking semantic/hash changes require explicit version decisions.

## Acceptance evidence

All applicable mandatory criteria pass, schemas/OpenAPI examples agree, every requirement has evidence, adversarial and two-adapter tests pass, deployed Vercel smoke passes, and unresolved gaps are named rather than marked complete.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
