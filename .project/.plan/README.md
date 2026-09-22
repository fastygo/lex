# LeX implementation plan

Status: canary release. Normative semantics live in
[the specification](../.lex/README.md); this folder owns the deployment
profile, durable decisions, and release evidence.

## Fixed scope

- Go REST service on `github.com/fastygo/framework` `v0.3.0` with
  `github.com/fastygo/context` `v0.1.0`, capability `memory-exact-v1`.
- One Vercel serverless handler; request-scoped RAM only.
- No database, disk, cache, object store, queue, or server-side history.
- Decisions and replay only; execution is outside this release.
- Provider-neutral typed decisions; no SDK or JavaScript runtime.

## Files

1. [architecture.md](architecture.md): request path, packages, deployment profile, budgets.
2. [decisions.md](decisions.md): accepted architecture decisions.
3. [progress.md](progress.md): canary baseline and gates open until stable.
4. [conformance-report.md](conformance-report.md): revision-pinned canary evidence and rollback point.
