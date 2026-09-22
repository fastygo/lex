# ADR-0013: Generic typed-decision API

Status: accepted. Date: 2026-09-22. `/v1/decisions` is implemented with
envelope `0.2` and proven on a revision-pinned canary. Hosted-adapter
deployment proof is a stable-release gate in [progress.md](../progress.md).

## Context

The current `/v1/evaluations` route is a vertical claim-validation slice. It
owns a fixed profile, fixed Context focus, claim-shaped state, policy
thresholds, and semantic verdicts. Those are useful consumer behavior, but
they cannot be the reusable protocol boundary for an orchestration agent that
supplies different state and different Jev questions for each step.

## Decision

Add `/v1/decisions` as the canonical API. Its input is a caller-owned JSON
state and caller-owned versioned QuestionSet, plus an optional frozen Context
binding. LeX validates and hashes the input, calls a pinned typed-decision
adapter, structurally validates the raw DecisionSet, and returns a replayable
bundle. It does not select questions, retrieve, rewrite state, apply domain
thresholds, derive a semantic verdict, or authorize a subsequent action.

Keep `/v1/evaluations` and legacy replay bundles through one deprecation
window as the claim-validation compatibility adapter. That adapter alone owns
the current profile, source-freezing convenience, policy gate, and semantic
verdict.

## Consequences and alternatives

The orchestration agent owns question meaning, thresholds, sequencing, and
the choice to clarify, invoke Context, call a tool, or make another decision
request. Context Runtime remains an optional upstream evidence producer:
LeX can verify a supplied frozen state but never merges it into the caller
state implicitly.

This avoids a universal profile registry, automatic profile selection, and
domain vocabulary in protocol core. It requires a new versioned wire contract
and generic conformance evidence; the existing claim-validation canary does
not prove that contract.

## Acceptance evidence

The generic path has independent Noul, Choice, and Score fixtures over
unrelated states, optional Context-binding fixtures, network-free replay,
direct and hosted adapter conformance, and a revision-pinned production
sample. Legacy evaluation fixtures remain unchanged through the compatibility
window.

## Links

[ADR register](README.md) | [Protocol](../../.lex/protocol.md) |
[Integration stack](../../.lex/integration-stack.md)
