# LeX concept

Status: informative. Normative text is [protocol.md](protocol.md).

## Purpose

LeX is subject-neutral infrastructure that turns an uncertain judgment into an
inspectable, reproducible record. A caller asks typed questions about its own
state; LeX obtains the answers from a pinned typed-decision model, checks their
structure deterministically, and returns a sealed bundle the caller can replay.

```text
evidence != judgment
judgment != authority
authority != execution
execution != verified success
```

LeX covers the judgment step and its proof. Evidence comes from the caller or
from Context Runtime. Authority, action, and success belong to the caller.

## A switch, not a router

LeX is a typed-decision switch. It has no domain profiles, no question
catalogue, no thresholds, and no verdicts. An orchestration agent writes the
State and QuestionSet for each step, reads the answers, applies its own
thresholds, and decides whether to ask again, fetch more context, call a tool,
escalate, or stop. Adding a new use case is new caller data, not LeX code.

## Primitives

| Primitive | Answers | Use |
|-----------|---------|-----|
| Noul | probability of yes | one independent predicate: support, conflict, safety, presence |
| Choice | one option + probability per option | one mutually exclusive selection: route, intent, action |
| Score | position on ordered levels | a genuinely ordered scale: urgency, severity, sufficiency |

Question design, owned by the caller:

- one atomic claim per question;
- independent Nouls need not sum to one;
- add `other` to an open taxonomy and a non-action option such as
  `manual_review` to a risky Choice;
- do not use one Choice both to discover hypotheses and to authorize an action;
- put facts in State or a frozen Context, not in instructions;
- probabilities are uncalibrated model output, never authority.

## Functional contract (ICOM)

| Role | For one decision |
|------|------------------|
| Input | caller State and optional frozen Context |
| Control | QuestionSet, schemas, pins, budgets, project authorization |
| Mechanism | typed-decision adapter, Context rebuild, structural verifier |
| Output | DecisionSet, structural report, trace, replay bundle |

Mechanisms may change; they cannot weaken Control. Three boundary tiers apply:
deterministic (schema, hash, budget, pins), semantic (typed answers over the
supplied state), and authoritative (credentials, approvals, human decisions).
LeX implements the first, carries the second, and never supplies the third.

## Context Runtime

Context owns retrieval, selection, budgeting, rejection, and ContextPack
construction. A caller that needs evidence builds a frozen state with Context
and passes it with the decision. LeX asks the pinned Context runtime to rebuild
that state and compares; it never retrieves or recomputes selection itself.

## Providers

Jev is the reference decision model. It is reachable through the direct
System One API and through OpenRouter; both are adapters. Provider names,
endpoints, keys, aliases, and billing stay inside adapters and never enter
protocol objects or answer semantics.
