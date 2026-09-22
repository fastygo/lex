import { existsSync, readFileSync, writeFileSync } from "node:fs";

const origin = "https://lexproto.vercel.app";
const corpusPath = new URL(
  "../.project/.jev/test-vercel/calibration/20260922-0947/corpus.json",
  import.meta.url,
);
const outDir = new URL("../.project/.jev/test-vercel/calibration/20260922-0947/", import.meta.url);
const resultsPath = new URL("results.json", outDir);
const reportPath = new URL("report.md", outDir);
const deriveOnly = process.argv.includes("--derive");
const minBinCount = 5;
const predicates = ["support", "established", "refuted", "conflict", "safe_to_auto_act"];
const thresholds = {
  supportMin: 0.7,
  establishMin: 0.8,
  refuteMin: 0.8,
  conflictMin: 0.5,
  safetyMin: 0.8,
};

function loadToken() {
  const line = readFileSync(new URL("../.env", import.meta.url), "utf8")
    .split(/\r?\n/)
    .find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
  if (!line) throw new Error("LEX_BEARER_TOKENS is missing");
  const token = Object.keys(JSON.parse(line.slice("LEX_BEARER_TOKENS=".length)))[0];
  if (!token) throw new Error("LEX_BEARER_TOKENS has no token");
  return token;
}

function side(label, yesValue, noValue) {
  if (label === "yes") return yesValue;
  if (label === "no") return noValue;
  throw new Error(`label must be yes or no, got ${label}`);
}

function goldVerdict(labels) {
  const support = side(labels.support, 0.9, 0.1);
  const established = side(labels.established, 0.9, 0.1);
  const refuted = side(labels.refuted, 0.9, 0.1);
  const conflict = side(labels.conflict, 0.9, 0.1);
  const safety = side(labels.safe_to_auto_act, 0.9, 0.1);
  const action = labels.action;
  const findings = [];
  const negative = refuted >= thresholds.refuteMin;
  const contradictory =
    negative && (established >= thresholds.establishMin || support >= thresholds.supportMin);
  if (conflict >= thresholds.conflictMin || contradictory) findings.push("conflict");
  if (support < thresholds.supportMin && !negative) findings.push("insufficient");
  if (established < thresholds.establishMin && !negative) findings.push("insufficient");
  if (negative) findings.push("rejected");
  if (action === "manual_review" || action === "other") findings.push("manual_review");
  if (action === "proceed") {
    if (safety < thresholds.safetyMin) findings.push("manual_review");
    if (
      support < thresholds.supportMin ||
      established < thresholds.establishMin ||
      conflict >= thresholds.conflictMin ||
      negative
    ) {
      findings.push("manual_review");
    }
  }
  if (action === "reject" && (!negative || established >= thresholds.establishMin)) {
    findings.push("manual_review");
  }
  const rank = { error: 6, conflict: 5, insufficient: 4, manual_review: 3, rejected: 2, validated: 1 };
  let verdict = "validated";
  for (const finding of findings) {
    if (rank[finding] > rank[verdict]) verdict = finding;
  }
  return verdict;
}

function noul(answers, id) {
  const value = answers?.[id]?.noul;
  return typeof value === "number" ? value : null;
}

function validateCorpus(corpus) {
  const ids = new Set();
  for (const item of corpus.items) {
    if (ids.has(item.id)) throw new Error(`duplicate id ${item.id}`);
    ids.add(item.id);
    for (const predicate of predicates) {
      if (item.labels[predicate] !== "yes" && item.labels[predicate] !== "no") {
        throw new Error(`${item.id} ${predicate} is not yes or no`);
      }
    }
    if (!["proceed", "reject", "manual_review", "other"].includes(item.labels.action)) {
      throw new Error(`${item.id} action is not a gold action`);
    }
    const texts = item.request.sources.map((source) => source.text);
    if (!texts.some((text) => text.includes(item.request.query.trim()))) {
      throw new Error(`${item.id} query is not in source text`);
    }
  }
}

function binIndex(probability) {
  if (probability < 0 || probability > 1) return null;
  if (probability === 1) return 9;
  return Math.floor(probability * 10);
}

function reliability(rows) {
  const bins = Array.from({ length: 10 }, (_, index) => ({
    lo: index / 10,
    hi: (index + 1) / 10,
    n: 0,
    yes: 0,
    fraction: null,
  }));
  for (const row of rows) {
    const index = binIndex(row.probability);
    if (index === null) continue;
    bins[index].n += 1;
    bins[index].yes += row.yes;
  }
  for (const bin of bins) {
    if (bin.n >= minBinCount) bin.fraction = bin.yes / bin.n;
  }
  return bins;
}

function confusion(rows, keyGold, keyLive, values) {
  const table = {};
  for (const gold of values) {
    table[gold] = {};
    for (const live of values) table[gold][live] = 0;
  }
  for (const row of rows) {
    table[row[keyGold]][row[keyLive]] += 1;
  }
  return table;
}

