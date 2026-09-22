import { mkdirSync, readdirSync, readFileSync, writeFileSync } from "node:fs";

// Posts every legacy scenario body to /v1/evaluations, replays each sealed
// bundle, and records canary signals. The token is never printed or stored.
const origin = process.env.LEX_ORIGIN ?? "https://lexproto.vercel.app";
const revision = process.env.LEX_REVISION;
const deploymentID = process.env.LEX_DEPLOYMENT_ID;
const rollbackDeploymentID = process.env.LEX_ROLLBACK_DEPLOYMENT_ID;
if (!revision || !deploymentID || !rollbackDeploymentID) {
  throw new Error("LEX_REVISION, LEX_DEPLOYMENT_ID, and LEX_ROLLBACK_DEPLOYMENT_ID are required");
}

function tokenFromEnvFile() {
  const line = readFileSync(new URL("../.env", import.meta.url), "utf8")
    .split(/\r?\n/)
    .find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
  return line ? Object.keys(JSON.parse(line.slice("LEX_BEARER_TOKENS=".length)))[0] ?? "" : "";
}
const token = process.env.LEX_TOKEN || tokenFromEnvFile();
if (!token) {
  throw new Error("Set LEX_TOKEN or LEX_BEARER_TOKENS in .env");
}

async function post(path, text) {
  const started = performance.now();
  const response = await fetch(new URL(path, origin), {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: text,
  });
  const raw = await response.text();
  let parsed = {};
  try {
    parsed = JSON.parse(raw);
  } catch {
    parsed = {};
  }
  return { status: response.status, elapsed_ms: Math.round(performance.now() - started), parsed };
}

const directory = new URL("../.project/.jev/test-vercel/requests/", import.meta.url);
const files = readdirSync(directory).filter((name) => name.endsWith(".json")).sort();
const rows = [];
for (const file of files) {
  const evaluation = await post("/v1/evaluations", readFileSync(new URL(file, directory), "utf8"));
  const bundle = evaluation.parsed.replay_bundle;
  const findings = (evaluation.parsed.findings ?? []).map((finding) => finding.code).sort();
  let replay = { status: 0, elapsed_ms: 0, parsed: {} };
  if (bundle) {
    replay = await post("/v1/replays", JSON.stringify(bundle));
  }
  rows.push({
    request: file,
    http: evaluation.status,
    elapsed_ms: evaluation.elapsed_ms,
    protocol_version: bundle?.protocol_version ?? null,
    reason: evaluation.parsed.reason ?? null,
    findings,
    verdict: evaluation.parsed.verdict ?? null,
    replay_http: replay.status,
    replay_elapsed_ms: replay.elapsed_ms,
    replay_verdict: replay.parsed.verdict ?? null,
    replay_matches: Boolean(bundle) && replay.status === 200 && replay.parsed.verdict === evaluation.parsed.verdict,
  });
}

const count = (code) => rows.filter((row) => row.findings.includes(code)).length;
const signals = {
  origin,
  revision,
  deployment_id: Number(deploymentID),
  requests: rows.length,
  http_5xx: rows.filter((row) => row.http >= 500 || row.replay_http >= 500).length,
  verification_error: rows.filter((row) => row.reason === "verification_error").length,
  pack_rebuild: count("pack_rebuild"),
  snapshot_identity: count("snapshot_identity"),
  wrong_protocol_version: rows.filter((row) => row.protocol_version !== "0.1").length,
  replay_matches: rows.filter((row) => row.replay_matches).length,
  rollback_deployment_id: Number(rollbackDeploymentID),
  alias_switched: false,
};

const runID = new Date().toISOString().replace(/[-:]/g, "").replace(/\.\d{3}Z$/, "Z");
const output = new URL(`../.project/.jev/test-vercel/canary/${runID}/`, import.meta.url);
mkdirSync(output, { recursive: true });
writeFileSync(new URL("summary.json", output), `${JSON.stringify(rows, null, 2)}\n`);
writeFileSync(new URL("signals.json", output), `${JSON.stringify(signals, null, 2)}\n`);
console.log(JSON.stringify(signals));
console.log(`.project/.jev/test-vercel/canary/${runID}`);
