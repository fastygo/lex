# Calibration trial 20260922-0947

Status: observation. Policy `claim-validation` `0.2.0` stays `uncalibrated`. This file does not select new thresholds.

The `safe_to_auto_act` labels follow the gloss in `calibration.md`: an automatic disposition would be safe. The embedded question asks whether the evidence is safe to accept for the claim without a person reading it first. Those are different questions. The three refutation items were labeled `yes` under the gloss and were not relabeled after the answers.

- Labeled at: `2026-09-22T06:47:00Z` (before any answer in this trial)
- Ran at: `2026-09-22T06:51:27.435Z`
- Origin: `https://lexproto.vercel.app`
- Policy: `claim-validation` `0.2.0` `uncalibrated`
- Retrieval: `exact_phrase` / `memory-exact-v1`
- Resolved model on scored rows: `jev-1.13.0`
- Scored trials: 18
- Excluded trials: 0
- A reliability fraction is blank when the bin has fewer than 5 scored trials.

## Excluded

None.

## support

| bin | n | gold-yes fraction |
| --- | --- | --- |
| 0.0-0.1 | 0 |  |
| 0.1-0.2 | 3 |  |
| 0.2-0.3 | 2 |  |
| 0.3-0.4 | 0 |  |
| 0.4-0.5 | 1 |  |
| 0.5-0.6 | 1 |  |
| 0.6-0.7 | 0 |  |
| 0.7-0.8 | 1 |  |
| 0.8-0.9 | 2 |  |
| 0.9-1.0 | 8 | 0.88 |

No overconfidence or underconfidence is reported for this predicate.

## established

| bin | n | gold-yes fraction |
| --- | --- | --- |
| 0.0-0.1 | 9 | 0.00 |
| 0.1-0.2 | 1 |  |
| 0.2-0.3 | 0 |  |
| 0.3-0.4 | 1 |  |
| 0.4-0.5 | 1 |  |
| 0.5-0.6 | 0 |  |
| 0.6-0.7 | 0 |  |
| 0.7-0.8 | 1 |  |
| 0.8-0.9 | 2 |  |
| 0.9-1.0 | 3 |  |

No overconfidence or underconfidence is reported for this predicate.

## refuted

| bin | n | gold-yes fraction |
| --- | --- | --- |
| 0.0-0.1 | 10 | 0.00 |
| 0.1-0.2 | 2 |  |
| 0.2-0.3 | 1 |  |
| 0.3-0.4 | 1 |  |
| 0.4-0.5 | 1 |  |
| 0.5-0.6 | 0 |  |
| 0.6-0.7 | 0 |  |
| 0.7-0.8 | 1 |  |
| 0.8-0.9 | 2 |  |
| 0.9-1.0 | 0 |  |

No overconfidence or underconfidence is reported for this predicate.

## conflict

| bin | n | gold-yes fraction |
| --- | --- | --- |
| 0.0-0.1 | 7 | 0.00 |
| 0.1-0.2 | 5 | 0.00 |
| 0.2-0.3 | 2 |  |
| 0.3-0.4 | 0 |  |
| 0.4-0.5 | 0 |  |
| 0.5-0.6 | 0 |  |
| 0.6-0.7 | 0 |  |
| 0.7-0.8 | 0 |  |
| 0.8-0.9 | 1 |  |
| 0.9-1.0 | 3 |  |

- 0.1-0.2 overconfident by 0.15 (n=5)

## safe_to_auto_act

| bin | n | gold-yes fraction |
| --- | --- | --- |
| 0.0-0.1 | 8 | 0.13 |
| 0.1-0.2 | 2 |  |
| 0.2-0.3 | 2 |  |
| 0.3-0.4 | 0 |  |
| 0.4-0.5 | 1 |  |
| 0.5-0.6 | 1 |  |
| 0.6-0.7 | 2 |  |
| 0.7-0.8 | 1 |  |
| 0.8-0.9 | 1 |  |
| 0.9-1.0 | 0 |  |

