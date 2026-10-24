#!/usr/bin/env node
// verify-staleness-live — the browser-level live watch for the staleness
// banner (plan M3/F11). Go tests pin the server-side stale computation; this
// pins the LIVE transition: the polled region actually recovers on screen
// when fresh data lands, without a reload.
//
// Contract S (staleness live):
//   1. A serve whose newest call is 48 h old renders "Index may be stale"
//      on / (seeded via smoke -age 48h).
//   2. Forcing the htmx poll (htmx.ajax on the PolledRegion, the same GET
//      the 60 s timer would fire) keeps the banner while the index is stale.
//   3. Ingesting a fresh turn (new stream, current timestamp) and forcing
//      the poll again makes the banner DISAPPEAR on screen — the polled
//      fragment swapped, not the whole page.
//
// Usage: node scripts/verify-staleness-live.js [--port 18116] [--out /tmp/lens-stale]
"use strict";

const probe = require("./lib/ui-probe");

const fail = probe.fail;

async function forceStalenessPoll(page) {
  return page.evaluate(async () => {
    const region = document.querySelector('[hx-get*="/frag/staleness"]');
    if (!region) return false;
    await htmx.ajax("GET", region.getAttribute("hx-get"), {
      target: region,
      swap: "outerHTML",
    });
    return true;
  });
}

(async () => {
  const {
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    base: BASE,
  } = await probe.probeSession("/tmp/lens-stale");
  console.log("port:", PORT);

  // Fresh DB every run: seedProbeDB keeps existing DBs (stream-identity
  // no-op), but THIS probe's whole premise is "newest call is 48h old" —
  // a reused DB carries seedFreshAgentTurn's now-ts turn from the last
  // run, the 48h seeding silently no-ops, and the banner never renders
  // (both phases then fail, as observed 2026-10-15). Reset like the
  // screenshot probe's empty-index phase does.
  const { bin, serve, browser, page } = await probe.launchProbe({
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    seedOpts: { smokeArgs: ["-age", "48h"] },
    resetDb: true,
  });

  try {
    await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });

    const staleBefore = await page.evaluate(() =>
      document.body.textContent.includes("Index may be stale"),
    );
    console.log("banner while stale:", staleBefore);
    if (!staleBefore) fail("48h-old index did not render the staleness banner");

    const polled1 = await forceStalenessPoll(page);
    const staleAfterPoll = await page.evaluate(() =>
      document.body.textContent.includes("Index may be stale"),
    );
    console.log("polled while stale:", polled1, "banner still up:", staleAfterPoll);
    if (!polled1) fail("staleness PolledRegion not found on /");
    if (!staleAfterPoll) fail("banner vanished while the index was still stale");

    probe.seedFreshAgentTurn(bin, DB, OUT);
    const polled2 = await forceStalenessPoll(page);
    const staleAfterFresh = await page.evaluate(() =>
      document.body.textContent.includes("Index may be stale"),
    );
    console.log("polled after fresh ingest:", polled2, "banner gone:", !staleAfterFresh);
    if (!polled2) fail("second poll did not run");
    if (staleAfterFresh) fail("banner did not disappear after fresh data + poll (no reload)");

    console.log(process.exitCode ? "RESULT: FAILED" : "RESULT: ALL PASS");
  } finally {
    await browser.close();
    await probe.stopProbe(serve);
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
