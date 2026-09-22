# Architecture decisions

Status: all decisions below are accepted and implemented for the canary.
Evidence still owed for stable is in [progress.md](progress.md). A new
decision starts as proposed and is accepted only with review and linked tests;
supersede an entry instead of silently editing it.

## 0001 Go, Framework, and dependency direction

Go REST service; `github.com/fastygo/framework` `v0.3.0` (commit `7cfc0b3`)
for HTTP composition; `github.com/fastygo/context` `v0.1.0` (commit
`0715ad1`) only through public packages. Core semantics carry no framework or
provider SDK types. Open: record the deployed toolchain.

## 0002 Vercel handler lifecycle

One `net/http` handler on the Vercel Go preset. Instances are disposable and
own no protocol state; secrets come from deployment configuration. Open:
record region, duration, bundle size, and cold start.

## 0003 Context boundary

The embedded `pkg/contextkit/runtime` (`memory-exact-v1`) is the only Context
path. Everything Context can do, Context does: LeX decodes a frozen state
strictly into Context's public types, checks project and runtime identity, has
Context rebuild it, and compares by identity and canonical hash. LeX never
retrieves or recomputes selection. Open: race evidence on a gcc runner.

## 0004 Wire schemas

JSON Schema 2020-12 is the wire-shape source, validated with
`github.com/santhosh-tekuri/jsonschema/v6`. Unknown fields are refused; the
only extension area is bounded `metadata` that cannot affect a decision. The
typed bundle record mirrors its schema and `TestDecisionBundleTypeMirrorsSchema`
fails on drift.

## 0005 Canonical hashes

RFC 8785 JCS with SHA-256 through `internal/canonical`. Each hash has an
explicit scope that excludes its own digest; the bundle hash covers all sealed
fields. Independent vectors come from `scripts/jcs_vectors.py` with pinned
`rfc8785`. Hashes detect change; they do not authenticate an issuer.

## 0006 Synchronous REST and error mapping

Synchronous `POST` routes with RFC 9457 problems, a stable reason catalog
matched against OpenAPI, strict content negotiation, and `no-store`. The
mapping is in [checks.md](../.lex/checks.md).

## 0007 Request RAM and caller-owned replay

Inputs and raw answers live in request RAM and return in a complete bounded
bundle. Replay takes the caller's bundle and never retrieves or calls a model.
A lost response is unrecoverable; there is no idempotency store.

## 0008 Lifecycle and retries

A request-scoped state machine renders every trace; illegal sequences cannot
be emitted. Cancellation stops new stages. LeX never retries a provider. A
repeated request is a new decision.

## 0009 Provider adapters

Direct (`https://api.typesafe.ai/v1/systemone`, `jev-1.13.0`) and hosted
(`https://openrouter.ai/api/v1/systemone`, `typesafe/jev-1.13` with an exact
`LEX_HOSTED_RESOLVED_MODEL`) adapters behind one interface. Both declare
primitives before a call, allowlist their endpoint, refuse redirects, bound
reads, pin the resolved model, and never fall back. Callers cannot supply
endpoints, credentials, or models. Open: hosted proof on the deployment and a
per-adapter calibration report.

## 0010 Authority and transport security

Bearer tokens bound to projects; no cookies, CORS, or caller-supplied
authority. Endpoint allowlists, no URL dereferencing, secrets only in platform
configuration. A checksum is not approval and replay mints no authority.
Open: vulnerability review repeated per release.

## 0011 Budgets and overload

Explicit input, output, depth, time, and admission budgets with controlled
overload errors; no in-service metrics store. Open: SLO measurement over the
28-day window.

## 0012 Conformance and release evidence

Every MUST line maps to a test. A canary is declared only with a
revision-pinned probe, replay of each bundle, and a named rollback deployment
in [conformance-report.md](conformance-report.md). No certification is
claimed. Open: stable conformance report.

## 0013 Typed-decision switch

LeX exposes one operation: caller State and QuestionSet in, raw DecisionSet
and replayable bundle out. There are no domain profiles, thresholds, or
verdicts in the core; the claim-validation route that had them was removed
before any consumer depended on it. New use cases are caller data.
