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
const crypto = require("crypto");
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

// ensureScratchDeps installs dev-only npm modules into the same scratch
// prefix as puppeteer-core (plan 05-40 M19/F59): the pixel-diff alert
// needs a PNG decoder + differ, and neither enters the repo, the flake,
// or any lockfile — the scratch prefix is disposable and machine-local.
function ensureScratchDeps(specs) {
  fs.mkdirSync(SCRATCH, { recursive: true });
  if (!fs.existsSync(path.join(SCRATCH, "package.json"))) {
    execFileSync("npm", ["init", "-y"], { cwd: SCRATCH, stdio: "ignore" });
  }
  const missing = specs.filter((s) => {
    const name = s.split("@")[0];
    return !fs.existsSync(path.join(SCRATCH, "node_modules", name));
  });
  if (missing.length > 0) {
    execFileSync("npm", ["i", ...missing, "--no-fund", "--no-audit"], {
      cwd: SCRATCH,
      stdio: "inherit",
    });
  }
  return Object.fromEntries(
    specs.map((s) => {
      const name = s.split("@")[0];
      return [name, require(path.join(SCRATCH, "node_modules", name))];
    }),
  );
}

function repoRoot() {
  return execFileSync("git", ["rev-parse", "--show-toplevel"]).toString().trim();
}

// inputHash hashes the codegen INPUTS (sorted .templ sources, app.css) so
// staleness is content-derived: mtimes lie across branch switches and
// fresh clones (plan 05-40 M12/F36 — hash primary, mtime stays as the
// fallback for pre-ledger trees).
function inputHash(webDir) {
  const hash = crypto.createHash("sha256");
  hash.update(
    [
      "static/app.css",
      ...fs
        .readdirSync(webDir)
        .filter((n) => n.endsWith(".templ"))
        .sort(),
    ]
      .map((n) => `${n}\n${fs.readFileSync(path.join(webDir, n))}`)
      .join("\n"),
  );
  return hash.digest("hex");
}

