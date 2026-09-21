import { mkdirSync, readdirSync, readFileSync, writeFileSync } from "node:fs";

const origin = "https://lexproto.vercel.app";
const line = readFileSync(new URL("../.env", import.meta.url), "utf8")
  .split(/\r?\n/)
  .find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
const token = Object.keys(JSON.parse(line.slice("LEX_BEARER_TOKENS=".length)))[0];
const requests = new URL("../.project/.jev/test-vercel/requests/", import.meta.url);
const evaluations = new URL("../.project/.jev/test-vercel/evaluations/", import.meta.url);
const examples = new URL("../.project/.jev/examples/", import.meta.url);

async function call(path, method, body) {
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

function noul(answers, id) {
  const value = answers?.[id]?.noul;
  return typeof value === "number" ? value : null;
}

function compact(name, request, evaluation, replay) {
  const bundle = evaluation.parsed?.replay_bundle ?? null;
  const decision = bundle?.decision_set ?? {};
  const answers = decision.answers ?? null;
  const items = bundle?.context?.pack?.evidence_items ?? [];
  const sources = Array.isArray(items)
    ? items.map((item) => item?.source_ref?.source_id).filter((id) => typeof id === "string")
    : [];
  return {
    protocol_version: evaluation.parsed?.protocol_version ?? null,
    verdict: evaluation.parsed?.verdict ?? null,
    findings: (evaluation.parsed?.findings ?? []).map((finding) => ({
      code: finding.code,
      verdict: finding.verdict,
    })),
    context_runtime: evaluation.parsed?.context_runtime ?? null,
    resolved_model: evaluation.parsed?.resolved_model ?? decision.resolved_model ?? null,
    replay_available: evaluation.parsed?.replay_available ?? false,
    policy: evaluation.parsed?.policy ?? null,
    adapter_id: decision.adapter_id ?? null,
    adapter_version: decision.adapter_version ?? null,
    question_set: bundle?.question_set
      ? { id: bundle.question_set.id, version: bundle.question_set.version }
      : null,
    answers,
    query: request.query,
    evidence_count: sources.length,
    evidence_sources: sources,
    http_status: evaluation.status,
    replay_http_status: replay?.status ?? null,
    replay_status: replay?.parsed?.replay_status ?? null,
    replay_verdict: replay?.parsed?.verdict ?? null,
    id: name.replace(/\.json$/, ""),
  };
}

const names = readdirSync(requests).filter((name) => name.endsWith(".json")).sort();
const rows = [];
mkdirSync(evaluations, { recursive: true });
for (const name of names) {
  const request = JSON.parse(readFileSync(new URL(name, requests), "utf8"));
  const evaluation = await call("/v1/evaluations", "POST", JSON.stringify(request));
  let replay = null;
  if (evaluation.parsed?.replay_bundle) {
    replay = await call("/v1/replays", "POST", JSON.stringify(evaluation.parsed.replay_bundle));
  }
  const view = compact(name, request, evaluation, replay);
  writeFileSync(new URL(name, evaluations), `${JSON.stringify(view, null, 2)}\n`);
  rows.push(view);
  const answers = view.answers ?? {};
  console.log([
    view.http_status,
    view.verdict,
    view.replay_http_status,
    view.replay_verdict,
    noul(answers, "support"),
    noul(answers, "established"),
    noul(answers, "refuted"),
    noul(answers, "conflict"),
    noul(answers, "safe_to_auto_act"),
    answers.action?.choice ?? "",
    view.id,
  ].join("\t"));
}

const raw = await call(
  "/v1/evaluations",
  "POST",
  readFileSync(new URL("playground/questions-jev.json", examples), "utf8"),
);
console.log(`raw-questions\t${raw.status}\t${raw.parsed?.reason ?? ""}`);

const output = new URL("../.project/.jev/test-vercel/results.json", import.meta.url);
writeFileSync(output, `${JSON.stringify({
  origin,
  ran_at: new Date().toISOString(),
  protocol: "LeX EvaluationRequest -> POST /v1/evaluations",
  policy: rows[0]?.policy ?? null,
  question_set: rows[0]?.question_set ?? null,
  resolved_model: rows[0]?.resolved_model ?? null,
  adapter_id: rows[0]?.adapter_id ?? null,
  raw_questions_status: raw.status,
  raw_questions_reason: raw.parsed?.reason ?? "",
  results: rows.map((row) => ({
    id: row.id,
    query: row.query,
    http_status: row.http_status,
    verdict: row.verdict,
    replay_http_status: row.replay_http_status,
    replay_verdict: row.replay_verdict,
    replay_status: row.replay_status,
    support: noul(row.answers, "support"),
    established: noul(row.answers, "established"),
    refuted: noul(row.answers, "refuted"),
    conflict: noul(row.answers, "conflict"),
    safe_to_auto_act: noul(row.answers, "safe_to_auto_act"),
    action: row.answers?.action?.choice ?? null,
    findings: row.findings.map((finding) => finding.code),
  })),
}, null, 2)}\n`);
