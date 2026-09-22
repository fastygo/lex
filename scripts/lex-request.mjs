import { readFileSync } from "node:fs";

// Usage: node scripts/lex-request.mjs [route] [method] [body-file]
// The token is read from .env and never printed. LEX_BASE_URL overrides the host.
const route = process.argv[2] ?? "v1/capabilities";
const method = process.argv[3] ?? "GET";
const bodyFile = process.argv[4];
const base = process.env.LEX_BASE_URL ?? "https://lexproto.vercel.app";
const path = route.startsWith("http://") || route.startsWith("https://")
  ? route
  : `/${route.replace(/^\/+/, "")}`;
const line = readFileSync(new URL("../.env", import.meta.url), "utf8")
  .split(/\r?\n/)
  .find((entry) => entry.startsWith("LEX_BEARER_TOKENS="));
if (!line) {
  console.error("LEX_BEARER_TOKENS is missing from .env");
  process.exit(1);
}

const tokens = JSON.parse(line.slice("LEX_BEARER_TOKENS=".length));
const token = Object.keys(tokens)[0];
if (!token) {
  console.error("LEX_BEARER_TOKENS does not contain a token");
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
