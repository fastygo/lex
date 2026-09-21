# LeX: Context Runtime × Jev research

Status: non-normative research. Examples are hand-authored scenarios and response captures, not released LeX conformance fixtures.

**Canonical specification:** [`.project/.lex/README.md`](../.lex/README.md)

The project goal is to develop the **LeX** protocol for checking and validating various entities in tandem:

- **Context Runtime** builds auditable context: selects evidence, preserves provenance, trust, spans, versions, and policy constraints;
- **Jev** performs fast typed judgments over that context via `Noul`, `Choice`, and `Score`;
- **LeX** connects the evidence plane and the decision plane into a reproducible validation protocol.

LeX must not treat the model’s answer as the source of truth. Its job is to specify which evidence is admissible, which questions are asked, how uncertainty is interpreted, when a decision is allowed, and how the result is verified and replayed.

## Documents

- [`../.lex/`](../.lex/README.md) — checks, protocol, analysis (normative for this repo).
- [`research-findings.md`](research-findings.md) — observations from completed experiments and limits of the conclusions.
- [`context-runtime-role.md`](context-runtime-role.md) — the role of Context Runtime in the Jev integration.
- [`lex-protocol-draft.md`](lex-protocol-draft.md) — historical extended draft, preserved for research history; not a maintained mirror.

## Experiments

- [`examples/playground/`](examples/playground/) — checks of Jev’s claimed properties and primitives.
- [`examples/llm/`](examples/llm/) — typed questions on LLM topics.
- [`examples/context/`](examples/context/) — Jev as a hypothetical decision plugin for Context Runtime.
- [`examples/chaos/`](examples/chaos/) — ambiguous, noisy, resolved, and conflicting evidence sets.

## Current status

Research supports the viability of the contract:

```text
Entity
  → ValidationIntent + Policy
  → Context Runtime / ContextPack
  → Jev typed questions
  → support + uncertainty + action signals
  → deterministic verification
  → LeX Verdict + replayable trace
```

This is observation of decision-model behavior on manually prepared states, not confirmation of a complete LeX implementation. The next level of proof is to obtain the same `ContextPack` through the real Context Runtime API and measure quality on a labeled corpus.
