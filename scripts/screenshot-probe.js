#!/usr/bin/env node
// screenshot-probe — pixel-truth harness for the pbx-lens UI (plan T3).
//
// Captures every operator-facing page at light/dark × 375/1280 px into
// /tmp/lens-shots as labeled PNGs for the eyeball passes (T3.3–T3.5) and as
// the visual instrument later tasks read from (T11 contrast, T12/T14
// verdicts).
//
// Capture tool decision (subtask 3.1): puppeteer-core driving NixOS's
// chromium. Full chrome CLI flags cannot emulate prefers-color-scheme (the
// app's theme default), and puppeteer gives deterministic waits + full-page
// shots. The dependency is dev-tooling only: this script installs
// puppeteer-core into a scratch prefix on first run — nothing enters the Go
// module graph, the flake, or package.json.
//
// Requirements: node, npm, nix chromium (auto-located via $CHROMIUM or
// /nix/store/*-chromium-*/bin/chromium), Go toolchain.
// Usage: node scripts/screenshot-probe.js [--port 18099] [--out /tmp/lens-shots]
"use strict";

const { execFileSync, spawn } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const args = process.argv.slice(2);
const arg = (name, dflt) => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : dflt;
};

const OUT = arg("--out", "/tmp/lens-shots");
const PORT = arg("--port", "18099");
const BASE = `http://127.0.0.1:${PORT}`;
const DB = path.join(OUT, "lens.db");
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

// go run wraps a child we cannot signal cleanly; build a throwaway binary.
function buildAndServe() {
  fs.mkdirSync(OUT, { recursive: true });
  const bin = path.join(OUT, "pbx-lens-probe-bin");
  execFileSync("go", ["build", "-o", bin, "./cmd/pbx-lens"], {
    cwd: repoRoot(),
    env: { ...process.env, GOEXPERIMENT: "jsonv2" },
    stdio: "ignore",
  });
  if (!fs.existsSync(DB)) {
    execFileSync(bin, ["smoke", "-db", DB], { stdio: "ignore" });
  }
  const serve = spawn(bin, ["serve", "-db", DB, "-addr", `127.0.0.1:${PORT}`], { stdio: "ignore" });
  const http = require("http");
  const deadline = Date.now() + 20000;
  return new Promise((resolve, reject) => {
    (function ping() {
      http
        .get(`${BASE}/`, (r) => {
          // A successful ping against a dead child means an older serve
          // still owns the port — captures would silently hit a stale
          // binary + CSS, so fail loudly instead.
          if (serve.exitCode !== null) {
            return reject(
              new Error(`port ${PORT} already in use (stale serve?); kill it or pass --port`),
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

(async () => {
  const puppeteer = ensurePuppeteer();
  const serve = await buildAndServe();
  try {
    const browser = await puppeteer.launch({
      executablePath: findChromium(),
      args: ["--no-sandbox", "--disable-gpu"],
    });

    // Discover two detail URLs (telnyx + freeswitch shape) from the list
    // page. Rows are library clickable rows: <tr data-tc-row-href=…>, not
    // anchors — scrape the attribute (a[href] kept as fallback).
    const seedPage = await browser.newPage();
    await seedPage.goto(`${BASE}/`, { waitUntil: "networkidle0" });
    const hrefs = await seedPage.$$eval('[data-tc-row-href], a[href^="/calls/"]', (els) => [
      ...new Set(els.map((el) => el.getAttribute("data-tc-row-href") || el.getAttribute("href"))),
    ]);
    await seedPage.close();
    const detail1 = hrefs[0] || "/calls/telnyx-conversation/none";
    const detail2 = hrefs[1] || detail1;

    const pages = [
      ["calls", "/"],
      ["search", "/search"],
      ["dashboard", "/dashboard"],
      ["detail-a", detail1],
      ["detail-b", detail2],
    ];
    const themes = ["light", "dark"];
    const viewports = [
      { w: 375, h: 812, tag: "mobile" },
      { w: 1280, h: 900, tag: "desktop" },
    ];

    let n = 0;
    for (const theme of themes) {
      const ctx = await browser.createBrowserContext();
      const page = await ctx.newPage();
      await page.emulateMediaFeatures([{ name: "prefers-color-scheme", value: theme }]);
      for (const vp of viewports) {
        await page.setViewport({ width: vp.w, height: vp.h });
        for (const [name, urlPath] of pages) {
          const resp = await page.goto(`${BASE}${urlPath}`, {
            waitUntil: "networkidle0",
            timeout: 20000,
          });
          const file = path.join(OUT, `${theme}-${vp.tag}-${name}.png`);
          await page.screenshot({ path: file, fullPage: true });
          n++;
          console.log("captured", file, `status=${resp.status()}`, `title=${await page.title()}`);
        }
      }
      await ctx.close();
    }
    await browser.close();
    console.log(`DONE: ${n} PNGs in ${OUT}`);
  } finally {
    serve.kill("SIGTERM");
    // Graceful drain can hang on a stuck keep-alive; never leave a serve
    // squatting on the port for the next run.
    setTimeout(() => serve.kill("SIGKILL"), 3000).unref();
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
