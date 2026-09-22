import { mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const scanRoot = join(root, ".ui8px", "scan");
const hostExt = new Set([".svelte", ".html"]);

function extname(name) {
  const index = name.lastIndexOf(".");
  return index < 0 ? "" : name.slice(index).toLowerCase();
}

async function walk(dir) {
  const entries = await readdir(dir, { withFileTypes: true }).catch(() => null);
  if (!entries) return [];
  const files = [];
  for (const entry of entries) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) files.push(...(await walk(path)));
    else if (hostExt.has(extname(entry.name))) files.push(path);
  }
  return files;
}

const keepIgnore = join(scanRoot, ".gitignore");
let ignoreText = "*\n!.gitignore\n";
try {
  ignoreText = await readFile(keepIgnore, "utf8");
} catch {
  /* first run */
}
await rm(scanRoot, { recursive: true, force: true });
await mkdir(scanRoot, { recursive: true });
await writeFile(keepIgnore, ignoreText);

let mirrored = 0;
for (const file of await walk(join(root, "src"))) {
  const dest = join(scanRoot, `${relative(root, file)}.html`);
  await mkdir(dirname(dest), { recursive: true });
  await writeFile(dest, await readFile(file, "utf8"));
  mirrored += 1;
}

console.log(`ui8px scan: mirrored ${mirrored} Svelte hosts as HTML.`);
