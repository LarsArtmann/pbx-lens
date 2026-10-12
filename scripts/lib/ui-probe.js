"use strict";
// ui-probe — shared bootstrap for the browser-level UI instruments
// (scripts/screenshot-probe.js, scripts/verify-ui-contract.js): chromium
// discovery, the puppeteer-core scratch install, the probe-binary build,
// corpus seeding (smoke + agent transcript fixture), and the serve
// lifecycle. Nothing here enters the Go module graph, the flake, or
// package.json — puppeteer-core installs into a scratch prefix on first
// use and is dev tooling only.
//
// Chromium decision record (see screenshot-probe.js history): puppeteer
// drives NixOS's chromium because CLI chrome flags cannot emulate
// prefers-color-scheme; steam-run/chrome-headless-shell lack libnss3.
const { execFileSync, spawn } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const SCRATCH = path.join(os.tmpdir(), "lens-shot-probe");

function findChromium() {
  if (process.env.CHROMIUM) return process.env.CHROMIUM;
  const store = "/nix/store";
  const hits = fs
    .readdirSync(store)
    .filter((n) => /^[\w-]+-chromium-\d/.test(n))
    .map((n) => path.join(store, n, "bin", "chromium"))
    .filter((p) => fs.existsSync(p))
    .sort();
  if (hits.length === 0) {
    throw new Error("no chromium found; set CHROMIUM=/path/to/chromium");
  }
  return hits[hits.length - 1];
}

function ensurePuppeteer() {
  const mod = path.join(SCRATCH, "node_modules", "puppeteer-core");
  if (!fs.existsSync(mod)) {
    fs.mkdirSync(SCRATCH, { recursive: true });
    if (!fs.existsSync(path.join(SCRATCH, "package.json"))) {
      execFileSync("npm", ["init", "-y"], { cwd: SCRATCH, stdio: "ignore" });
    }
    execFileSync("npm", ["i", "puppeteer-core@23", "--no-fund", "--no-audit"], {
      cwd: SCRATCH,
      stdio: "inherit",
    });
  }
  return require(path.join(mod));
}

function repoRoot() {
  return execFileSync("git", ["rev-parse", "--show-toplevel"]).toString().trim();
}

// buildProbeBinary compiles ./cmd/pbx-lens to a throwaway path (go run
// wraps a child we cannot signal cleanly). Returns the binary path.
function buildProbeBinary(outDir) {
  fs.mkdirSync(outDir, { recursive: true });
  const bin = path.join(outDir, "pbx-lens-probe-bin");
  execFileSync("go", ["build", "-o", bin, "./cmd/pbx-lens"], {
    cwd: repoRoot(),
    env: { ...process.env, GOEXPERIMENT: "jsonv2" },
    stdio: "ignore",
  });
  return bin;
}

// seedAgentTranscript ingests one turn-bearing agent transcript
// ("thermostat" as the match term) into dbPath. The bare smoke corpus has
// no transcript turns, so browser-level contracts about the ?q= match log
// need this fixture; re-ingest is idempotent (same ContentHash replays to
// zero events).
function seedAgentTranscript(bin, dbPath, outDir) {
  const dir = path.join(outDir, "agent-fixture");
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(
    path.join(dir, "probe-egg-1.jsonl"),
    [
      JSON.stringify({ type: "start", ts: 1760041885, uuid: "probe-egg-1", models: ["gpt-rt"] }),
      JSON.stringify({
        type: "turn",
        role: "agent",
        text: "Hi, this is Woolly calling about your thermostat booking.",
      }),
      JSON.stringify({
        type: "turn",
        role: "caller",
        text: "Yes, the thermostat is broken, when can someone come?",
      }),
      JSON.stringify({ type: "end", reason: "caller_hungup" }),
    ].join("\n") + "\n",
  );
  execFileSync(bin, ["ingest-agent", "-db", dbPath, "-dir", dir], { stdio: "ignore" });
}

// seedProbeDB prepares dbPath: smoke corpus on first use (existing DBs are
// kept — re-seeding is a no-op by stream identity), agent fixture always.
function seedProbeDB(bin, dbPath) {
  if (!fs.existsSync(dbPath)) {
    execFileSync(bin, ["smoke", "-db", dbPath], { stdio: "ignore" });
  }
  seedAgentTranscript(bin, dbPath, path.dirname(dbPath));
}

// serveProbe spawns the built binary and resolves once / responds.
// A successful ping against a dead child means an older serve still owns
// the port — captures would silently hit a stale binary + CSS, so it
// fails loudly instead.
function serveProbe(bin, dbPath, port) {
  const base = `http://127.0.0.1:${port}`;
  const serve = spawn(bin, ["serve", "-db", dbPath, "-addr", `127.0.0.1:${port}`], {
    stdio: "ignore",
  });
  const http = require("http");
  const deadline = Date.now() + 20000;
  return new Promise((resolve, reject) => {
    (function ping() {
      http
        .get(`${base}/`, (r) => {
          r.resume();
          if (serve.exitCode !== null) {
            return reject(
              new Error(`port ${port} already in use (stale serve?); kill it or pass --port`),
            );
          }
          return resolve(serve);
        })
        .on("error", retry);
      function retry() {
        if (Date.now() > deadline) return reject(new Error("serve did not come up"));
        setTimeout(ping, 250);
      }
    })();
  });
}

// stopProbe drains the serve; the SIGKILL backstop never leaves one
// squatting on the port for the next run.
function stopProbe(serve) {
  serve.kill("SIGTERM");
  setTimeout(() => serve.kill("SIGKILL"), 3000).unref();
}

module.exports = {
  SCRATCH,
  buildProbeBinary,
  ensurePuppeteer,
  findChromium,
  repoRoot,
  seedProbeDB,
  serveProbe,
  stopProbe,
};
