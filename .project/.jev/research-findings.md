# Jev research findings

Status: non-normative empirical observations on `jev-1.13.0`, not a general benchmark and not proof of the model’s internal architecture.

The scenarios were hand-authored Jev question maps (`playground`, `test` on LLM
topics, `context`, `chaos`) sent directly to the model before LeX existed.
Their captures are retired; git history keeps them. Measurements are research
observations, not independent reproduction or LeX conformance proof.

## Summary

Jev is useful as a fast **typed decision engine** over a pre-formed state. On narrow questions with non-overlapping criteria it returns sharp, practically usable distributions. Quality drops noticeably when:

- a question combines several judgments;
- `Choice` options semantically overlap;
- state mixes evidence and model-generated hypotheses;
- the model is expected to both establish facts and choose an action at once.

The best pattern found is to separate:

1. support for each hypothesis;
2. sufficiency and coherence of evidence;
3. safety of automatic action;
4. the final operational action.

## Run results

| Set | Questions | Time | Observation |
|---|---:|---:|---|
| `playground` | 16 | 295.0 ms | Basic definitions mostly correct; one Noul wrong direction, several answers uncertain |
| `test` | 9 | 142.5 ms | All LLM answers matched expectations; clear atomic questions gave sharp distributions |
| `context` | 9 | 129.2 ms | ContextPack → Jev → verification contract recognized without errors |
| `chaos`, first version | 12 | 89.6 ms | Resolved pack gave exact route, conflicted pack abstained; one Choice hid support for individual hypotheses |
| `chaos`, split version | 20 | 98.2 ms | Independent support signals, safety gate, and action Choice preserved support, conflict, and final action together |

Times are from one request per scenario. They show fan-out feasibility, not a latency benchmark.

## What observations support

### 1. Atomicity matters more than topic complexity

In the LLM test Jev clearly distinguished:

- autoregressive generation;
- RLHF preference optimization;
- structured output via constrained/validated generation;
- unreliability of confidence written by the model in prose;
- LLM as sole gate on destructive action being unacceptable.

Almost all distributions peaked on the expected answer. That was not because the topic was easy, but because each question tested one claim.

### 2. Typed output does not guarantee correct judgment

In `playground`, `questions_evaluated_in_parallel` got `noul = 0.32` although state asserted the opposite. The answer schema stayed valid, but the substantive decision was wrong.

Implication: shape guarantees and no values outside `criteria` are not guarantees of factual accuracy.

### 3. Overlapping Choice blurs the distribution

When two criteria describe similar meanings, probability splits between them. A winner does not remove ambiguity. `confidence` is useful as a distribution-shape indicator but does not explain the source of conflict.

Implication: criteria must be mutually distinguishable by definitions and evidence. If not, use `manual_review` or separate Noul per hypothesis.

### 4. Model inference adds noise even without authority to establish facts

In the chaos test, LLM summary could not establish a billing fact but lowered confidence on the operational Choice:

- raw `manual_review`: confidence `0.95`;
- LLM-augmented `manual_review`: confidence `0.68`.

Implication: exclude `model_inference` from the decision pack unless a specific question needs it. Evidence-class labeling reduces risk but does not make noise neutral.

### 5. A coherent ContextPack sharpens the decision

For `context_resolved`:

- billing supported: `0.04`;
- account access supported: `0.98`;
- safe to auto-route: `0.94`;
- action: `route_account_access` with probability `1.0`;
- evidence sufficiency: `3.0` with confidence `1.0`.

This is the expected shape for automation: one supported hypothesis, high safety signal, and unambiguous action.

### 6. A conflicted ContextPack should preserve multiple truths

For `context_conflicted`:

- billing supported: `0.69`;
- account access supported: `0.98`;
- safe to auto-route: `0.03`;
- action: `manual_review` with probability `0.99`;
- evidence sufficiency: `1.99`, almost all mass on “meaningful but conflicting”.

Independent Noul values need not sum to 1. That is useful: several incompatible actions can be supported at once. A normalized Choice over hypotheses would lose that information.

### 7. Support, truth, and action are different entities

Observing `billing supported = 0.69` with conflicting ledgers shows Jev partly conflates:

- “eligible evidence exists for the hypothesis”;
- “the hypothesis is established by coherent evidence”.

Callers should ask separate predicates:

- `supported`: eligible witness exists;
- `established`: evidence is sufficient and coherent;
- `safe_to_act`: model-reported safety signal; the caller's policy and external authority still decide whether action is allowed;
- `action`: chosen operational branch.

## Recommended question design

### Noul

Use for independent predicates:

- is the claim supported;
- is the fact established;
- is there conflict;
- is evidence sufficient;
- is automatic action safe.

Do not combine two predicates with `and` in one question if the trace needs them separately.

### Choice

Use for a single operational switch:

- `route_billing`;
- `route_account_access`;
- `manual_review`.

Add `other` for incomplete taxonomies. Add explicit `manual_review` when auto-choice is unsafe, but do not use it instead of independent diagnostics of supported hypotheses.

### Score

Use only for ordered scales:

- evidence sufficiency;
- severity;
- urgency;
- policy risk.

Do not use Score for unordered categories.

## Not yet proven

- That Context Runtime automatically builds packs of the same quality as manual scenarios.
- That Jev probabilities are calibrated on LeX domain data.
- That results are stable across repeats and model versions.
- That universal thresholds apply across entity types and risk levels.
- That high `confidence` means high factual accuracy without external verification.
- That Jev’s internal architecture matches all public marketing claims.

## Next measurements

1. Build the frozen Context state with Context Runtime, not by hand.
2. Build a labeled corpus with expected answers per question.
3. Compare raw state, plain RAG, and ContextPack on the same questions.
4. Repeat each request and measure consistency.
5. Measure calibration: accuracy by Noul/confidence bands.
6. Separately score retrieval error, question design error, model decision error, and verification error.
7. Pin model version when tuning thresholds.
