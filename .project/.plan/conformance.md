# Standards and conformance acceptance

Status: planned criteria. Passing selected standards profiles is not IETF,
ISO, or vendor certification. Core LeX invariants remain in
[protocol.md](../.lex/protocol.md). Each criterion below requires recorded proof.

## Report format

Each result records criterion id, normative requirement id, role/profile,
fixture id and hash, expected/actual outcome, test command, implementation
revision, Go/toolchain and module versions, deployment/configuration, timestamp,
and pass/fail/not-applicable with justification. All applicable mandatory cases
must pass. Proposed artifact paths are assigned during P1; no files are
represented as already existing.

## STD-01: requirement language and governance

Use [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and
[RFC 8174](https://www.rfc-editor.org/rfc/rfc8174) for uppercase requirements.
Every normative obligation has an id and a test or a documented review check.
Separate normative semantics from examples, measured observations, and plans.
Proof: requirement inventory with no uncovered mandatory obligation.

## STD-02: JSON and schema validation

Use [RFC 8259](https://www.rfc-editor.org/rfc/rfc8259) JSON and
[JSON Schema 2020-12](https://json-schema.org/draft/2020-12).
The LeX profile rejects duplicate keys, invalid UTF-8, trailing JSON values,
non-finite/out-of-domain numbers, and unsupported versions. Specify unknown
properties/enums explicitly. Enforce required fields and limits before use.
Enable format validation where required; a format annotation alone is not proof.
Proof: positive/negative schema corpus, including decoder ambiguity and limits.

## STD-03: canonical identity

Use [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785) JCS and SHA-256 for the
declared JSON content scope. Specify exclusions for self-hash fields and
volatile metadata. Hash source bytes separately without Unicode normalization.
Do not equate Go map sorting with complete JCS conformance.
Proof: independent expected digests for key ordering, Unicode, numeric edges,
negative zero, rejected duplicate names, changed criteria, and source-byte edits.

## STD-04: timestamps and references

Use [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339) with a documented LeX
subset: UTC Z, fixed millisecond precision, and no leap-second input in v0.1.
Any narrower profile must be stated rather than called full RFC coverage.
Define reference identity, version, project binding, and resolution; a URI is
not permission to fetch it.
Proof: valid/invalid time boundaries, cross-project references, unresolved refs,
and prohibited network targets.

## STD-05: HTTP and REST contract

Apply [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110) semantics and
[RFC 9111](https://www.rfc-editor.org/rfc/rfc9111) caching rules.
Declare methods, status codes, Content-Type, Accept behavior, and limits.
POST evaluation/replay returns a synchronous representation; no durable
resource Location or 202 job is fabricated. Sensitive responses use no-store.
Proof: method/content negotiation cases, cache headers, authentication failures,
body caps, timeouts, and all documented status mappings on a deployed handler.

## STD-06: machine-readable API and errors

Publish [OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1.html) referencing the
canonical schemas; examples must validate. Use
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) application/problem+json for
HTTP errors, with stable type identifiers and stage/reason extensions.
HTTP status and problem status agree. Do not confuse an epistemic negative
verdict with a malformed HTTP request.
Proof: OpenAPI validation, black-box contract tests, and safe error payloads.
Platform failures before the handler may have platform-native bodies; test and
document that boundary rather than claiming LeX can rewrite them.

## LEX-01: evidence, policy, and authority

All ten invariants pass adversarial fixtures. No decision is evidence for its
own verdict; no caller-supplied policy grants authority. Verify project,
provenance, hashes, evidence classes, source bounds, and external permissions.
Proof: inference-only, missing, contradictory, compound-question, unauthorized,
and policy-tampering cases with explicit expected findings and verdicts.

## LEX-02: state, decisions, and replay

Pin all inputs and resolved versions. Preserve raw typed answers; define total
verdict precedence while retaining all findings. Replay with the saved
DecisionSet has no retrieval/model calls and produces the same deterministic
result. Exact trace timestamps need not match.
Proof: every verdict, every allowed/forbidden transition, altered bundles,
network-disabled replay, and two conforming provider-adapter paths.
Independently test wire interoperability; shared adapter code alone is insufficient.

## SEC-01: deployment security profile

TLS at the platform edge, authenticated principal-to-project binding,
deployment-controlled policy, bounded input, provider endpoint allowlists,
safe errors, and secret-free telemetry are mandatory deployment controls.
Hash integrity is not a signature or permission. Signing and OIDC are not
claimed implementations. No arbitrary evidence-URL fetching in the baseline.
Proof: access-control matrix, injection, SSRF, malformed input, concurrent
tenant isolation, dependency review, and telemetry inspection.

## DEP-01: pinned embedded Context compatibility

Resolve [Context v0.1.0](context-version.md) to the recorded commit and module
checksums. Test public New/ContextPack integration through Framework, exact-mode
capability checks, source versions/spans, rejected material, and the host's
trust assignment. Preserve upstream checksums separately from LeX JCS hashes.
Proof: pinned dependency manifest, a real runtime-built pack, negative profile
cases, complete bundle round-trip, and race tests on a supported runner.
Upstream offline tests do not prove LeX or Vercel deployment conformance.

## OPS-01: memory-only Vercel behavior

Use the [Go runtime contract](https://vercel.com/docs/functions/runtimes/go)
and record [actual platform limits](https://vercel.com/docs/functions/limitations).
Prove no runtime persistence, no correctness dependency on warm memory, and
complete bundles within the response cap. Replay must work on a fresh instance.
Proof: deployed cold/warm runs, filesystem/network instrumentation, concurrency
and memory tests, response-loss behavior, and cancellation tests.

## OPS-02: SLO evidence

Implement the denominators, probe boundaries, targets, and reporting in
[slo.md](slo.md). Capture missing telemetry as unknown and include upstream
failures in user-visible availability.
Proof: reproducible benchmark report plus live SLI time series; label a window
shorter than 28 days as preliminary.

## Explicit exclusions

No TypeScript SDK, binary transports, asynchronous messaging, distributed
exactly-once execution, signed receipts, durable history, or side-effect
execution conformance is claimed. Add new role/profile criteria before any
of these features is introduced.
