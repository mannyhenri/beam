// Shared helpers for the simnet-harness acceptance tests: build the driver
// (internal/simnet/cmd/simnet) into a temp dir, run it, and read its JSON lines.
import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { mkdtempSync, realpathSync, rmSync } from "node:fs";
import { connect } from "node:net";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
const GO = process.env.GO || "go";

// A fresh temp dir (symlinks resolved, so /var and /private/var compare equal on macOS).
export function tempDir(t) {
  const d = realpathSync(mkdtempSync(join(tmpdir(), "simnet-ac-")));
  t.after(() => rmSync(d, { recursive: true, force: true }));
  return d;
}

// Builds the driver; a missing or broken driver fails the test here.
export function buildDriver(t) {
  const out = join(tempDir(t), "simnet-driver");
  const r = spawnSync(GO, ["build", "-o", out, "./internal/simnet/cmd/simnet"], {
    cwd: ROOT,
    encoding: "utf8",
    timeout: 300000,
    env: { ...process.env, GOTOOLCHAIN: "local", GOPROXY: "off", GOFLAGS: "", CGO_ENABLED: "0" },
  });
  if (r.error) assert.fail(`could not run go build: ${r.error.message}`);
  assert.equal(r.status, 0, `go build ./internal/simnet/cmd/simnet failed:\n${r.stdout}${r.stderr}`);
  return out;
}

// Starts the driver with stdin as a pipe. Returns helpers to read stdout lines and await exit.
export function startDriver(t, bin, args) {
  const child = spawn(bin, args, { cwd: ROOT, stdio: ["pipe", "pipe", "pipe"] });
  let stdout = "";
  let stderr = "";
  const lines = [];
  const waiters = [];
  let buf = "";
  child.stdout.setEncoding("utf8");
  child.stderr.setEncoding("utf8");
  child.stdout.on("data", (chunk) => {
    stdout += chunk;
    buf += chunk;
    let i;
    while ((i = buf.indexOf("\n")) >= 0) {
      const line = buf.slice(0, i).trim();
      buf = buf.slice(i + 1);
      if (line === "") continue;
      lines.push(line);
      while (waiters.length) waiters.shift()();
    }
  });
  child.stderr.on("data", (chunk) => (stderr += chunk));

  let exitInfo = null;
  const exited = new Promise((res) => {
    child.on("exit", (code, signal) => {
      exitInfo = { code, signal };
      while (waiters.length) waiters.shift()();
      res(exitInfo);
    });
  });

  // Never leave the driver or its machines behind, even if an assertion fails.
  const pids = new Set();
  t.after(() => {
    if (exitInfo === null) child.kill("SIGKILL");
    for (const pid of pids) {
      try {
        process.kill(pid, "SIGKILL");
      } catch {}
    }
  });

  let next = 0;
  async function nextLine(ms) {
    const deadline = Date.now() + ms;
    while (next >= lines.length) {
      if (exitInfo !== null) {
        assert.fail(`driver exited (${JSON.stringify(exitInfo)}) before printing line ${next + 1}\nstdout:\n${stdout}\nstderr:\n${stderr}`);
      }
      const left = deadline - Date.now();
      if (left <= 0) assert.fail(`no line ${next + 1} from driver within ${ms} ms\nstdout:\n${stdout}\nstderr:\n${stderr}`);
      await new Promise((res) => {
        const timer = setTimeout(res, left);
        waiters.push(() => {
          clearTimeout(timer);
          res();
        });
      });
    }
    return lines[next++];
  }

  async function waitExit(ms) {
    let timer;
    const timeout = new Promise((res) => (timer = setTimeout(() => res(null), ms)));
    const r = await Promise.race([exited, timeout]);
    clearTimeout(timer);
    if (r === null) assert.fail(`driver did not exit within ${ms} ms\nstdout:\n${stdout}\nstderr:\n${stderr}`);
    return r;
  }

  return {
    child,
    nextLine,
    waitExit,
    track: (pid) => pids.add(pid),
    allLines: () => lines.slice(),
    output: () => ({ stdout, stderr }),
  };
}

// Parses one machine line and checks its shape.
export function parseMachine(line) {
  let m;
  try {
    m = JSON.parse(line);
  } catch {
    assert.fail(`driver line is not JSON: ${line}`);
  }
  for (const k of ["name", "pid", "addr", "home", "log"]) {
    assert.ok(k in m, `machine line lacks "${k}": ${line}`);
  }
  assert.ok(Number.isInteger(m.pid) && m.pid > 0, `pid is not a positive integer: ${line}`);
  const match = /^(\d+\.\d+\.\d+\.\d+):(\d+)$/.exec(m.addr);
  assert.ok(match, `addr is not host:port: ${line}`);
  m.host = match[1];
  m.port = Number(match[2]);
  assert.ok(m.port > 0 && m.port < 65536, `bad port in ${line}`);
  return m;
}

export function isAlive(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch (e) {
    return e.code === "EPERM";
  }
}

// Connects to host:port and reads everything until the peer closes.
export function fetchLine(host, port, ms = 10000) {
  return new Promise((res, rej) => {
    const s = connect({ host, port });
    let data = "";
    const timer = setTimeout(() => {
      s.destroy();
      rej(new Error(`no answer from ${host}:${port} within ${ms} ms`));
    }, ms);
    s.setEncoding("utf8");
    s.on("data", (c) => (data += c));
    s.on("end", () => {
      clearTimeout(timer);
      res(data);
    });
    s.on("error", (e) => {
      clearTimeout(timer);
      rej(e);
    });
  });
}

// Resolves to the connect error code (e.g. "ECONNREFUSED"), or "connected".
export function tryConnect(host, port, ms = 10000) {
  return new Promise((res) => {
    const s = connect({ host, port });
    const timer = setTimeout(() => {
      s.destroy();
      res("timeout");
    }, ms);
    s.on("connect", () => {
      clearTimeout(timer);
      s.destroy();
      res("connected");
    });
    s.on("error", (e) => {
      clearTimeout(timer);
      res(e.code);
    });
  });
}

// PIDs of running processes whose command line contains `needle` (the unique driver path).
export function processesMatching(needle) {
  const r = spawnSync("ps", ["-axo", "pid=,command="], { encoding: "utf8" });
  if (r.error || r.status !== 0) assert.fail(`ps failed: ${r.error?.message || r.stderr}`);
  return r.stdout
    .split("\n")
    .filter((l) => l.includes(needle))
    .map((l) => Number(l.trim().split(/\s+/)[0]))
    .filter((pid) => pid !== process.pid && isAlive(pid));
}

export const EXPECTED = [
  ["A", "127.0.0.2"],
  ["B", "127.0.0.3"],
  ["C", "127.0.0.4"],
];