function markdownTable(headers, rows) {
  const head = `| ${headers.join(" | ")} |`;
  const rule = `| ${headers.map(() => "---").join(" | ")} |`;
  const body = rows.map((row) => `| ${row.join(" | ")} |`).join("\n");
  return `${head}\n${rule}\n${body}`;
}

function formatFraction(bin) {
  if (bin.fraction === null) return "";
  return bin.fraction.toFixed(2);
}

function bias(bins) {
  const notes = [];
  for (const bin of bins) {
    if (bin.fraction === null) continue;
    const center = (bin.lo + bin.hi) / 2;
    const gap = bin.fraction - center;
    if (gap >= 0.1) notes.push(`${bin.lo.toFixed(1)}-${bin.hi.toFixed(1)} underconfident by ${gap.toFixed(2)} (n=${bin.n})`);
    if (gap <= -0.1) notes.push(`${bin.lo.toFixed(1)}-${bin.hi.toFixed(1)} overconfident by ${(-gap).toFixed(2)} (n=${bin.n})`);
  }
  return notes;
}

async function call(token, path, method, body) {
  const headers = { Authorization: `Bearer ${token}`, Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const response = await fetch(new URL(path, origin), { method, headers, body });
  const text = await response.text();
  let parsed = null;
  try {
    parsed = JSON.parse(text);
  } catch {
    parsed = null;
  }
  return { status: response.status, parsed };
}

const corpus = JSON.parse(readFileSync(corpusPath, "utf8"));
validateCorpus(corpus);
const derived = corpus.items.map((item) => ({ id: item.id, gold_verdict: goldVerdict(item.labels) }));
if (deriveOnly) {
  for (const row of derived) console.log(`${row.id} ${row.gold_verdict}`);
  process.exit(0);
}
if (existsSync(resultsPath)) {
  throw new Error("results.json already exists; a repeat is not a new labeled trial");
}

const token = loadToken();
const capabilities = await call(token, "/v1/capabilities", "GET");
const policy = capabilities.parsed?.policy ?? {};
const context = capabilities.parsed?.context ?? {};
if (capabilities.status !== 200) throw new Error(`capabilities HTTP ${capabilities.status}`);
if (policy.id !== "claim-validation" || policy.version !== "0.2.0" || policy.calibration !== "uncalibrated") {
  throw new Error("capabilities pins do not match claim-validation 0.2.0 uncalibrated");
}
if (context.capability !== "memory-exact-v1" || context.retrieval !== "exact_phrase") {
  throw new Error("capabilities retrieval pin does not match memory-exact-v1 exact_phrase");
}

const trials = [];
const excluded = [];
for (const item of corpus.items) {
  const evaluation = await call(token, "/v1/evaluations", "POST", JSON.stringify(item.request));
  const bundle = evaluation.parsed?.replay_bundle ?? null;
  const decision = bundle?.decision_set ?? {};
  const answers = decision.answers ?? null;
  let replay = null;
  if (bundle) replay = await call(token, "/v1/replays", "POST", JSON.stringify(bundle));
  const row = {
    id: item.id,
    gold: {
      support: item.labels.support,
      established: item.labels.established,
      refuted: item.labels.refuted,
      conflict: item.labels.conflict,
      safe_to_auto_act: item.labels.safe_to_auto_act,
      action: item.labels.action,
      verdict: goldVerdict(item.labels),
    },
    http_status: evaluation.status,
    verdict: evaluation.parsed?.verdict ?? null,
    findings: (evaluation.parsed?.findings ?? []).map((finding) => finding.code),
    policy: evaluation.parsed?.policy ?? null,
    question_set: bundle?.question_set
      ? { id: bundle.question_set.id, version: bundle.question_set.version }
      : null,
    adapter_id: decision.adapter_id ?? null,
    adapter_version: decision.adapter_version ?? null,
    resolved_model: evaluation.parsed?.resolved_model ?? decision.resolved_model ?? null,
    context_runtime: evaluation.parsed?.context_runtime ?? null,
    nouls: Object.fromEntries(predicates.map((predicate) => [predicate, noul(answers, predicate)])),
    action: answers?.action?.choice ?? null,
    action_confidence: answers?.action?.confidence ?? null,
    replay_http_status: replay?.status ?? null,
    replay_status: replay?.parsed?.replay_status ?? null,
    replay_verdict: replay?.parsed?.verdict ?? null,
  };
  const pinned =
    row.http_status === 200 &&
    row.policy?.version === "0.2.0" &&
    row.policy?.calibration === "uncalibrated" &&
    row.question_set?.version === "0.2.0" &&
    row.resolved_model === "jev-1.13.0" &&
    row.replay_status === "verdict_reproduced" &&
    row.replay_verdict === row.verdict &&
    predicates.every((predicate) => typeof row.nouls[predicate] === "number");
  if (!pinned) excluded.push(row);
  else trials.push(row);
  process.stdout.write(`${item.id} ${row.http_status} ${row.verdict ?? "none"} replay ${row.replay_status ?? "none"}\n`);
}

const tables = {};
for (const predicate of predicates) {
  tables[predicate] = reliability(
    trials.map((trial) => ({
      probability: trial.nouls[predicate],
      yes: trial.gold[predicate] === "yes" ? 1 : 0,
    })),
  );
}
const actionValues = ["proceed", "reject", "manual_review", "other"];
const verdictValues = ["validated", "rejected", "insufficient", "conflict", "manual_review", "error"];
const actionRows = trials.map((trial) => ({ goldAction: trial.gold.action, liveAction: trial.action }));
const verdictRows = trials.map((trial) => ({ goldVerdict: trial.gold.verdict, liveVerdict: trial.verdict }));
const scoredAction = confusion(actionRows, "goldAction", "liveAction", actionValues);
const scoredVerdict = confusion(verdictRows, "goldVerdict", "liveVerdict", verdictValues);

const results = {
  origin,
  labeled_at: corpus.labeled_at,
  ran_at: new Date().toISOString(),
  policy,
  context,
  min_bin_count: minBinCount,
  trial_count: trials.length,
  excluded_count: excluded.length,
  trials,
  excluded,
  reliability: tables,
  action_confusion: scoredAction,
  verdict_confusion: scoredVerdict,
};
writeFileSync(resultsPath, `${JSON.stringify(results, null, 2)}\n`);

const lines = [];
lines.push("# Calibration trial 20260922-0947");
lines.push("");
lines.push("Status: observation. Policy `claim-validation` `0.2.0` stays `uncalibrated`. This file does not select new thresholds.");
lines.push("");
lines.push(`- Labeled at: \`${corpus.labeled_at}\` (before any answer in this trial)`);
lines.push(`- Ran at: \`${results.ran_at}\``);
lines.push(`- Origin: \`${origin}\``);
lines.push(`- Policy: \`${policy.id}\` \`${policy.version}\` \`${policy.calibration}\``);
lines.push(`- Retrieval: \`${context.retrieval}\` / \`${context.capability}\``);
lines.push(`- Resolved model on scored rows: \`jev-1.13.0\``);
lines.push(`- Scored trials: ${trials.length}`);
lines.push(`- Excluded trials: ${excluded.length}`);
lines.push(`- A reliability fraction is blank when the bin has fewer than ${minBinCount} scored trials.`);
lines.push("");
lines.push("## Excluded");
lines.push("");
if (excluded.length === 0) lines.push("None.");
for (const row of excluded) {
  lines.push(`- \`${row.id}\` HTTP ${row.http_status}, verdict ${row.verdict}, replay ${row.replay_status}, model ${row.resolved_model}`);
}
lines.push("");
for (const predicate of predicates) {
  lines.push(`## ${predicate}`);
  lines.push("");
  lines.push(
    markdownTable(
      ["bin", "n", "gold-yes fraction"],
      tables[predicate].map((bin) => [`${bin.lo.toFixed(1)}-${bin.hi.toFixed(1)}`, String(bin.n), formatFraction(bin)]),
    ),
  );
  lines.push("");
  const notes = bias(tables[predicate]);
  lines.push(notes.length === 0 ? "No overconfidence or underconfidence is reported for this predicate." : notes.map((note) => `- ${note}`).join("\n"));
  lines.push("");
}
lines.push("## Action");
lines.push("");
lines.push("Rows are gold action, columns are the returned choice. Choice confidence is not an accuracy rate.");
lines.push("");
lines.push(
  markdownTable(
    ["gold \\ live", ...actionValues],
    actionValues.map((gold) => [gold, ...actionValues.map((live) => String(scoredAction[gold][live]))]),
  ),
);
lines.push("");
lines.push("## Verdict");
lines.push("");
lines.push("Rows are the gold-predicate verdict. Columns are the service verdict. This checks the pinned thresholds.");
lines.push("");
lines.push(
  markdownTable(
    ["gold \\ live", ...verdictValues],
    verdictValues.map((gold) => [gold, ...verdictValues.map((live) => String(scoredVerdict[gold][live]))]),
  ),
);
lines.push("");
lines.push("## Scored rows");
lines.push("");
lines.push(
  markdownTable(
    ["id", "gold verdict", "live verdict", "gold action", "live action", "support", "established", "refuted", "conflict", "safe"],
    trials.map((trial) => [
      trial.id,
      trial.gold.verdict,
      trial.verdict,
      trial.gold.action,
      trial.action,
      trial.nouls.support.toFixed(2),
      trial.nouls.established.toFixed(2),
      trial.nouls.refuted.toFixed(2),
      trial.nouls.conflict.toFixed(2),
      trial.nouls.safe_to_auto_act.toFixed(2),
    ]),
  ),
);
lines.push("");
writeFileSync(reportPath, `${lines.join("\n")}\n`);
console.log(`scored ${trials.length} excluded ${excluded.length}`);
