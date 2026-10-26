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

const probe = require("./lib/ui-probe");

const fail = probe.fail;

(async () => {
  const {
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    base: BASE,
  } = await probe.probeSession("/tmp/lens-ops-verdict");
  console.log("port:", PORT);

  const { serve, browser, page } = await probe.launchProbe({
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    serveArgs: ["-wired-ops"],
  });

  try {
    const resp = await page.goto(`${BASE}/ops`, { waitUntil: "networkidle0" });
    console.log("/ops ->", resp.status());

    const alert = await page.evaluate(() => {
      const el = [...document.querySelectorAll("[role='alert'], [role='status']")].find((n) =>
        n.textContent.includes("Poller pass is stale"),
      );
      return el ? { found: true, text: el.textContent.slice(0, 200) } : { found: false };
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

    // Contract O.3 (2026-10-26): the retention SLO section rides the same
    // JSON. The probe serve runs without -max-age, so active must be false;
    // the nullable facts are presence-checked (null = unknown/empty is a
    // legal answer, a MISSING key is not).
    const retention = api.body?.retention;
    if (!retention || typeof retention !== "object") {
      fail("/api/ops missing the retention section entirely");
    } else {
      if (typeof retention.active !== "boolean") {
        fail(`retention.active missing or not a boolean (got ${JSON.stringify(retention.active)})`);
      } else if (retention.active !== false) {
        fail(
          `retention.active = ${retention.active} on the probe serve (no -max-age: expected false)`,
        );
      }
      for (const key of [
        "max_age_seconds",
        "oldest_age_seconds",
        "over_age",
        "purge_lag_seconds",
      ]) {
        if (!(key in retention)) fail(`retention.${key} missing from /api/ops`);
      }
      for (const key of ["oldest_age_seconds", "purge_lag_seconds", "over_age"]) {
        const v = retention[key];
        if (v !== null && typeof v !== "number") {
          fail(`retention.${key} must be a number or null (got ${JSON.stringify(v)})`);
        }
      }
      console.log(
        "retention section:",
        `active=${retention.active}`,
        `max_age=${retention.max_age_seconds}`,
        `oldest=${retention.oldest_age_seconds}`,
        `over_age=${retention.over_age}`,
        `purge_lag=${retention.purge_lag_seconds}`,
      );
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
