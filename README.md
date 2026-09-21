# LeX

LeX is subject-neutral operational infrastructure for turning evidence and
uncertain judgments into governed, inspectable, reproducible outcomes.

Its central rule is:

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

LeX combines:

- **Context Runtime** as the evidence plane;
- provider-neutral typed decisions as the judgment plane;
- deterministic policy and verification gates;
- verdicts, traces, replay bundles, and conformance evidence.

Jev is the current reference typed-decision model. Provider APIs are adapters,
not protocol identities.

## Status

LeX is a **v0.1 working draft and implementation plan**. No released protocol,
conformance claim, achieved SLO, or production deployment is claimed.

The fixed first implementation profile is:

- Go REST service using Framework and
  [`github.com/fastygo/context@v0.1.0`](.project/.plan/context-version.md);
- embedded Context capability `memory-exact-v1`;
- deployment target: Vercel Go runtime;
- request-scoped RAM only for mutable protocol data;
- synchronous validation and deterministic replay;
- caller-owned replay bundles;
- no database, disk persistence, durable queue, server-side run history, or
  side-effect execution;
- no TypeScript SDK or JavaScript application runtime in v0.1.

Context v0.1.0 and Framework v0.3.0 are pinned. The local service freezes a
Context pack, requests one embedded question set through a direct or hosted
System One adapter, applies an uncalibrated policy, and returns a verdict plus
a caller-owned replay bundle. Replay reproduces that verdict without contacting
a retrieval service or a provider. It recomputes the Context pack from the
frozen snapshot already in the bundle and does not replace the saved pack. A draft OpenAPI document and local adversarial tests exist.
Calibration, a 28-day SLO, race evidence on a supported runner, and proof of
the latest deployment revision remain open. No conformance certification is claimed.

## Architecture

```text
EntityEnvelope + ValidationIntent
  -> frozen PolicySnapshot + profiles + QuestionSet
  -> Context Runtime -> frozen ContextPack + EvidenceBinding
  -> preflight checks
  -> typed-decision adapter -> raw DecisionSet
  -> deterministic verifier
  -> Verdict + VerificationReport + EvaluationTrace + replay bundle
```

Replay consumes a caller-retained frozen bundle. It does not contact a retrieval
service or invoke a decision provider. A recomputation from the frozen snapshot
checks the saved pack and does not replace it.

Controlled execution is deliberately outside the first slice. A validation
verdict does not prove that an operation occurred.

## Protocol model

Core protocol objects:

- `EntityEnvelope` — the versioned object under validation;
- `ValidationIntent` — the question being resolved;
- `ContextPack` and `EvidenceBinding` — frozen admissible evidence;
- `SemanticProfile` and `QuestionSet` — versioned meanings and typed questions;
- `DecisionSet` — unmodified provider output with resolved identity;
- `PolicySnapshot` — frozen thresholds, risk, and authority rules;
- `VerificationReport` — deterministic contract checks;
- `Verdict` — final validation disposition;
- `EvaluationTrace` — replayable record;
- `Receipt` — proof of a separately authorized and verified operation.

LeX preserves these verdicts:

```text
validated | rejected | insufficient | conflict | manual_review | error
```

`insufficient` and `conflict` are epistemic outcomes.
`manual_review` is an operational disposition.

## Typed question design

The reference decision contract uses:

- independent **Noul** questions for support, establishment, conflict, and
  safety predicates;
- one **Choice** for a mutually exclusive operational recommendation;
- **Score** only for a genuinely ordered rubric.

```text
support Noul x N
established/conflict Noul
safe_to_auto_act Noul
one action Choice
optional ordered Score
```

A decision is never evidence for its own verdict, and confidence never grants
authority.

## Planned REST profile

The current plan proposes:

```text
POST /v1/evaluations
POST /v1/replays
GET  /v1/capabilities
GET  /healthz
```

The wire profile uses JSON Schema 2020-12, RFC 8785 JCS with SHA-256 for LeX
canonical hashes, and RFC 9457 Problem Details. A draft OpenAPI 3.1.1 document
is at `internal/wire/schema/openapi.json`. The handler enforces that evaluation
schema. Routes are not a released protocol until the ADRs are accepted.

## Delivery path

1. **P0 — feasibility:** pin Framework, compose the Go handler with Context,
   and prove the Vercel deployment profile.
2. **P1 — wire and replay:** schemas, OpenAPI, hash vectors, verdict precedence,
   verifier, and network-free replay.
3. **P2 — evidence and decisions:** real Context packs, two decision-adapter
   paths, adversarial cases, and calibration evidence.
4. **P3 — operational proof:** security, isolation, concurrency, resource
   budgets, deployment probes, SLO measurements, and conformance report.

See the complete [delivery and proof sequence](.project/.plan/delivery.md).

## Documentation

Start here:

1. [Canonical specification](.project/.lex/README.md)
2. [Concept and boundaries](.project/.lex/concept.md)
3. [Protocol entities and lifecycle](.project/.lex/protocol.md)
4. [Checks, verdicts, and errors](.project/.lex/checks.md)
5. [Implementation plan](.project/.plan/README.md)
6. [Architecture and deployment profile](.project/.plan/architecture.md)
7. [ADR register](.project/.plan/adr/README.md)
8. [Conformance gates](.project/.plan/conformance.md)
9. [SLOs and resource budgets](.project/.plan/slo.md)

Additional material:

- [VSA and ICOM guidance](.project/.vsa/README.md)
- [Jev research](.project/.jev/README.md) — non-normative captured evidence
- [Project documentation map](.project/README.md)

When documents disagree, `.project/.lex/` owns protocol semantics.
`.project/.plan/` owns implementation and deployment planning.

## Optional tooling

Human-readable content outside `.manual/` is English only. Optional:

```bash
npm run check:english
```

Current foundation checks:

```bash
go test ./... -count=1
go vet ./...
```

The protocol schemas, OpenAPI description, and conformance suite remain work in progress.

## Local API

The local handler exposes `GET /healthz`, authenticated `GET /v1/capabilities`,
`POST /v1/evaluations`, and `POST /v1/replays`. Evaluation accepts a project,
an entity of type `claim` with schema `0.1`, an exact-phrase query, and
versioned source texts. The server assigns source trust and evidence class,
freezes the pack, and applies the embedded `claim-validation` policy. That
policy is explicitly uncalibrated. An empty exact retrieval returns
`insufficient` and does not call a provider. A technical `error` verdict is
HTTP 422 and still returns the sealed replay bundle.

Configure bearer tokens outside version control. Set one decision credential,
or set `LEX_DECISION_ADAPTER` to `direct` or `hosted` when both are present:

```bash
export LEX_BEARER_TOKENS='{"development-token":["example-project"]}'
export LEX_TYPESAFE_API_KEY='replace-with-secret'
go run ./cmd/api
```

`LEX_BEARER_TOKENS` and provider credentials are deployment secrets. Never add
them to source files, request payloads, traces, replay bundles, or logs.

## License

Licensed under the [Apache License 2.0](LICENSE).
