#!/usr/bin/env node
// verify-ops-verdict — the browser-level probe for the /ops stale verdict
// (plan 2026-10-18 T20). Go tests pin the decode and render rules; this
// pins the LIVE composition the probes could never reach before: a serve
// with the poller counters wired (via the -wired-ops dev fixture) actually
// renders the stale-verdict alert on /ops, and /api/ops answers the same
// verdict as JSON.
//
// Contract O (ops verdict, canned fixture):
//   1. GET /ops on a -wired-ops serve shows the "Poller pass is stale"
//      error alert (feedback.Alert, not just the badge).
//   2. GET /api/ops answers stale:true with both canned source rows.
//
// Usage: node scripts/verify-ops-verdict.js [--port 18118] [--out /tmp/lens-ops-verdict]
"use strict";

const path = require("path");
const probe = require("./lib/ui-probe");

const args = process.argv.slice(2);
const arg = (name, dflt) => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : dflt;
};

const OUT = arg("--out", "/tmp/lens-ops-verdict");
const DB = path.join(OUT, "lens.db");

function fail(msg) {
  console.error("FAIL:", msg);
  process.exitCode = 1;
}

(async () => {
  const PORT = args.includes("--port") ? arg("--port", null) : await probe.findFreePort();
  const BASE = `http://127.0.0.1:${PORT}`;
  console.log("port:", PORT);

  probe.ensureCodegen();
  const puppeteer = probe.ensurePuppeteer();
  const bin = probe.buildProbeBinary(OUT);
  probe.seedProbeDB(bin, DB);
  const serve = await probe.serveProbe(bin, DB, PORT, ["-wired-ops"]);
  const browser = await puppeteer.launch({
    executablePath: probe.findChromium(),
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 900 });

    const resp = await page.goto(`${BASE}/ops`, { waitUntil: "networkidle0" });
    console.log("/ops ->", resp.status());

    const alert = await page.evaluate(() => {
      const el = [...document.querySelectorAll("[role='alert'], .lens-alert, div")]
        .find((n) => n.textContent.includes("Poller pass is stale"));
      return el
        ? { found: true, text: el.textContent.slice(0, 160) }
        : { found: false };
    });
    console.log("stale alert rendered:", alert.found, alert.found ? `(${alert.text.trim()}…)` : "");
    if (!alert.found) fail("-wired-ops /ops did not render the 'Poller pass is stale' alert");
    if (alert.found && !alert.text.includes("2h")) {
      fail("stale alert did not carry the canned 2h pass age");
    }

    const api = await page.evaluate(async () => {
      const r = await fetch("/api/ops");
      return { status: r.status, body: await r.json() };
    });
    console.log("/api/ops ->", api.status, "stale:", api.body?.stale);
    if (api.status !== 200) fail("/api/ops status " + api.status);
    if (api.body?.stale !== true) fail("/api/ops stale != true on the -wired-ops fixture");
    if (!api.body?.sources?.["freeswitch-cdr"]) {
      fail("/api/ops missing the canned freeswitch-cdr source row");
    }

    console.log(process.exitCode ? "RESULT: FAILED" : "RESULT: ALL PASS");
  } finally {
    await browser.close();
    await probe.stopProbe(serve);
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
