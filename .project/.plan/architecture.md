# Architecture and Vercel profile

Status: proposed implementation architecture under the fixed scope in [README.md](README.md).

## Request path

```text
REST caller: entity + intent + (versioned source texts | Context frozen state)
  -> Framework HTTP middleware
  -> authentication + project binding + bounded decoding
  -> evidence input: freeze text through Context v0.1.0, or accept a frozen
     state and have Context reproduce it
  -> typed-decision adapter -> frozen DecisionSet
  -> deterministic verifier
  -> Verdict + VerificationReport + EvaluationTrace + replay bundle
```

Replay uses the caller-retained bundle. It does not contact a retrieval service
or a decision provider. It asks the embedded Context runtime to rebuild the
snapshot and pack from the frozen sources and pack request in the bundle and
compares that result with the saved state. The saved pack is not replaced, and
LeX does not recompute selection, budgeting, or rejection itself.

Profiles are objects, not packages: a profile pins its question set, policy
document and gate, entity kind, and Context focus, and hashes them. The
deployment composes a registry of profiles; live evaluation uses the first,
and replay resolves the profile a bundle names by question-set and policy
reference. The verifier stays generic over profiles.

Core logic uses Go domain types and explicit ports. Framework owns HTTP
composition. Context public types and compatibility checks stay in the evidence
adapter. Provider details stay in decision adapters. No upstream internal imports.

Deployment-owned policy and profiles are immutable embedded configuration or
validated configuration loaded at initialization. Caller-provided policy is
data to verify against authenticated authority, never permission to lower gates.
Request-local maps and byte buffers own all entity, evidence, and answer state.

## Embedded Context contract

Use the [pinned Context baseline](context-version.md), capability memory-exact-v1.
Create one runtime from approved source inputs per evaluation and call its
ContextPack method with explicit focus controls. Map the supported profile
subset without silently dropping requirements. The runtime performs exact
phrase retrieval only; it is not a dense or morphology-enabled runtime.

Export Snapshot and exact PackRequest alongside the frozen pack. Preserve its
upstream checksum separately from the LeX canonical envelope hash. Source trust
and evidence classes come from authenticated provenance mapping, not unchecked
caller labels. HTTP contextkit.Client is not on the RAM-only execution path.

## Vercel integration

Use one Go HTTP application, exposing the same `http.Handler` locally and on
Vercel. Prefer the Go framework preset and `cmd/api/main.go`, listening on
`PORT`. A thin `api/*.go` handler is an alternative only if the deployment
probe selects it; do not maintain two independent routing implementations.

Current [Vercel Go documentation](https://vercel.com/docs/functions/runtimes/go)
describes both modes, Go/toolchain selection from root go.mod, and Beta status.
Pin exact module revisions and a supported toolchain after a real build probe.
Do not infer Go support from Node.js/Fluid examples.

Framework's public `App.Handler()` permits composition. Disable static assets,
HTML redirects, locale behavior, browser authentication flows, and background
workers unless needed by the API. Test middleware response formats and caps.
No shutdown hook or work continuing after the response may be needed for correctness.

Warm instances may reuse immutable configuration and concurrency-safe HTTP
transports. They must not cache mutable protocol artifacts or authoritative
per-caller state. Do not assume requests reach the same instance.

## RAM ownership and replay

Freeze and retain raw answers in request RAM before interpretation. Return the
full replay material before a successful response ends. Bundle contents include
original evidence needed by bindings, pinned profiles/policy/questions,
DecisionSet, verifier identity, findings, and hashes. Avoid duplicating payloads.

Replay supplies the bundle in a new request; it neither retrieves evidence nor
calls a model. A digest detects tampering relative to an expected value but
does not authenticate who issued a bundle. Replay reports reproducibility;
it does not grant current authority or prove historical server issuance.

There is no GET-by-run-id history, durable 202 job, resumable upload, global
deduplication, or exactly-once guarantee. Lost responses cannot be recovered
from the server. A new evaluation may call a provider again and incur cost.
Client disconnects cancel work where possible but cannot recall an accepted
provider request.

## Memory-only interpretation

LeX application data uses RAM, with no writes to /tmp or mounted storage.
Build artifacts and read-only embedded schemas are allowed. Platform access
logs and provider-side retention are outside application RAM ownership;
document their actual settings and never put evidence, credentials, or raw
answers into telemetry. Do not claim zero retention by Vercel or providers.

## Bounded execution

Apply [SLO budgets](slo.md) to decoded input, provider responses, bundle size,
concurrency, and time. Validate the maximum possible response budget before
calling a provider; reject excessive upstream output. Never truncate evidence
or omit required raw answers to produce a success-shaped response.
