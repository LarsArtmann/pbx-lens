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
const DB = path.join(OUT, "lens.db");

(async () => {
  // --port is optional (M4/F13): omit it and the OS hands over a free port.
  const PORT = args.includes("--port") ? arg("--port", null) : await probe.findFreePort();
  const BASE = `http://127.0.0.1:${PORT}`;
  console.log("port:", PORT);

  probe.ensureCodegen();
  const puppeteer = probe.ensurePuppeteer();
  const bin = probe.buildProbeBinary(OUT);
  // -with-recording turns the smoke-0001 recording card into the real
  // <audio> surface (plan M3/F9); -age 0 keeps every other page fresh.
  probe.seedProbeDB(bin, DB, { smokeArgs: ["-with-recording", "rec-probe-1"] });
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
      // Unseen states (plan M3/F8+F9): the 404 page and the audio card
      // (smoke-0001 seeded with -with-recording, so the <audio> element
      // renders instead of the empty state; the player never fetches until
      // played, so no network error noise lands in the capture).
      ["notfound", "/calls/no-such-stream/none"],
      ["detail-rec", "/calls/telnyx-conversation/smoke-0001"],
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

    // --- Print check (plan M5/F17): chrome drops, facts stay. The @media
    // print block in app.css hides nav/footer/buttons; this asserts the
    // computed result and leaves a PNG for the eyeball pass.
    {
      const ctx = await browser.createBrowserContext();
      const page = await ctx.newPage();
      await page.setViewport({ width: 1280, height: 900 });
      await page.emulateMediaType("print");
      await page.goto(`${BASE}${detail1}`, { waitUntil: "networkidle0" });
      await page.screenshot({
        path: path.join(OUT, "print-detail.png"),
        fullPage: true,
      });
      n++;
      const hidden = await page.evaluate(() =>
        ["nav", "footer", "button"].map((sel) => {
          const el = document.querySelector(sel);
          return el ? getComputedStyle(el).display === "none" : true;
        }),
      );
      console.log("print chrome hidden (nav/footer/button):", JSON.stringify(hidden));
      if (hidden.some((h) => !h)) fail("print media did not drop all chrome");
      await ctx.close();
    }

    // --- Empty-index phase (plan M3/F8): first-run corpus variant. The
    // smoke corpus never shows the "no calls yet" ladders, so restart the
    // serve against a truly empty DB and capture the three index pages.
    await probe.stopProbe(serve);
    const emptyDB = path.join(OUT, "empty.db");
    if (fs.existsSync(emptyDB)) fs.rmSync(emptyDB);
    const emptyServe = await probe.serveProbe(bin, emptyDB, PORT);
    try {
      const emptyPages = [
        ["empty-calls", "/"],
        ["empty-search", "/search"],
        ["empty-dashboard", "/dashboard"],
      ];
      for (const theme of themes) {
        const ctx = await browser.createBrowserContext();
        const page = await ctx.newPage();
        await page.emulateMediaFeatures([{ name: "prefers-color-scheme", value: theme }]);
        for (const vp of [viewports[1]]) {
          await page.setViewport({ width: vp.w, height: vp.h });
          for (const [name, urlPath] of emptyPages) {
            const resp = await page.goto(`${BASE}${urlPath}`, {
              waitUntil: "networkidle0",
              timeout: 20000,
            });
            const file = path.join(OUT, `${theme}-${vp.tag}-${name}.png`);
            await page.screenshot({ path: file, fullPage: true });
            n++;
            console.log("captured", file, `status=${resp.status()}`);
          }
        }
        await ctx.close();
      }
    } finally {
      probe.stopProbe(emptyServe);
    }

    await browser.close();
    console.log(`DONE: ${n} PNGs in ${OUT}`);
  } finally {
    await probe.stopProbe(serve);
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
