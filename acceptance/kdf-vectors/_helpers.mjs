// Shared helpers for the kdf-vectors acceptance tests (HKDF and PBKDF2 fallbacks).
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { cpSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
export const GO = process.env.GO || "go";

export const HKDF_FILE = "internal/session/hkdf_vectors_test.go";
export const PBKDF2_FILE = "internal/pairing/pbkdf2_vectors_test.go";

export const SKELETON_PACKAGES = [
  "beam/cmd/beam",
  ...["ws", "relay", "identity", "pairing", "session", "mux", "transfer", "sysinfo", "tunnel", "ui", "simnet"].map((n) => `beam/internal/${n}`),
  // The locked simnet driver (simnet-harness / no-direct-check) is a 13th package.
  "beam/internal/simnet/cmd/simnet",
].sort();

const range = (a, b) => Buffer.from(Array.from({ length: b - a + 1 }, (_, i) => a + i));

// RFC 5869 Appendix A, HKDF with SHA-256.
export const HKDF_VECTORS = {
  "A.1": {
    ikm: Buffer.alloc(22, 0x0b), salt: range(0x00, 0x0c), info: range(0xf0, 0xf9), L: 42,
    prk: "077709362c2e32df0ddc3f0dc47bba6390b6c73bb50f9c3122ec844ad7c2b3e5",
    okm: "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865",
  },
  "A.2": {
    ikm: range(0x00, 0x4f), salt: range(0x60, 0xaf), info: range(0xb0, 0xff), L: 82,
    prk: "06a6b88c5853361a06104c9ceb35b45cef760014904671014a193f40c15fc244",
    okm: "b11e398dc80327a1c8e7f78c596a49344f012eda2d4efad8a050cc4c19afa97c" +
      "59045a99cac7827271cb41c65e590e09da3275600c2f09b8367793a9aca3db71" +
      "cc30c58179ec3e87c14c01d5c1f3434f1d87",
  },
  "A.3": {
    ikm: Buffer.alloc(22, 0x0b), salt: Buffer.alloc(0), info: Buffer.alloc(0), L: 42,
    prk: "19ef24a32c717b167f33a91d6f648bdf96596776afdb6377ac434c1c293ccb04",
    okm: "8da4e775a563c18f715f802a063c5a31b8a11f5c5ee1879ec3454e5f3c738d2d9d201395faa4b61a96c8",
  },
};

// RFC 7914 section 11, PBKDF2-HMAC-SHA256.
export const PBKDF2_VECTORS = {
  "c=1": {
    P: "passwd", S: "salt", c: 1, dkLen: 64,
    dk: "55ac046e56e3089fec1691c22544b605f94185216dde0465e68b9d57c20dacbc" +
      "49ca9cccf179b645991664b39d77ef317c71b845b1e30bd509112041d3a19783",
  },
  "c=80000": {
    P: "Password", S: "NaCl", c: 80000, dkLen: 64,
    dk: "4ddcd8f60b98be21830cee5ef22701f9641a4418d04c0414aeff08876b34ab56" +
      "a1d425a1225833549adb841b51c9b3176a272bdebba1d078478f62b397f33c8d",
  },
};

export function go(args, { cwd = ROOT, env = {} } = {}) {
  const r = spawnSync(GO, args, {
    cwd,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "-count=1", ...env },
  });
  if (r.error) assert.fail(`could not run \`${GO} ${args.join(" ")}\`: ${r.error.message}`);
  return { code: r.status, stdout: r.stdout, stderr: r.stderr, out: r.stdout + r.stderr };
}

export function requireGoMod() {
  assert.ok(existsSync(join(ROOT, "go.mod")), "go.mod missing at repo root (the go-project-skeleton must be in place)");
}

export function readRepoFile(rel) {
  const p = join(ROOT, rel);
  assert.ok(existsSync(p), `${rel} missing`);
  return readFileSync(p, "utf8");
}

// Asserts a `go test -v` run selected and passed a top-level test whose name matches `nameRe`,
// with nothing failing or skipped.
export function assertVerbosePass(r, nameRe, label) {
  assert.equal(r.code, 0, `${label} failed:\n${r.out}`);
  assert.doesNotMatch(r.out, /no tests to run/, `${label} ran no tests:\n${r.out}`);
  assert.doesNotMatch(r.out, /--- (FAIL|SKIP)/, `${label} has failing or skipped tests:\n${r.out}`);
  const passed = [...r.out.matchAll(/^--- PASS: (\S+)/gm)].map((m) => m[1]);
  assert.ok(passed.some((n) => nameRe.test(n)), `${label}: no top-level test matching ${nameRe} passed:\n${r.out}`);
}