// ensureCodegen regenerates the gitignored UI codegen a clean checkout
// lacks (plan M4/F14): `*_templ.go` when the hashed .templ inputs differ
// from the ledger (or an input is newer — the mtime fallback), via the
// go.mod-pinned `go tool templ` (the devShell templ binary drifts), and
// styles.css when missing or app.css changed (flake-pinned tailwindcss,
// same command as the package preBuild). go:embed is compile-time, so
// this MUST run before buildProbeBinary or a fresh clone probes against
// yesterday's assets.
function ensureCodegen() {
  const root = repoRoot();
  const webDir = path.join(root, "internal", "web");
  const ledger = path.join(webDir, ".codegen-inputs.sha256");

  const current = inputHash(webDir);
  let recorded = null;
  try {
    recorded = fs.readFileSync(ledger, "utf8").trim();
  } catch {
    // First run after adoption (or after clean): the hash check cannot
    // veto, so the mtime/exists fallbacks below decide.
  }
  const inputsChanged = recorded !== null && recorded !== current;

  // Per-file mtime fallback: each generated file against ITS OWN .templ
  // (the old global-newest comparison never converged, because templ skips
  // rewriting unchanged outputs and preserved their older mtimes), and a
  // missing generated file is stale (fresh clone).
  let templStale = false;
  for (const name of fs.readdirSync(webDir)) {
    if (!name.endsWith(".templ")) continue;
    const gen = path.join(webDir, `${name.replace(/\.templ$/, "")}_templ.go`);
    if (
      !fs.existsSync(gen) ||
      fs.statSync(gen).mtimeMs < fs.statSync(path.join(webDir, name)).mtimeMs
    ) {
      templStale = true;
    }
  }
  if (templStale || inputsChanged) {
    console.log(
      `codegen: ${inputsChanged ? ".templ inputs changed since the ledger" : ".templ newer than generated Go"} — running go tool templ generate`,
    );
    execFileSync("go", ["tool", "templ", "generate", "./internal/web/"], {
      cwd: root,
      env: { ...process.env, GOEXPERIMENT: "jsonv2" },
      stdio: "inherit",
    });
  }

  const styles = path.join(webDir, "static", "styles.css");
  if (!fs.existsSync(styles) || inputsChanged) {
    console.log(
      `codegen: styles.css ${!fs.existsSync(styles) ? "missing" : "may predate app.css changes"} — running flake-pinned tailwindcss`,
    );
    execFileSync(
      "nix",
      [
        "develop",
        "-c",
        "tailwindcss",
        "-i",
        "internal/web/static/app.css",
        "-o",
        "internal/web/static/styles.css",
        "--minify",
      ],
      { cwd: root, stdio: "inherit" },
    );
  }

  fs.writeFileSync(ledger, `${current}\n`);
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

// seedFreshAgentTurn ingests a second turn-bearing transcript whose
// timestamp is NOW (the primary fixture carries a fixed 2025 ts, which is
// stale by the staleness probe's standards). The distinct uuid makes it a
// distinct stream, so re-runs never conflict with the fixed fixture.
function seedFreshAgentTurn(bin, dbPath, outDir) {
  const dir = path.join(outDir, "agent-fresh-fixture");
  fs.mkdirSync(dir, { recursive: true });
  const uuid = `probe-fresh-${Math.floor(Date.now() / 1000)}`;
  const now = Math.floor(Date.now() / 1000);
  fs.writeFileSync(
    path.join(dir, "probe-fresh-1.jsonl"),
    [
      JSON.stringify({ type: "start", ts: now, uuid, models: ["gpt-rt"] }),
      JSON.stringify({
        type: "turn",
        role: "agent",
        text: "Fresh probe turn: the boiler is fixed.",
      }),
      JSON.stringify({ type: "end", reason: "normal" }),
    ].join("\n") + "\n",
  );
  execFileSync(bin, ["ingest-agent", "-db", dbPath, "-dir", dir], { stdio: "ignore" });
  return uuid;
}

// seedProbeDB prepares dbPath: smoke corpus on first use (existing DBs are
// kept — re-seeding is a no-op by stream identity), agent fixture always.
// opts.smokeArgs forwards extra smoke flags (e.g. ["-age", "48h",
// "-with-recording", "rec-probe-1"]) for the unseen-state probes.
function seedProbeDB(bin, dbPath, opts = {}) {
  if (!fs.existsSync(dbPath)) {
    execFileSync(bin, ["smoke", "-db", dbPath, ...(opts.smokeArgs || [])], { stdio: "ignore" });
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

// stopProbe drains the serve and RESOLVES ON EXIT (the SIGKILL backstop
// never leaves one squatting on the port). The wait matters for phase
// switches inside one run: a graceful drain keeps the listener open for a
// moment, and a caller that rebinds the port immediately would ping the
// dying process, "succeed", then capture against a corpse.
function stopProbe(serve) {
  return new Promise((resolve) => {
    if (serve.exitCode !== null || serve.signalCode !== null) return resolve();
    serve.once("exit", () => resolve());
    serve.kill("SIGTERM");
    setTimeout(() => {
      if (serve.exitCode === null && serve.signalCode === null) serve.kill("SIGKILL");
    }, 3000).unref();
  });
}

// findFreePort asks the OS for an unused loopback port (plan M4/F13): the
// stale-port stumble is the probe suite's most common friction, so callers
// can default to a free port instead of a pinned one.
function findFreePort() {
  const net = require("net");
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.listen(0, "127.0.0.1", () => {
      const port = srv.address().port;
      srv.close(() => resolve(port));
    });
    srv.on("error", reject);
  });
}

module.exports = {
  SCRATCH,
  buildProbeBinary,
  ensureCodegen,
  ensurePuppeteer,
  ensureScratchDeps,
  findChromium,
  findFreePort,
  repoRoot,
  seedFreshAgentTurn,
  seedProbeDB,
  serveProbe,
  stopProbe,
};
