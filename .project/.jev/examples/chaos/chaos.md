# Context Runtime × Jev chaos test

This test compares four representations of the same customer incident. Evaluate each case independently and use only the evidence listed inside that case.

Evidence classes:

- `source_text`: original user or policy text.
- `tool_output_authoritative`: a verified result from an authoritative system.
- `model_inference`: a generated hypothesis that may guide retrieval but cannot independently establish factual support.

Routing criteria are intentionally overlapping:

- `billing`: payment failures, duplicate charges, refunds, or loss of service immediately after payment.
- `account_access`: sign-in failures, authentication errors, locked accounts, or access restoration, including failures after payment.
- `insufficient_or_ambiguous`: the available evidence does not distinguish the routes, or authoritative evidence supports more than one route.

Decision semantics:

- A route is `supported` when at least one eligible evidence item directly supports it. More than one route may be supported at the same time.
- A fact is `established` only when the eligible evidence is sufficiently coherent and authoritative; one disputed record does not settle a conflict.
- Automatic routing is safe only when coherent evidence supports one route and no unresolved authoritative conflict supports another.
- The final action must be `route_billing`, `route_account_access`, or `manual_review`. `manual_review` is an operational decision, not a claim that no route has any support.

## cases.raw

- `customer_message` (`source_text`): “I paid yesterday. Today I cannot sign in. Please fix this before my meeting.”

## cases.llm_augmented

- `customer_message` (`source_text`): “I paid yesterday. Today I cannot sign in. Please fix this before my meeting.”
- `incident_summary` (`model_inference`): “The customer was probably charged twice and is implicitly requesting a refund.”

## cases.context_resolved

- `customer_message` (`source_text`, trusted): “I paid yesterday. Today I cannot sign in. Please fix this before my meeting.”
- `payment_record` (`tool_output_authoritative`): exactly one successful charge exists; no duplicate charge, reversal, or failed payment exists.
- `authentication_log` (`tool_output_authoritative`): the account was locked after repeated failed sign-in attempts.
- `account_runbook` (`source_text`, trusted): verified account lockouts are routed to account access support.
- `refund_policy` (`source_text`, trusted): a refund requires an explicit refund request, a failed charge, or a confirmed duplicate charge.

## cases.context_conflicted

- `customer_message` (`source_text`, trusted): “I paid yesterday. Today I cannot sign in. Please fix this before my meeting.”
- `payment_ledger_primary` (`tool_output_authoritative`): two captured charges with the same order identifier are present.
- `payment_ledger_replica` (`tool_output_authoritative`): exactly one captured charge is present; replication status is delayed.
- `authentication_log` (`tool_output_authoritative`): the account is locked after repeated failed sign-in attempts.
- `billing_policy` (`source_text`, trusted): confirmed duplicate charges are routed to billing and are eligible for review.
- `account_runbook` (`source_text`, trusted): verified account lockouts are routed to account access support.
- `data_quality_notice` (`source_text`, trusted): when authoritative payment systems disagree, do not assume which ledger is current; request reconciliation.
