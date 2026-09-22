import { readFileSync } from "node:fs";

// Usage: node scripts/lex-request.mjs [route] [method] [body-file]
// The token comes from LEX_TOKEN, else the first LEX_BEARER_TOKENS key in .env,
// and is never printed. LEX_BASE_URL overrides the host.
const route = process.argv[2] ?? "v1/capabilities";
const method = process.argv[3] ?? "GET";
const bodyFile = process.argv[4];
const base = process.env.LEX_BASE_URL ?? "https://lexproto.vercel.app";
const path = route.startsWith("http://") || route.startsWith("https://")
  ? route
  : `/${route.replace(/^\/+/, "")}`;

function tokenFromEnvFile() {
  let text;
  try {
    text = readFileSync(new URL("../.env", import.meta.url), "utf8");
  } catch {
    return "";
  }
  const line = text.split(/\r?\n/).find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
  if (!line) return "";
  return Object.keys(JSON.parse(line.slice("LEX_BEARER_TOKENS=".length)))[0] ?? "";
}

const token = process.env.LEX_TOKEN || tokenFromEnvFile();
if (!token) {
  console.error("Set LEX_TOKEN or LEX_BEARER_TOKENS in .env");
  process.exit(1);
}

const headers = { Authorization: `Bearer ${token}`, Accept: "application/json" };
let body;
if (bodyFile) {
  body = readFileSync(bodyFile, "utf8");
  headers["Content-Type"] = "application/json";
}
const response = await fetch(path.startsWith("http") ? path : new URL(path, base), {
  method,
  headers,
  body,
});
console.log(response.status);
console.log(await response.text());
if (!response.ok) process.exit(1);
