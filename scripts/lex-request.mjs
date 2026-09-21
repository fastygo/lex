import { readFileSync } from "node:fs";

const route = process.argv[2] ?? "v1/capabilities";
const method = process.argv[3] ?? "GET";
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

const response = await fetch(path.startsWith("http") ? path : new URL(path, "https://lexproto.vercel.app"), {
  method,
  headers: { Authorization: `Bearer ${token}` },
});
console.log(response.status);
console.log(await response.text());
if (!response.ok) process.exit(1);
