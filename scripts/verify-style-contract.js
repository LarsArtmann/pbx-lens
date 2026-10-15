#!/usr/bin/env node
// verify-style-contract — browser-level probe for the session's two fixes
// that goldens cannot see (plan M1/F1+F2). Goldens pin markup, not pixels;
// the violet heatmap leak lived invisible for weeks because of exactly that.
//
// Contracts pinned:
//   S1. ds-brand bridge: --ds-brand-rgb resolves to teal-700 "15, 118, 110"
//       in light and teal-400 "45, 212, 191" under .dark, and NEVER to the
//       library's violet defaults (124, 58, 237 / 139, 92, 246).
//   S2. Heatmap pixels: a rendered density cell's computed background-color
//       resolves through the bridge to rgb(15, 118, 110) (alpha may vary).
//   S3. Truthful filter selects (All-option): on the unfiltered list both
//       selects' first option is value="" labelled "All" and IS the selected
//       one; with ?direction=inbound the Inbound option is the selected one.
//
// The CSS gate twin (TestCSSGateBridgesHeatmapBrandToTeal) pins the built
// stylesheet; this script pins what the BROWSER computes. Server-side filter
// parsing is pinned in Go tests; this pins the DOM a real browser sees.
//
// Bootstrap (chromium, puppeteer, build, seed, serve) is shared via
// scripts/lib/ui-probe.js.
//
// Usage: node scripts/verify-style-contract.js [--port 18110] [--out /tmp/lens-style-verify]
"use strict";

const path = require("path");
const probe = require("./lib/ui-probe");

const args = process.argv.slice(2);
const arg = (name, dflt) => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : dflt;
};

const OUT = arg("--out", "/tmp/lens-style-verify");
const DB = path.join(OUT, "lens.db");

function fail(msg) {
  console.error("FAIL:", msg);
  process.exitCode = 1;
}

const VIOLET = ["124, 58, 237", "139, 92, 246"];

(async () => {
  // --port is optional (M4/F13): omit it and the OS hands over a free port.
  const PORT = args.includes("--port") ? arg("--port", null) : await probe.findFreePort();
  const BASE = `http://127.0.0.1:${PORT}`;
  console.log("port:", PORT);

  probe.ensureCodegen();
  const puppeteer = probe.ensurePuppeteer();
  const bin = probe.buildProbeBinary(OUT);
  probe.seedProbeDB(bin, DB);
  const serve = await probe.serveProbe(bin, DB, PORT);
  const browser = await puppeteer.launch({
    executablePath: probe.findChromium(),
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 900 });

    // --- S1 + S2: the ds-brand bridge, light then dark ------------------
    await page.goto(`${BASE}/dashboard`, { waitUntil: "networkidle0" });

    async function brandVar() {
      return page.evaluate(() =>
        getComputedStyle(document.documentElement).getPropertyValue("--ds-brand-rgb").trim(),
      );
    }

    const lightVar = await brandVar();
    console.log("light --ds-brand-rgb:", lightVar);
    if (lightVar !== "15, 118, 110") fail(`light bridge resolves to ${lightVar}, want teal-700 triplet`);
    for (const v of VIOLET) {
      if (lightVar === v) fail(`light bridge is violet (${v})`);
    }

    await page.evaluate(() => document.documentElement.classList.add("dark"));
    const darkVar = await brandVar();
    console.log("dark --ds-brand-rgb:", darkVar);
    if (darkVar !== "45, 212, 191") fail(`dark bridge resolves to ${darkVar}, want teal-400 triplet`);
    for (const v of VIOLET) {
      if (darkVar === v) fail(`dark bridge is violet (${v})`);
    }
    await page.evaluate(() => document.documentElement.classList.remove("dark"));

    // S2: computed pixel truth of a real cell (smoke corpus is seeded with
    // current dates, so the last-30-day grid has cells).
    const cell = await page.evaluate(() => {
      const el = document.querySelector('[style*="--ds-brand-rgb"]');
      if (!el) return null;
      return getComputedStyle(el).backgroundColor;
    });
    if (!cell) {
      fail("no heatmap cell rendered (smoke corpus did not land in the last-30-day window?)");
    } else {
      console.log("heatmap cell background:", cell);
      if (!/^rgba?\(\s*15,\s*118,\s*110\b/.test(cell)) {
        fail(`heatmap cell resolves to ${cell}, want the teal-700 family`);
      }
    }

    // --- S3: truthful All-selects ----------------------------------------
    await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });
    const unfiltered = await page.evaluate(() =>
      ["direction", "ai"].map((name) => {
        const sel = document.querySelector(`select[name="${name}"]`);
        if (!sel) return null;
        const first = sel.options[0];
        return { name, value: first.value, label: first.textContent.trim(), selected: first.selected };
      }),
    );
    console.log("unfiltered selects:", JSON.stringify(unfiltered));
    for (const s of unfiltered) {
      if (!s) return fail("filter select missing from list page");
      if (s.value !== "") fail(`select ${s.name}: first option value=${s.value}, want ""`);
      if (s.label !== "All") fail(`select ${s.name}: first option label=${s.label}, want "All"`);
      if (!s.selected) fail(`select ${s.name}: the All option is not selected on the unfiltered view`);
    }

    await page.goto(`${BASE}/?direction=inbound`, { waitUntil: "networkidle0" });
    const filtered = await page.evaluate(() => {
      const sel = document.querySelector('select[name="direction"]');
      return { value: sel.value, selectedText: sel.options[sel.selectedIndex].textContent.trim() };
    });
    console.log("direction=inbound select:", JSON.stringify(filtered));
    if (filtered.value !== "inbound") fail(`?direction=inbound renders value=${filtered.value}`);
    if (!/^inbound/i.test(filtered.selectedText)) {
      fail(`?direction=inbound selected option reads "${filtered.selectedText}", want Inbound`);
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
