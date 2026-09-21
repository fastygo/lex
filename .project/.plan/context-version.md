# Context dependency baseline

Status: published upstream dependency verified on 2026-09-21.
This records upstream capability, not completion of the LeX implementation.

## Pin

- Module: `github.com/fastygo/context`.
- Tag: [v0.1.0](https://github.com/fastygo/context/tree/v0.1.0).
- Annotated tag object: `dac616c38efe448573d84ccc0ca65096a8f899ea`.
- Resolved commit: `0715ad1ddbc1f749f8e0f0ac1864218eb09a2d13`.
- Embedded package: `github.com/fastygo/context/pkg/contextkit/runtime`.
- Capability version: `memory-exact-v1`.
- Module Go directive: `1.25.0`; upstream local tests used Go `1.25.5`.

When the LeX Go module is introduced, require this exact version and commit
go.sum. Do not use latest, a moving branch, or a release-local replace directive.
The Framework dependency still needs its own exact version selection and
deployment proof; the Context tag does not pin Framework.

## Module checksum verification

`go mod download -json github.com/fastygo/context@v0.1.0` succeeded from an
isolated module cache and resolved the commit recorded above. Expected go.sum
entries for the future LeX module:

```text
github.com/fastygo/context v0.1.0 h1:niqha0Uh29cOeh9axL3g3CTNydixF0oLZSGITvJTSn0=
github.com/fastygo/context v0.1.0/go.mod h1:XzFWD43u/SEYS4F0lDn0CwPg1fSvOnNMVB+YN0IKcKk=
```

These authenticate the downloaded module against the recorded dependency
baseline, not runtime evidence, source truth, or LeX policy authority.

## Public integration

Use `runtime.New(ctx, Config)` to freeze one project and a bounded list of
versioned source texts. Call `Runtime.ContextPack(ctx, PackRequest)` with the
reviewed focus, query, instructions, and policy references. Export PackResult
and its Snapshot with the exact request in the LeX replay bundle.

`Runtime.Search` supports the exact mode. `Runtime.Snapshot` returns an
independent source manifest. The parent `contextkit.Client` remains the
existing HTTP client; the RAM-only LeX profile does not need it.

The host authenticates the project and assigns source trust/evidence classes
from verified provenance. Runtime labels and policy references are not authority.
LeX still validates source bindings, policy, typed decisions, and verdicts.

## Supported scope and limits

- Immutable project/snapshot, original UTF-8 text and byte spans, source checksums.
- Case-sensitive exact phrase search; one complete source per chunk.
- Existing Context pack builder with trust/class gates and rejected matching items.
- Copied inputs and independent result slices/JSON; no network, disk, database,
  model call, worker, or server-side history in the embedded path.
- At most 128 sources; configurable encoded-input ceiling no greater than 2 MiB.
- Positive bounded pack budgets; MaxChars counts UTF-8 bytes.
- No dense, sparse, morphology, arbitrary file parsing, or full CLI parity.

Start LeX with a 256 KiB Context input ceiling and the smaller reference workload
in [slo.md](slo.md). The 2 MiB upstream input ceiling is not a bound on response
size or resident RAM. Packs, rejected entries, source snapshots, and raw provider
answers must together fit the LeX response and later replay request. Fail
admission before provider execution when that cannot be guaranteed.

## Hash and replay boundary

Preserve the upstream snapshot/request identity and ADR-0020 pack checksum.
These are versioned Context hash contracts, not RFC 8785 JCS.
LeX hashes its own declared envelopes under its canonicalization profile and
retains the original upstream digests as separate fields.

LeX verdict replay consumes a frozen saved pack and DecisionSet without
retrieval. Reconstructing a Context pack from retained Snapshot + PackRequest
is a separate optional evidence-construction check; never replace the saved
pack silently during verdict replay.

## Existing evidence and remaining gates

Upstream [usage](../../../@Context/docs/embedded-runtime.md),
[ADR-0045](../../../@Context/docs/decisions/0045-embedded-memory-runtime.md),
and [verification report](../../../@Context/.proofs/embedded-runtime-v0.1.0.md)
record full default offline tests, public package tests, vet, core-builder
parity, and a separate external Go consumer. These local links assume sibling
checkouts; the tag and commit above identify the authoritative artifact.

Concurrent tests passed without race instrumentation. The recorded environment
could not run the race detector. LeX still needs a supported race runner,
a real Framework + Context composition test, deployed Vercel smoke checks,
bundle-size and memory measurements, and its own conformance/calibration work.
No Vercel compatibility or SLO achievement follows from this tag alone.
