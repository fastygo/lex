# LeX verification infrastructure

Status: canonical working draft, not a released interoperability contract.

LeX turns evidence and uncertain judgments into governed, inspectable,
replayable outcomes. Its protocol defines contracts; implementations supply
schemas, adapters, policy enforcement, verification, and execution controls.

## Reading order and authority

1. [concept.md](concept.md): purpose, vocabulary, and boundaries.
2. [protocol.md](protocol.md): normative draft for entities, invariants, lifecycle, and replay.
3. [checks.md](checks.md): normative draft for checks, verdicts, and errors.
4. [integration-stack.md](integration-stack.md): normative draft for integration boundaries and adapter obligations; examples are informative.
5. [generic-decision-api.md](generic-decision-api.md): additive `0.2-draft`
   caller-owned State + QuestionSet API.
6. [scope.md](scope.md): system boundary and scope constraints.
7. [analysis.md](analysis.md): informative implementation plan and release blockers.
8. [VSA guidance](../.vsa/README.md): subordinate implementation architecture.
9. [sources.md](sources.md): references and applicability.

`protocol.md` owns shared semantics. Checks and integration contracts refine
them without overriding them. Conceptual explanations, plans, research,
examples, and illustrations cannot weaken normative requirements. Conflicting
canonical text is a defect to resolve before release, not an implementation
choice. See [governance.md](governance.md) for change control.

## Current deployment plan

The [implementation plan](../.plan/README.md) fixes Go REST, Framework + Context,
Vercel, and request-scoped RAM with caller-owned replay bundles. TypeScript,
databases, and server-side history are outside the current scope. Embedded pack
construction uses [Context v0.1.0](../.plan/context-version.md), capability
memory-exact-v1, under [ADR-0003](../.plan/adr/0003-context-boundary.md).

## Generic typed-decision slice

```text
caller State + caller QuestionSet + optional frozen Context binding
  -> typed-decision adapter -> retained raw DecisionSet
  -> structural verifier -> DecisionTrace + replay bundle
```

The caller interprets answers and decides whether to ask again, use Context,
call a tool, or stop. LeX does not own that orchestration.

## Compatibility validation slice

```text
EntityEnvelope + ValidationIntent
  -> frozen PolicySnapshot + SemanticProfile + FocusProfile + QuestionSet
  -> Context Runtime -> frozen ContextPack + EvidenceBinding
  -> preflight -> typed-decision adapter -> retained raw DecisionSet
  -> deterministic verifier -> Verdict + EvaluationTrace
```

Controlled execution is a separate extension: explicit authorization, an
OperationContract, execution, postcondition verification, and a Receipt.
A validation verdict alone does not prove that an operation occurred.

## Current proof boundary

Research contains hand-authored inputs and response captures. It does not
establish real Context retrieval quality or LeX conformance. Versioned wire
schemas, a verifier, release fixtures, a transport binding, and independent
interoperability results remain work tracked in [analysis.md](analysis.md).
Optional root tooling can scan for English-only text. The deployment-specific
ADR register, SLO targets, and acceptance criteria live in [the plan](../.plan/README.md).
