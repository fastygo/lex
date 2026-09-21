import { readFileSync, writeFileSync } from "node:fs";

const origin = "https://lexproto.vercel.app";
const line = readFileSync(new URL("../.env", import.meta.url), "utf8")
  .split(/\r?\n/)
  .find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
const token = Object.keys(JSON.parse(line.slice("LEX_BEARER_TOKENS=".length)))[0];
const examples = new URL("../.project/.jev/examples/", import.meta.url);

const files = [
  ["playground-questions", "playground/questions-jev.json"],
  ["playground-response", "playground/response-jev.json"],
  ["context-questions", "context/questions-context.json"],
  ["context-response", "context/response-context.json"],
  ["llm-questions", "llm/questions-llm.json"],
  ["llm-response", "llm/response-llm.json"],
  ["chaos-questions", "chaos/questions-chaos.json"],
  ["chaos-response", "chaos/response-chaos.json"],
];

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
  try {
    reason = JSON.parse(text).reason ?? "";
  } catch {
    reason = "";
  }
  return {
    id: scenario.id,
    method: scenario.method,
    path: scenario.path,
    status: response.status,
    reason,
    cache_control: response.headers.get("cache-control") ?? "",
  };
}

const scenarios = [
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

for (const [id, relativePath] of files) {
  const body = readFileSync(new URL(relativePath, examples));
  for (const path of ["/v1/evaluations", "/v1/replays"]) {
    scenarios.push({
      id: `${id}:${path}`,
      method: "POST",
      path,
      auth: true,
      headers: { "Content-Type": "application/json" },
      body,
    });
  }
}

const results = [];
for (const scenario of scenarios) results.push(await call(scenario));
const output = new URL("../.project/.jev/test-vercel/results.json", import.meta.url);
writeFileSync(output, `${JSON.stringify({ origin, ran_at: new Date().toISOString(), results }, null, 2)}\n`);
for (const result of results) {
  console.log(`${result.status} ${result.method} ${result.path} ${result.id} ${result.reason}`);
}
