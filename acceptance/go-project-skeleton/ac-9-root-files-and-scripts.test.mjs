// AC-9: root docs and placeholder scripts exist; both scripts are executable bash.
import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const DOCS = ["PROGRESS.md", "REPORT.md", "OUTSIDE_TESTS.md", "SECURITY.md"];
const SCRIPTS = ["scripts/run_gates.sh", "scripts/build_all.sh"];

test("AC-9: PROGRESS.md, REPORT.md, OUTSIDE_TESTS.md, SECURITY.md exist with a title", () => {
  for (const f of DOCS) {
    const p = join(ROOT, f);
    assert.ok(existsSync(p) && statSync(p).isFile(), `${f} missing at repo root`);
    assert.match(readFileSync(p, "utf8"), /^#\s+\S/m, `${f} must have a markdown title`);
  }
});

test("AC-9: scripts/run_gates.sh and scripts/build_all.sh exist, are executable bash with strict mode", () => {
  for (const f of SCRIPTS) {
    const p = join(ROOT, f);
    assert.ok(existsSync(p) && statSync(p).isFile(), `${f} missing`);
    assert.ok((statSync(p).mode & 0o111) !== 0, `${f} must be executable`);
    const src = readFileSync(p, "utf8");
    assert.equal(src.split(/\r?\n/)[0], "#!/usr/bin/env bash", `${f} must start with #!/usr/bin/env bash`);
    assert.match(src, /^\s*set -euo pipefail\s*$/m, `${f} must set -euo pipefail`);
  }
});