// Go source with comments blanked, then reduced to hex-comparable text: `0x`/`\x` prefixes,
// quotes, `+`, commas, braces and whitespace are dropped and the rest lowercased. `idx[i]` is
// the offset in the original source of normalized character i.
function normalizeGo(src) {
  const masked = src.replace(/\/\*[\s\S]*?\*\/|\/\/[^\n]*/g, (m) => m.replace(/[^\n]/g, " "));
  let norm = "";
  const idx = [];
  for (let i = 0; i < masked.length; i++) {
    const c = masked[i];
    if ((c === "0" || c === "\\") && (masked[i + 1] === "x" || masked[i + 1] === "X")) { i++; continue; }
    if (/[\s"`+,{}]/.test(c)) continue;
    norm += c.toLowerCase();
    idx.push(i);
  }
  return { norm, idx };
}

// Offsets in `src` (code, not comments) of every occurrence of `hex` written as a hex literal.
export function findHexLiteral(src, hex) {
  const { norm, idx } = normalizeGo(src);
  const out = [];
  for (let at = norm.indexOf(hex); at !== -1; at = norm.indexOf(hex, at + 1)) {
    out.push(idx.slice(at, at + hex.length));
  }
  return out;
}

export function assertHasHexLiteral(src, hex, what) {
  assert.ok(findHexLiteral(src, hex).length > 0, `${what} (${hex}) is not written in the test code as a hex literal`);
}

function flipHex(ch) {
  const v = ((parseInt(ch, 16) + 1) % 16).toString(16);
  return ch === ch.toUpperCase() ? v.toUpperCase() : v;
}

// Copies the module to a temp dir, flips one hex digit (at position `pos` of `hex`) in every
// code occurrence of `hex` inside `rel`, and runs `go test ./<pkg> -run <run>` there.
export function runMutated(rel, hex, pos, run) {
  const src = readRepoFile(rel);
  const hits = findHexLiteral(src, hex);
  assert.ok(hits.length > 0, `${hex} not found as a hex literal in ${rel}`);
  const chars = src.split("");
  for (const offsets of hits) chars[offsets[pos]] = flipHex(chars[offsets[pos]]);
  const tmp = mkdtempSync(join(tmpdir(), "kdf-vectors-"));
  try {
    cpSync(ROOT, tmp, {
      recursive: true,
      filter: (p) => !/[\\/](\.git|node_modules|acceptance)([\\/]|$)/.test(p.slice(ROOT.length)),
    });
    writeFileSync(join(tmp, rel), chars.join(""));
    return go(["test", "./" + dirname(rel), "-run", run], { cwd: tmp });
  } finally {
    rmSync(tmp, { recursive: true, force: true });
  }
}

// The vector test must fail when the expected value is corrupted at its first or last digit.
export function assertValueIsChecked(rel, hex, run, what) {
  for (const pos of [0, hex.length - 1]) {
    const r = runMutated(rel, hex, pos, run);
    assert.notEqual(r.code, 0, `${rel} still passes with ${what} corrupted at hex digit ${pos}; it does not check ${what} byte for byte:\n${r.out}`);
    assert.match(r.out, /--- FAIL/, `${rel} with ${what} corrupted should fail a test, got:\n${r.out}`);
  }
}

// Copies the module to a temp dir and reroutes every `hkdf.<fn>(` call in HKDF_FILE through a
// wrapper that flips the first byte, then (separately) the last byte, of its output; the HKDF
// test must fail both times, proving that function's output is compared in full.
export function assertHkdfOutputIsChecked(fn) {
  const src = readRepoFile(HKDF_FILE);
  const { norm } = normalizeGo(src);
  assert.ok(norm.includes(`hkdf.${fn.toLowerCase()}(`), `${HKDF_FILE} never calls hkdf.${fn}`);
  const masked = src.replace(/\/\*[\s\S]*?\*\/|\/\/[^\n]*/g, (m) => m.replace(/[^\n]/g, " "));
  let out = "";
  let last = 0;
  for (const m of masked.matchAll(new RegExp(`\\bhkdf\\.${fn}\\s*\\(`, "g"))) {
    out += src.slice(last, m.index) + `kdfvecCorrupt${fn}(`;
    last = m.index + m[0].length;
  }
  out += src.slice(last);
  const params = {
    Extract: ["secret, salt []byte", "secret, salt"],
    Expand: ["prk []byte, info string, n int", "prk, info, n"],
    Key: ["secret, salt []byte, info string, n int", "secret, salt, info, n"],
  }[fn];
  for (const [where, index] of [["first", "0"], ["last", "len(b)-1"]]) {
    const shim = `package session

import (
	"crypto/hkdf"
	"hash"
)

func kdfvecCorrupt${fn}[H hash.Hash](h func() H, ${params[0]}) ([]byte, error) {
	b, err := hkdf.${fn}(h, ${params[1]})
	if len(b) > 0 {
		b[${index}] ^= 0x01
	}
	return b, err
}
`;
    const tmp = mkdtempSync(join(tmpdir(), "kdf-vectors-"));
    try {
      cpSync(ROOT, tmp, {
        recursive: true,
        filter: (p) => !/[\\/](\.git|node_modules|acceptance)([\\/]|$)/.test(p.slice(ROOT.length)),
      });
      writeFileSync(join(tmp, HKDF_FILE), out);
      writeFileSync(join(tmp, dirname(HKDF_FILE), "zz_kdfvec_corrupt_test.go"), shim);
      const r = go(["test", "./" + dirname(HKDF_FILE), "-run", "HKDF"], { cwd: tmp });
      assert.notEqual(r.code, 0, `${HKDF_FILE} still passes when hkdf.${fn} returns a corrupted ${where} byte; its output is not checked byte for byte:\n${r.out}`);
      assert.match(r.out, /--- FAIL/, `with the ${where} byte of hkdf.${fn} corrupted the HKDF test should fail, got:\n${r.out}`);
    } finally {
      rmSync(tmp, { recursive: true, force: true });
    }
  }
}

export function sections(md) {
  const out = [];
  let cur = null;
  for (const line of md.split(/\r?\n/)) {
    const m = /^#{1,6}\s+(.+?)\s*#*\s*$/.exec(line);
    if (m) { cur = { heading: m[1].trim(), body: "" }; out.push(cur); }
    else if (cur) cur.body += line + "\n";
  }
  return out;
}
