#!/usr/bin/env node
// screenshot-probe — pixel-truth harness for the pbx-lens UI (plan T3).
//
// Captures every operator-facing page at light/dark × 375/1280 px into
// --out as labeled PNGs for the eyeball passes and as the visual
// instrument later tasks read from. Pages: calls, search, dashboard, two
// discovered details, and the ?q= match-log detail (T1.3 — URL scraped
// from a real search hit; the agent-transcript fixture makes it show the
// amber marks).
//
// Bootstrap (chromium discovery, puppeteer scratch install, binary build,
// corpus seeding, serve lifecycle) lives in scripts/lib/ui-probe.js.
//
// Usage: node scripts/screenshot-probe.js [--port 18099] [--out /tmp/lens-shots]
"use strict";

const fs = require("fs");
const path = require("path");
const probe = require("./lib/ui-probe");

const args = process.argv.slice(2);
const arg = (name, dflt) => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : dflt;
};

const OUT = arg("--out", "/tmp/lens-shots");
const PORT = arg("--port", "18099");
const BASE = `http://127.0.0.1:${PORT}`;
const DB = path.join(OUT, "lens.db");

(async () => {
  const puppeteer = probe.ensurePuppeteer();
  const bin = probe.buildProbeBinary(OUT);
  probe.seedProbeDB(bin, DB);
  const serve = await probe.serveProbe(bin, DB, PORT);
  try {
    const browser = await puppeteer.launch({
      executablePath: probe.findChromium(),
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

    // Discover the ?q= match-log URL (plan T1.3): scrape the first search
    // hit for the fixture term; that href + ?q= is the detail page whose
    // turns carry the amber mark (the server 302s once to at=0#turn-0 and
    // puppeteer follows it, so the capture shows the post-jump state).
    const hitPage = await browser.newPage();
    await hitPage.goto(`${BASE}/search?q=thermostat`, { waitUntil: "networkidle0" });
    const hitHrefs = await hitPage.$$eval('[data-tc-row-href], a[href^="/calls/"]', (els) => [
      ...new Set(els.map((el) => el.getAttribute("data-tc-row-href") || el.getAttribute("href"))),
    ]);
    await hitPage.close();
    const detailQ = hitHrefs[0]
      ? `${hitHrefs[0]}${hitHrefs[0].includes("?") ? "&" : "?"}q=thermostat`
      : detail1;

    const pages = [
      ["calls", "/"],
      ["search", "/search"],
      ["dashboard", "/dashboard"],
      ["detail-a", detail1],
      ["detail-b", detail2],
      ["detail-q", detailQ],
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
    probe.stopProbe(serve);
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
