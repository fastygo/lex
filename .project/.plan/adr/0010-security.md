# ADR-0010: Authority, isolation, and safe transport

Status: proposed. Date: 2026-09-21. Project binding and transport limits are implemented.
Vulnerability review is deferred. See [progress.md](../progress.md).

## Context

Statelessness does not make caller assertions trustworthy. Memory and network budgets are security boundaries.

## Decision

Bind authenticated principals to allowed projects and deployment-owned policies. Use TLS, explicit input limits, endpoint allowlists, and no arbitrary URL dereferencing. Keep secrets in platform configuration, never requests/bundles/logs. Disable browser-cookie auth/CSRF surfaces for the bearer-auth API; CORS is deny-by-default unless explicitly configured. Define a narrow authentication profile before exposure.

## Consequences and alternatives

No claim of built OIDC, PKI, signed receipts, or global rate limiting. Local concurrency protection cannot enforce cross-instance quotas; platform controls or per-request budgets must cover that deployment concern. A checksum is not approval, and replay cannot mint authority.

## Acceptance evidence

Access-control and project-isolation tests; caller-policy downgrade, forged approval, prompt injection, SSRF, oversized/recursive JSON, leaked secrets, and concurrent tenant attacks. Record TLS/auth configuration, credential rotation behavior, and dependency vulnerability review.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
