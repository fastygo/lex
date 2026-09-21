# Jev as a Context Runtime decision plugin

Context Runtime is a project-scoped context operating layer. It deterministically indexes source material, performs exact, sparse, dense, hybrid, and morphology-aware retrieval, and builds a budgeted `ContextPack`. A pack separates `instructions` and policy from `evidence_items`, records rejected evidence, and preserves source spans, checksums, trust labels, and provenance. Generated model text is an inference, not source truth.

Jev is integrated here as a hypothetical replaceable decision plugin, not as the context store and not as a prose completer. Context Runtime selects and validates the evidence. Jev receives a narrow `ContextPack` state and asks typed questions about that state. It returns Noul, Choice, or Score answers. Consumer code applies thresholds, policy, and deterministic verification before any action.

The plugin contract is:

1. A `FocusProfile` narrows retrieval to one decision objective and specifies trust, evidence classes, and a context budget.
2. Hybrid retrieval merges and deduplicates candidate spans. Exact terms, sparse matches, dense similarity, morphology, source trust, and citations may contribute to ranking.
3. The pack keeps authoritative source spans and attestations. `model_inference`, lexical analysis, and concept mappings cannot independently justify a factual claim.
4. Choice criteria are built from distinct domain concepts or senses, with explicit definitions and an `other` option when the taxonomy may be incomplete. Two labels that share the same definition or supporting spans are considered an ambiguous taxonomy.
5. Jev evaluates narrow typed questions against the same pack. Its probabilities express uncertainty over the answer space supplied by the consumer; they do not upgrade weak evidence into authoritative evidence.
6. Verification checks that the selected decision has sufficient source support, satisfies policy, and meets the risk-dependent threshold. Missing or conflicting evidence produces an abstention or human review, not a forced classification.
7. Destructive actions such as transfers, refunds, or deletion require an independent policy or human confirmation. Neither an LLM's fluent text nor a high Jev score is sufficient authority by itself.

## Test scenario

An LLM-generated incident report says: “The customer probably wants a refund because the account page failed after payment.” The report is stored as `model_inference`.

The retrieved source evidence contains:

- `customer_message` (`source_text`, trusted): “I paid yesterday. Today I cannot sign in. Please restore access.”
- `payment_record` (`tool_output`, authoritative): the payment succeeded and no duplicate charge exists.
- `refund_policy` (`source_text`, trusted): refunds require an explicit refund request or a confirmed duplicate charge.
- `account_runbook` (`source_text`, trusted): sign-in failures after successful payment are routed to account access support.

The candidate routing criteria are:

- `billing`: charges, invoices, duplicate payments, or explicit refund requests.
- `account_access`: sign-in, authentication, locked-account, or access-restoration issues.
- `other`: none of the listed categories is adequately supported.

The LLM report is useful as a hypothesis, but it cannot justify a refund. The authoritative evidence supports `account_access`. If `billing` and `account_access` had overlapping definitions or were supported by the same indistinguishable spans, the plugin should abstain and request taxonomy refinement rather than hide the ambiguity behind a winning Choice.
