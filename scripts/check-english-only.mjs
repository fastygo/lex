#!/usr/bin/env node
/**
 * Ensures repository text uses English-only content (Latin script + allowed punctuation).
 * Skips the `.manual/` directory tree entirely.
 */

import { readdir, readFile } from "node:fs/promises";
import path from "node:path";
import process from "node:process";

const ROOT = path.resolve(import.meta.dirname, "..");

const SKIP_DIR_NAMES = new Set([
  ".git",
  "node_modules",
  ".manual",
]);

const SKIP_FILE_NAMES = new Set([
  "check-english-only.mjs",
]);

/** Unicode code points allowed outside ASCII for English typography. */
const ALLOWED_EXTRA = new Set([
  0x00a0, // nbsp
  0x00d7, // multiplication sign (e.g. Context × Jev)
  0x2013, // en dash
  0x2014, // em dash
  0x2018, // left single quote
  0x2019, // right single quote / apostrophe
  0x201c, // left double quote
  0x201d, // right double quote
  0x2192, // rightwards arrow
  0x2193, // downwards arrow
]);

/** Major non-Latin scripts and symbols (Cyrillic, CJK, Arabic, emoji, etc.). */
const FORBIDDEN =
  /[\u0370-\u03FF\u0400-\u052F\u0590-\u05FF\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF\u0900-\u097F\u0980-\u09FF\u0A00-\u0A7F\u0A80-\u0AFF\u0B00-\u0B7F\u0B80-\u0BFF\u0C00-\u0C7F\u0C80-\u0CFF\u0D00-\u0D7F\u1100-\u11FF\u3040-\u30FF\u3400-\u4DBF\u4E00-\u9FFF\uAC00-\uD7AF\uF900-\uFAFF\u{1F000}-\u{1FAFF}]/u;

function isAllowedCodePoint(code) {
  if (code === 0x09 || code === 0x0a || code === 0x0d) return true;
  if (code >= 0x20 && code <= 0x7e) return true;
  if (code >= 0xa0 && code <= 0x024f) return true; // Latin-1 + Latin Extended
  if (ALLOWED_EXTRA.has(code)) return true;
  return false;
}

function findViolations(text) {
  const hits = [];
  for (let i = 0; i < text.length; ) {
    const code = text.codePointAt(i);
    const char = String.fromCodePoint(code);
    const line = text.slice(0, i).split("\n").length;
    const col = i - text.lastIndexOf("\n", i - 1);

    if (FORBIDDEN.test(char)) {
      hits.push({ line, col, char, code, reason: "non-English script or emoji" });
    } else if (!isAllowedCodePoint(code)) {
      hits.push({
        line,
        col,
        char,
        code,
        reason: "character outside allowed English/Latin set",
      });
    }

    i += char.length;
  }
  return hits;
}

function shouldSkipDir(name) {
  return SKIP_DIR_NAMES.has(name);
}

async function walk(dir, relBase, files) {
  const entries = await readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const rel = relBase ? `${relBase}/${entry.name}` : entry.name;
    if (entry.isDirectory()) {
      if (shouldSkipDir(entry.name)) continue;
      await walk(path.join(dir, entry.name), rel, files);
      continue;
    }
    if (!entry.isFile()) continue;
    if (SKIP_FILE_NAMES.has(entry.name)) continue;
    files.push({ abs: path.join(dir, entry.name), rel: rel.replace(/\\/g, "/") });
  }
}

async function isBinary(abs) {
  const buf = await readFile(abs);
  const sample = buf.subarray(0, Math.min(buf.length, 8192));
  return sample.includes(0);
}

async function main() {
  const files = [];
  await walk(ROOT, "", files);

  let failed = false;
  const reports = [];

  for (const { abs, rel } of files) {
    if (rel.split("/").some((seg) => seg === ".manual")) continue;

    if (await isBinary(abs)) continue;

    const text = await readFile(abs, "utf8");
    const violations = findViolations(text);
    if (violations.length === 0) continue;

    failed = true;
    reports.push({ rel, violations: violations.slice(0, 20) });
  }

  if (!failed) {
    console.log("check-english-only: OK");
    return;
  }

  console.error("check-english-only: FAILED\n");
  console.error(
    "Only English (Latin) text is allowed. Exception: `.manual/` is skipped.\n",
  );

  for (const { rel, violations } of reports) {
    console.error(`${rel}:`);
    for (const v of violations) {
      const hex = `U+${v.code.toString(16).toUpperCase().padStart(4, "0")}`;
      console.error(
        `  line ${v.line}, col ${v.col}: ${hex} (${JSON.stringify(v.char)}) — ${v.reason}`,
      );
    }
    if (violations.length >= 20) {
      console.error("  … (more violations omitted)");
    }
    console.error("");
  }

  process.exit(1);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
