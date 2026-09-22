import { mkdirSync, readFileSync, writeFileSync } from "node:fs";

const origin = process.env.LEX_ORIGIN ?? "https://lexproto.vercel.app";
const revision = process.env.LEX_REVISION;
const deploymentID = process.env.LEX_DEPLOYMENT_ID;

if (!revision || !deploymentID) {
  throw new Error("LEX_REVISION and LEX_DEPLOYMENT_ID are required");
}

const tokenLine = readFileSync(new URL("../.env", import.meta.url), "utf8")
  .split(/\r?\n/)
  .find((line) => line.startsWith("LEX_BEARER_TOKENS="));
if (!tokenLine) {
  throw new Error("LEX_BEARER_TOKENS is missing from .env");
}
const token = Object.keys(JSON.parse(tokenLine.slice("LEX_BEARER_TOKENS=".length)))[0];
if (!token) {
  throw new Error("LEX_BEARER_TOKENS has no bearer token");
}

function readJSON(relative) {
  return JSON.parse(readFileSync(new URL(relative, import.meta.url), "utf8"));
}

const contextFixture = readJSON("../.project/.jev/test-vercel/requests/context-account-access-frozen.json");
const probes = [
  {
    name: "intent-noul-choice",
    request: readJSON("../.project/.jev/generic/intent-decision-request.json"),
  },
  {
    name: "storage-noul-score",
    request: readJSON("../.project/.jev/generic/database-decision-request.json"),
  },
  {
    name: "context-noul-choice",
    request: {
      project_id: contextFixture.project_id,
      decision: { id: "context-binding-step", version: "1" },
      state: { task: "classify the supplied request without merging frozen evidence" },
      question_set: {
        id: "example.context-binding",
        version: "1",
        questions: {
          route: {
            type: "choice",
            instructions: "Which route is best supported by the supplied state?",
            options: {
              account_access: "Account access support",
              billing: "Billing support",
              other: "No declared route",
            },
          },
          has_frozen_context: {
            type: "noul",
            instructions: "Is a frozen context binding supplied with this state?",
          },
        },
      },
      context: contextFixture.context,
      metadata: { fixture: "generic-canary-context" },
    },
  },
];

async function call(path, body) {
  const started = performance.now();
  const response = await fetch(new URL(path, origin), {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let parsed;
  try {
    parsed = JSON.parse(text);
  } catch {
    parsed = {};
  }
  return { status: response.status, elapsed_ms: Math.round(performance.now() - started), parsed };
}

const capability = await fetch(new URL("/v1/capabilities", origin), {
  headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
});
const capabilityJSON = await capability.json();
if (
  capability.status !== 200 ||
  capabilityJSON.operations?.decision !== true ||
  capabilityJSON.typed_decision?.semantic_verdict !== false
) {
  throw new Error("production capabilities do not expose the generic decision contract");
}

const rows = [];
for (const probe of probes) {
  const decision = await call("/v1/decisions", probe.request);
  const bundle = decision.parsed?.replay_bundle;
  if (decision.status !== 200 || decision.parsed?.structural_status !== "valid" || !bundle) {
    throw new Error(`${probe.name} did not return a valid generic DecisionSet`);
  }
  const replay = await call("/v1/replays", bundle);
  if (
    replay.status !== 200 ||
    replay.parsed?.replay_status !== "decision_reproduced" ||
    replay.parsed?.structural_status !== "valid"
  ) {
    throw new Error(`${probe.name} replay did not reproduce structural validity`);
  }
  const bundleText = JSON.stringify(bundle);
  if (bundleText.includes("generic-canary-context") || bundleText.includes('"fixture"')) {
    throw new Error(`${probe.name} retained non-normative metadata in its replay bundle`);
  }
  rows.push({
    name: probe.name,
    primitives: Object.values(probe.request.question_set.questions).map((question) => question.type).sort(),
    context_bound: Boolean(probe.request.context),
    decision_http: decision.status,
    decision_elapsed_ms: decision.elapsed_ms,
    structural_status: decision.parsed.structural_status,
    resolved_model: decision.parsed.decision_set?.resolved_model ?? null,
    adapter_id: decision.parsed.decision_set?.adapter_id ?? null,
    bundle_hash: bundle.bundle_hash ?? null,
    replay_http: replay.status,
    replay_elapsed_ms: replay.elapsed_ms,
    replay_status: replay.parsed.replay_status,
    replay_structural_status: replay.parsed.structural_status,
  });
}

const runID = new Date().toISOString().replace(/[-:]/g, "").replace(/\.\d{3}Z$/, "Z");
const output = new URL(`../.project/.jev/test-vercel/generic-canary/${runID}/`, import.meta.url);
mkdirSync(output, { recursive: true });
writeFileSync(
  new URL("summary.json", output),
  `${JSON.stringify({ origin, revision, deployment_id: Number(deploymentID), probes: rows }, null, 2)}\n`,
);
writeFileSync(
  new URL("capabilities.json", output),
  `${JSON.stringify({
    operations: capabilityJSON.operations,
    typed_decision: capabilityJSON.typed_decision,
  }, null, 2)}\n`,
);

for (const row of rows) {
  console.log(`${row.name}\t${row.decision_http}\t${row.replay_http}\t${row.bundle_hash}`);
}
console.log(`.project/.jev/test-vercel/generic-canary/${runID}`);
