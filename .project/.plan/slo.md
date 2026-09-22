# SLOs and resource budgets

Status: canary targets, not measured performance or an SLA. Numeric choices
below are LeX engineering targets, not vendor guarantees. Measured baselines
over the declared window are a stable-release gate. Method: [Google SRE SLO guidance](https://sre.google/workbook/implementing-slos/).

## Measurement contract

Use a rolling 28-day window, aggregated outside the function using platform
telemetry and an external probe. No database is needed by the LeX service.
If telemetry is unavailable, report unknown, never 100% compliance.
Use unsampled counts for denominators; sampled traces are diagnostic only.

Eligible requests are authenticated, schema-valid requests within published
size/rate limits, including cold starts. Exclude malformed/unauthorized input
and client-abandoned requests, but report their counts separately. Count
capacity shedding within the advertised load envelope, platform errors,
provider failures, missing results, and timeouts as bad events. Do not hide
upstream failures by excluding them.

A good availability event delivers a complete valid result before the deadline.
Valid negative verdicts, insufficient evidence, conflict, and manual review
count as successful evaluation. A technical `error` verdict counts as bad even
if its envelope is schema-valid. Health checks are not substitutes for this SLI.

## Targets

- SLO-01 replay availability: at least 99.9% good replay requests.
  Error budget = 0.001 times eligible replay requests.
- SLO-02 live evaluation availability: at least 99.5% good evaluation requests,
  including provider availability. Error budget = 0.005 times eligible requests.
- SLO-03 warm replay latency: at least 95% finish within 250 ms and 99% within
  1 s, measured from handler ingress to complete response for the reference load.
- SLO-04 live evaluation latency: at least 95% finish within 10 s and 99% within
  20 s, measured by an external same-region probe, including cold start/network.
  Errors and timeouts are latency misses rather than disappearing from histograms.
- SLO-05 cold replay latency: at least 95% finish within 3 s under the same probe.
  Label cold samples only with platform evidence or deliberately fresh deployments.

Segment by route, deployment, region, cold/warm, adapter path, and input-size band.
Keep labels bounded; no entity ids or evidence text in metrics.

## Reference workload and hard limits

Benchmark the same frozen corpus: up to 256 KiB request JSON, 32 evidence items,
20 atomic questions, no more than 4 simultaneous requests per process. Run
10,000 warm replay requests, 1,000 provider-stub evaluations, and 100 cold
invocations. Record actual CPU/memory allocation, Go/module versions, payload
distribution, dependency latency, and P50/P95/P99. A separate authorized live
provider sample measures external behavior; stub results cannot establish SLO-02.

Use [Context v0.1.0](context-version.md) with an initial 256 KiB configured
snapshot/request ceiling for the reference profile. Independently cap the
complete LeX envelope. Upstream's 128-source/2 MiB hard ceilings do not bound
response size or total process memory; its snapshot and rejected material also
consume the LeX bundle budget.

Advertised maximums:

- Request and response bodies: 2 MiB each, measured as uncompressed UTF-8 bytes.
  Replay's enclosing request must fit too; reserve envelope space when exporting.
- At most 128 evidence items, 64 questions, and JSON nesting depth 32.
- Bound individual strings, total schema complexity, and provider response bytes
  in the final wire profile; no unbounded reads or dynamic remote schema resolution.
- Process concurrency admission: 4 active requests initially; reject overflow
  without an unbounded queue. This is not a global quota.
- Peak process RSS: no more than 70% of configured memory at supported concurrency;
  zero OOMs in the maximum-input stress suite.
- Request deadline: 20 s, provider time budget at most 12 s, no automatic
  decision-provider retry in the initial deployment profile.
- Configure platform duration at least 5 s above the application deadline.
  If the actual Go deployment cannot support this, revise advertised budgets
  before release; do not rely on platform termination to send protocol errors.

The [Vercel limits page](https://vercel.com/docs/functions/limitations) currently
lists a 4.5 MB function request/response cap. The smaller LeX cap leaves margin.
Record actual Go plan/region limits in the deployment proof; generic Fluid
duration or memory figures must not be assumed to apply unchanged.

## Correctness gates, not statistical SLOs

Require 100% golden replay verdict/hash agreement, zero accepted invalid
authorization cases, zero cross-project leaks, and zero provider/secret leakage
in the security corpus. These are release gates, not promises of perfect
production accuracy. Domain accuracy and calibration have separate corpus
reports; high availability does not imply a correct semantic judgment.

## Error-budget response

Budget consumption is bad events divided by the applicable error budget.
Above 50% consumed within seven days, prioritize diagnosis and capacity/provider
mitigation. At 100%, pause feature releases until a corrective release and
passing verification; security fixes remain eligible. Alert on sustained
multi-window burn, not every request. Record owner, cause, stage, and recovery
evidence. A short deployment test does not prove 28-day SLO achievement.
