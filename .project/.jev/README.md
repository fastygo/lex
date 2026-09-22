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
- [`lex-protocol-history.md`](lex-protocol-history.md) — historical extended protocol notes, preserved for research history; not a maintained mirror.

## Experiments

- [`examples/playground/`](examples/playground/) — checks of Jev’s claimed properties and primitives.
- [`examples/llm/`](examples/llm/) — typed questions on LLM topics.
- [`examples/context/`](examples/context/) — Jev as a hypothetical decision plugin for Context Runtime.
- [`examples/chaos/`](examples/chaos/) — ambiguous, noisy, resolved, and conflicting evidence sets.

## Current status

The files under `examples/` are the original Jev question maps and captured
responses. They are not `POST /v1/evaluations` bodies. The same scenarios as
LeX evaluation requests are in
[`test-vercel/requests/`](test-vercel/requests/). Each body is a project,
a `claim` entity of schema `0.1`, an exact-phrase query, and versioned source
text. The service asks the embedded `claim-validation` `0.2.0` profile:
`support`, `established`, `refuted`, `conflict`, `safe_to_auto_act`, and one
action Choice.

Research supports the viability of the contract:

```text
Entity
  -> ValidationIntent + Policy
  -> embedded Context Runtime / ContextPack
  -> frozen QuestionSet
  -> typed Noul / Choice answers
  -> deterministic verification
  -> LeX Verdict + caller-owned replay bundle
```

These notes record decision-model behavior on manually prepared states. They
are not a calibration corpus or a conformance certification. The captured
verdicts under `test-vercel/evaluations/` are the `2026-09-21T22:14:57Z`
sample of profile `0.2.0`.