No overconfidence or underconfidence is reported for this predicate.

## Action

Rows are gold action, columns are the returned choice. Choice confidence is not an accuracy rate.

| gold \ live | proceed | reject | manual_review | other |
| --- | --- | --- | --- | --- |
| proceed | 3 | 0 | 0 | 0 |
| reject | 0 | 3 | 0 | 0 |
| manual_review | 1 | 0 | 11 | 0 |
| other | 0 | 0 | 0 | 0 |

## Verdict

Rows are the gold-predicate verdict. Columns are the service verdict. This checks the pinned thresholds.

| gold \ live | validated | rejected | insufficient | conflict | manual_review | error |
| --- | --- | --- | --- | --- | --- | --- |
| validated | 1 | 0 | 0 | 0 | 2 | 0 |
| rejected | 0 | 2 | 1 | 0 | 0 | 0 |
| insufficient | 0 | 0 | 5 | 0 | 0 | 0 |
| conflict | 0 | 0 | 0 | 4 | 0 | 0 |
| manual_review | 0 | 0 | 1 | 0 | 2 | 0 |
| error | 0 | 0 | 0 | 0 | 0 | 0 |

## Scored rows

| id | gold verdict | live verdict | gold action | live action | support | established | refuted | conflict | safe |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| library-book-returned | validated | validated | proceed | proceed | 0.98 | 0.95 | 0.02 | 0.03 | 0.80 |
| yard-gate-closed | validated | manual_review | proceed | proceed | 0.98 | 0.91 | 0.02 | 0.03 | 0.65 |
| lab-freezer-logged | validated | manual_review | proceed | proceed | 0.99 | 0.92 | 0.02 | 0.03 | 0.73 |
| pharmacy-bag-dispensed | manual_review | manual_review | manual_review | manual_review | 0.98 | 0.85 | 0.04 | 0.07 | 0.50 |
| crane-lift-approved | manual_review | manual_review | manual_review | manual_review | 0.97 | 0.85 | 0.03 | 0.08 | 0.40 |
| bakery-batch-baked | manual_review | insufficient | manual_review | proceed | 0.98 | 0.76 | 0.03 | 0.03 | 0.65 |
| parcel-was-delivered | rejected | rejected | reject | reject | 0.22 | 0.02 | 0.84 | 0.09 | 0.10 |
| invoice-was-paid | rejected | insufficient | reject | reject | 0.27 | 0.03 | 0.73 | 0.18 | 0.06 |
| exam-was-rescheduled | rejected | rejected | reject | reject | 0.17 | 0.03 | 0.82 | 0.10 | 0.22 |
| repair-is-covered | conflict | conflict | manual_review | manual_review | 0.87 | 0.08 | 0.42 | 0.94 | 0.05 |
| flight-was-cancelled | conflict | conflict | manual_review | manual_review | 0.88 | 0.09 | 0.36 | 0.95 | 0.05 |
| badge-was-revoked | conflict | conflict | manual_review | manual_review | 0.92 | 0.13 | 0.27 | 0.93 | 0.06 |
| valve-was-replaced | insufficient | insufficient | manual_review | manual_review | 0.43 | 0.07 | 0.04 | 0.17 | 0.06 |
| roster-was-posted | insufficient | insufficient | manual_review | manual_review | 0.14 | 0.05 | 0.04 | 0.13 | 0.05 |
| meter-was-read | insufficient | insufficient | manual_review | manual_review | 0.75 | 0.35 | 0.15 | 0.27 | 0.21 |
| backup-finished | insufficient | insufficient | manual_review | manual_review | 0.96 | 0.43 | 0.04 | 0.13 | 0.19 |
| fee-was-waived | insufficient | insufficient | manual_review | manual_review | 0.14 | 0.04 | 0.08 | 0.26 | 0.04 |
| signature-was-collected | conflict | conflict | manual_review | manual_review | 0.50 | 0.06 | 0.10 | 0.83 | 0.05 |

