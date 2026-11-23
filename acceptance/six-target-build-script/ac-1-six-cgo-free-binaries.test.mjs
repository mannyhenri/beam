// AC-1: `bash scripts/build_all.sh` from the repo root exits 0 and leaves exactly six
// CGO_ENABLED=0 binaries in dist/, one per GOOS/GOARCH target.
import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdirSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const DIST = join(ROOT, "dist");
const GO = process.env.GO || "go";

const TARGETS = [
  ["darwin", "amd64", "beam-darwin-amd64"],
  ["darwin", "arm64", "beam-darwin-arm64"],
  ["linux", "amd64", "beam-linux-amd64"],
  ["linux", "arm64", "beam-linux-arm64"],
  ["windows", "amd64", "beam-windows-amd64.exe"],
  ["windows", "arm64", "beam-windows-arm64.exe"],
];

test("AC-1: build_all.sh exits 0 and produces six CGO_ENABLED=0 binaries", () => {
  // A stale binary from an earlier run must be cleaned away.
  mkdirSync(DIST, { recursive: true });
  writeFileSync(join(DIST, "beam-freebsd-amd64"), "stale");

  // Hostile caller environment: the script must set CGO_ENABLED/GOOS/GOARCH itself.
  const r = spawnSync("bash", ["scripts/build_all.sh"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 600000,
    env: {
      ...process.env,
      GOTOOLCHAIN: "local",
      GOPROXY: "off",
      GOFLAGS: "",
      CGO_ENABLED: "1",
      GOOS: "plan9",
      GOARCH: "386",
    },
  });
  if (r.error) assert.fail(`could not run build_all.sh: ${r.error.message}`);
  assert.equal(r.status, 0, `build_all.sh exited ${r.status}:\n${r.stdout}${r.stderr}`);

  const binaries = readdirSync(DIST).filter((f) => f.startsWith("beam-")).sort();
  assert.deepEqual(
    binaries,
    TARGETS.map((t) => t[2]).sort(),
    "dist/ must contain exactly the six beam-<goos>-<goarch>[.exe] binaries",
  );

  for (const [goos, goarch, name] of TARGETS) {
    const v = spawnSync(GO, ["version", "-m", join(DIST, name)], {
      cwd: ROOT,
      encoding: "utf8",
      timeout: 60000,
      env: { ...process.env, GOTOOLCHAIN: "local", GOFLAGS: "" },
    });
    if (v.error) assert.fail(`could not run go version -m: ${v.error.message}`);
    assert.equal(v.status, 0, `go version -m ${name} failed:\n${v.stdout}${v.stderr}`);
    const settings = new Set(
      v.stdout
        .split("\n")
        .map((l) => l.trim().split(/\s+/))
        .filter((p) => p[0] === "build" && p[1])
        .map((p) => p[1]),
    );
    assert.ok(settings.has("CGO_ENABLED=0"), `${name}: expected CGO_ENABLED=0\n${v.stdout}`);
    assert.ok(settings.has(`GOOS=${goos}`), `${name}: expected GOOS=${goos}\n${v.stdout}`);
    assert.ok(settings.has(`GOARCH=${goarch}`), `${name}: expected GOARCH=${goarch}\n${v.stdout}`);
  }
});
