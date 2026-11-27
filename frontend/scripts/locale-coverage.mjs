import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const localesDir = path.join(path.dirname(fileURLToPath(import.meta.url)), "../src/locales");
const sourceId = "en";
const compact = process.argv.includes("--compact");

function flatten(value, prefix, out) {
  if (value !== null && typeof value === "object" && !Array.isArray(value)) {
    for (const [key, child] of Object.entries(value)) {
      flatten(child, prefix ? `${prefix}.${key}` : key, out);
    }
    return;
  }
  if (typeof value !== "string") {
    throw new Error(`${prefix || "<root>"} is not a string or object`);
  }
  if (!prefix) {
    throw new Error("locale file has a string at the root");
  }
  out.set(prefix, value);
}

function load(file) {
  const tree = JSON.parse(readFileSync(file, "utf8"));
  const messages = new Map();
  flatten(tree, "", messages);
  return messages;
}

function percent(part, total) {
  if (total === 0) {
    return "0.0%";
  }
  return `${((part / total) * 100).toFixed(1)}%`;
}

const files = readdirSync(localesDir)
  .filter((name) => name.endsWith(".json"))
  .sort((a, b) => {
    const aId = a.slice(0, -".json".length);
    const bId = b.slice(0, -".json".length);
    if (aId === sourceId) return -1;
    if (bId === sourceId) return 1;
    return aId.localeCompare(bId);
  });

const sourceFile = `${sourceId}.json`;
if (!files.includes(sourceFile)) {
  console.error(`missing source locale ${sourceFile} in ${localesDir}`);
  process.exit(1);
}

const catalogs = new Map();
for (const file of files) {
  const id = file.slice(0, -".json".length);
  try {
    catalogs.set(id, load(path.join(localesDir, file)));
  } catch (err) {
    console.error(`${file}: ${err instanceof Error ? err.message : err}`);
    process.exit(1);
  }
}

const source = catalogs.get(sourceId);
const sourceKeys = [...source.keys()].sort();

if (compact) {
  console.log(`${sourceId} ${percent(sourceKeys.length, sourceKeys.length)}`);
} else {
  console.log(`${sourceId}  source  ${sourceKeys.length} keys`);
}

for (const [id, messages] of catalogs) {
  if (id === sourceId) {
    continue;
  }
  const missing = [];
  const identical = [];
  let filled = 0;
  for (const key of sourceKeys) {
    if (!messages.has(key) || messages.get(key).trim() === "") {
      missing.push(key);
      continue;
    }
    filled += 1;
    if (messages.get(key) === source.get(key)) {
      identical.push(key);
    }
  }
  const extra = [...messages.keys()].filter((key) => !source.has(key)).sort();
  if (compact) {
    console.log(`${id} ${percent(filled, sourceKeys.length)}`);
    continue;
  }
  console.log(`${id}  ${percent(filled, sourceKeys.length)}  ${filled}/${sourceKeys.length}`);
  if (identical.length > 0) {
    console.log(`  same as ${sourceId} (${identical.length})`);
    for (const key of identical) {
      console.log(`    ${key}`);
    }
  }
  if (missing.length > 0) {
    console.log(`  missing (${missing.length})`);
    for (const key of missing) {
      console.log(`    ${key}`);
    }
  }
  if (extra.length > 0) {
    console.log(`  not in ${sourceId} (${extra.length})`);
    for (const key of extra) {
      console.log(`    ${key}`);
    }
  }
}
