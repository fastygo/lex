import { readdirSync, readFileSync, writeFileSync } from "node:fs";

const origin = "https://lexproto.vercel.app";
const line = readFileSync(new URL("../.env", import.meta.url), "utf8")
  .split(/\r?\n/)
  .find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
const token = Object.keys(JSON.parse(line.slice("LEX_BEARER_TOKENS=".length)))[0];
const requests = new URL("../.project/.jev/test-vercel/requests/", import.meta.url);

async function call(scenario) {
  const headers = { ...(scenario.headers ?? {}) };
  if (scenario.auth) headers.Authorization = `Bearer ${token}`;
  const response = await fetch(new URL(scenario.path, origin), {
    method: scenario.method,
    headers,
    body: scenario.body,
  });
  const text = await response.text();
  let reason = "";
  let verdict = "";
  let replayStatus = "";
  let bundle;
  try {
    const parsed = JSON.parse(text);
    reason = parsed.reason ?? "";
    verdict = parsed.verdict ?? "";
    replayStatus = parsed.replay_status ?? "";
    bundle = parsed.replay_bundle;
  } catch {
    reason = "";
  }
  return {
    id: scenario.id,
    method: scenario.method,
    path: scenario.path,
    status: response.status,
    reason,
    verdict,
    replay_status: replayStatus,
    cache_control: response.headers.get("cache-control") ?? "",
    bundle,
  };
}

function published(result) {
  const view = { ...result };
  delete view.bundle;
  return view;
}

const probes = [
  { id: "health-open", method: "GET", path: "/healthz" },
  { id: "capabilities-open", method: "GET", path: "/v1/capabilities" },
  { id: "capabilities-auth", method: "GET", path: "/v1/capabilities", auth: true },
  { id: "replay-get", method: "GET", path: "/v1/replays", auth: true },
  {
    id: "replay-empty-object",
    method: "POST",
    path: "/v1/replays",
    auth: true,
    headers: { "Content-Type": "application/json" },
    body: "{}",
  },
  {
    id: "replay-plain-text",
    method: "POST",
    path: "/v1/replays",
    auth: true,
    headers: { "Content-Type": "text/plain" },
    body: "{}",
  },
];

const results = [];
for (const scenario of probes) results.push(published(await call(scenario)));

const names = readdirSync(requests).filter((name) => name.endsWith(".json")).sort();
for (const name of names) {
  const evaluation = await call({
    id: name,
    method: "POST",
    path: "/v1/evaluations",
    auth: true,
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: readFileSync(new URL(name, requests)),
  });
  results.push(published(evaluation));
  if (evaluation.bundle === undefined) continue;
  const replay = await call({
    id: `${name}:replay`,
    method: "POST",
    path: "/v1/replays",
    auth: true,
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(evaluation.bundle),
  });
  results.push(published(replay));
}

const output = new URL("../.project/.jev/test-vercel/results.json", import.meta.url);
writeFileSync(output, `${JSON.stringify({ origin, ran_at: new Date().toISOString(), results }, null, 2)}\n`);
for (const result of results) {
  console.log(`${result.status} ${result.method} ${result.path} ${result.id} ${result.reason} ${result.verdict} ${result.replay_status}`);
}
