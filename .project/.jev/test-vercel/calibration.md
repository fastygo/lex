# How to calibrate claim-validation

Status: procedure. The embedded policy `claim-validation` `0.2.0` is
`uncalibrated`. The 18 files in [requests/](requests/) and the table in
[findings.md](findings.md) are one live sample. They are not a calibration
corpus and they do not justify new thresholds.

Calibration measures whether a pinned Noul probability matches the frequency
of a label. It does not grant authority, and a Choice confidence value is not
that measurement.

## What stays fixed

Record these from `GET /v1/capabilities` and from one evaluation response
before scoring any item:

- entity type `claim`, schema `0.1`;
- question set `claim-validation` `0.2.0`;
- policy `claim-validation` `0.2.0`, including `support_min` 0.7,
  `establish_min` 0.8, `refute_min` 0.8, `conflict_min` 0.5, and
  `safety_min` 0.8;
- retrieval `exact_phrase`, runtime `memory-exact-v1`;
- adapter id and adapter version;
- resolved model id, currently `jev-1.13.0` on the direct adapter.

A different model, adapter, entity type, or policy version is a separate
report. Do not pool those rows into one rate.

The current gates are: conflict at or above `conflict_min` is a conflict
finding; support, establishment, and safety findings use a strict less-than
comparison; refutation at or above `refute_min` is `negative_result`.
Verdict precedence remains `error > conflict > insufficient > manual_review >
rejected > validated`.

## Labels

Write labels from the claim and the source text before reading model answers.
Each labeled item is one LeX evaluation body, the same shape as
[requests/](requests/): `project_id`, a `claim` entity, an exact-phrase
`query`, and versioned source text. The caller does not send the question set.
The phrase in `query` must occur in the source text that the label is about.

Use an independent yes/no label for each predicate:

| Answer | Label is yes when |
| --- | --- |
| `support` | Admissible evidence is sufficient to support the claim. |
| `established` | Admissible evidence establishes the claim. |
| `refuted` | Admissible evidence establishes that the claim is false. Low support alone is not refutation. |
| `conflict` | Admissible evidence supports mutually incompatible conclusions. A coherent refutation alone is not conflict. |
| `safe_to_auto_act` | An automatic disposition would be safe for this claim and evidence. |

Also record one gold action: `proceed`, `reject`, `manual_review`, or `other`.
That label is the disposition a reviewer would choose. It is not a rewrite of
the five predicate labels.

Keep the gold verdict, if you record one, as the verdict the same precedence
would produce from the gold predicates. An operational preference that
disagrees with those predicates is a second column, not a replacement.

The research maps under `../examples/` are not evaluation bodies and are not
labels. Posting `questions-*.json` is not a calibration trial.

## Run

Send every labeled body once, and only once, to the pinned deployment:

```text
POST /v1/evaluations
Authorization: Bearer <deployment token>
Content-Type: application/json
Accept: application/json
```

Save a compact record in the same spirit as [evaluations/](evaluations/):
HTTP status, policy id and version, question-set id and version, adapter id,
resolved model, the five Noul values, the action choice, finding codes, and
the verdict. Then `POST /v1/replays` with the returned `replay_bundle` and
keep the trial only when the replay verdict matches. Do not store the bearer
token or the provider credential.

Do not change a threshold, question, or model while the corpus is running.
A failed HTTP trial is a transport or contract failure. Leave it out of the
probability tables and count it separately. Do not impute a Noul value.

Repeating the same body measures stability. It does not add a new labeled
outcome. Report repeats apart from the labeled corpus.

## Score

For each Noul, sort the saved probabilities into fixed bins that cover 0
through 1, such as ten bins of width 0.1. In each bin, report the number of
trials and the fraction of gold-yes labels. A bin with too few trials stays
blank. Publish the count beside every fraction.

Read the result as a reliability table: a bin centered near 0.8 should
contain gold-yes labels about 80 percent of the time if that Noul is
calibrated. Report overconfidence and underconfidence per predicate. Do not
average the five Nouls into one score.

For the action, report a confusion table of gold action against the returned
choice. The confidence field on that choice is not an accuracy rate.

For the verdict, report a confusion table of the gold-predicate verdict
against the service verdict. That table checks the pinned thresholds. It is
not a substitute for the Noul reliability tables.

The 18-request sample cannot fill these bins. Do not turn
[findings.md](findings.md) into calibration rates.

## Publishing a new policy

A measured table may justify different thresholds. That change is a new
policy version and a new policy hash. Leave `0.2.0` and its `uncalibrated`
disclosure in place for replay of bundles that name it.

Ship the report with the new version. The report names the corpus, its
label rules, the pins above, the bin counts, and the threshold values that
were selected from those counts. The service may then disclose that policy
as calibrated for that entity type, resolved model, adapter, and corpus.
One corpus does not calibrate a second domain or a second model.

Until that report exists, capabilities and evaluation responses keep
`calibration: uncalibrated`.
