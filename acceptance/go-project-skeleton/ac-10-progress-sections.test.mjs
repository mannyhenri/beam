// AC-10: PROGRESS.md has the six required sections; Decisions records module path `beam`.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");

const REQUIRED = [
  ["Current phase", /^current phase$/i],
  ["Time spent", /^time spent$/i],
  ["Go environment", /^go environment$/i],
  ["Test results", /^test results$/i],
  ["Decisions that differ from the spec", /^decisions that differ from (the )?spec$/i],
  ["Cuts taken", /^cuts taken$/i],
];

function sections(md) {
  const out = [];
  let cur = null;
  for (const line of md.split(/\r?\n/)) {
    const m = /^#{1,6}\s+(.+?)\s*#*\s*$/.exec(line);
    if (m) { cur = { heading: m[1].trim(), body: "" }; out.push(cur); }
    else if (cur) cur.body += line + "\n";
  }
  return out;
}

function readProgress() {
  const p = join(ROOT, "PROGRESS.md");
  assert.ok(existsSync(p), "PROGRESS.md missing at repo root");
  return readFileSync(p, "utf8");
}

test("AC-10: PROGRESS.md has all six section headings", () => {
  const headings = sections(readProgress()).map((s) => s.heading);
  for (const [name, re] of REQUIRED) {
    assert.equal(headings.filter((h) => re.test(h)).length, 1, `expected exactly one \`${name}\` heading; headings: ${headings.join(" | ")}`);
  }
});

test("AC-10: Decisions section records module path `beam`", () => {
  const dec = sections(readProgress()).find((s) => REQUIRED[4][1].test(s.heading));
  assert.ok(dec, "missing `Decisions that differ from the spec` section");
  const line = dec.body.split(/\r?\n/).find((l) => /module path/i.test(l) && /`beam`|\bbeam\b/.test(l));
  assert.ok(line, `Decisions section must record the module path \`beam\`, got:\n${dec.body}`);
});
