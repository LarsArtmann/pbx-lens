#!/usr/bin/env node
// verify-ui-contract — the durable form of the one-off T20/T21 verification
// probe: one command re-proves the browser-level UI contracts against a
// freshly built binary (plan T4). Go tests pin the server-side rules;
// this script pins what only a real browser can prove (label swap on
// click, real fragment scroll) end to end over HTTP.
//
// Contracts pinned:
//   A. Copy affordances (T20): a detail page with a native id renders
//      exactly three `data-tc-copy` elements (Copy ID + From + To), all
//      BUTTON variant (the anchor variant must not leak in), and clicking
//      swaps the label to "Copied!" (the singleton CSP-safe script runs).
//   B. Search deep link (T21): a search hit href carries `?q=`; following
//      it 302s once to `?q=…&at=N#turn-N`, the page renders 200, the
//      anchored turn carries the amber mark, and the fragment actually
//      scrolled the line into the viewport.
//      (Server-side rules — same-origin redirect, tampered-at, no-match —
//      live in internal/web/search_jump_test.go.)
//   C. Match-log fixture (T4.3): the corpus is smoke + one agent
//      transcript ("thermostat"), seeded here so contract B always has a
//      turn-bearing stream to jump into; seeding is idempotent.
//
// Bootstrap (chromium, puppeteer, build, seed, serve) is shared with
// screenshot-probe.js in scripts/lib/ui-probe.js.
//
// Usage: node scripts/verify-ui-contract.js [--port 18107] [--out /tmp/lens-verify]
"use strict";

const path = require("path");
const probe = require("./lib/ui-probe");

const args = process.argv.slice(2);
const arg = (name, dflt) => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : dflt;
};

const OUT = arg("--out", "/tmp/lens-verify");
const PORT = arg("--port", "18107");
const BASE = `http://127.0.0.1:${PORT}`;
const DB = path.join(OUT, "lens.db");

function fail(msg) {
  console.error("FAIL:", msg);
  process.exitCode = 1;
}

(async () => {
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

    // --- Contract A: copy affordances on the detail page ---------------
    await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });
    const detailHref = await page.$$eval('[data-tc-row-href], a[href^="/calls/"]', (els) =>
      els.map((el) => el.getAttribute("data-tc-row-href") || el.getAttribute("href")).find(Boolean),
    );
    console.log("detail href:", detailHref);
    if (!detailHref) fail("no detail link on list page");

    await page.goto(`${BASE}${detailHref}`, { waitUntil: "networkidle0" });
    const copies = await page.$$eval("[data-tc-copy]", (els) =>
      els.map((el) => ({
        tag: el.tagName,
        text: el.getAttribute("data-tc-copy"),
        label: el.textContent.trim(),
      })),
    );
    console.log("copy elements:", JSON.stringify(copies));
    if (copies.length !== 3) fail(`expected 3 copy elements, got ${copies.length}`);
    if (copies.some((c) => c.tag !== "BUTTON")) fail("anchor copy variant leaked");

    const idValue = copies[0].text;
    const idBtn = await page.$(`[data-tc-copy="${idValue}"]`);
    await idBtn.click();
    await new Promise((r) => setTimeout(r, 300));
    const swapped = await page.$eval(`[data-tc-copy="${idValue}"]`, (el) => el.textContent.trim());
    console.log("label after click:", swapped);
    if (!swapped.toLowerCase().includes("copied")) fail("copy label did not swap on click");

    // --- Contract B: search deep link lands at the match ---------------
    await page.goto(`${BASE}/search?q=thermostat`, { waitUntil: "networkidle0" });
    const hitHref = await page
      .$eval('a[href*="/agent-transcript/"]', (el) => el.getAttribute("href"))
      .catch(() => null);
    if (!hitHref) fail("fixture transcript not found via /search?q=thermostat");
    console.log("hit href:", hitHref);
    if (!hitHref.includes("?q=")) fail("search hit missing ?q=");

    const resp2 = await page.goto(`${BASE}${hitHref}`, { waitUntil: "networkidle0" });
    console.log("detail status:", resp2.status(), "final url:", page.url());
    if (resp2.status() !== 200) fail("redirected detail did not render");
    if (!/#turn-\d+/.test(page.url())) fail(`expected fragment jump, url=${page.url()}`);

    const marks = await page.$$eval("[id^='turn-']", (els) =>
      els.map((el) => ({ id: el.id, marked: el.className.includes("bg-amber-100") })),
    );
    console.log("anchored turns:", JSON.stringify(marks));
    const firstAnchor = page.url().split("#turn-")[1];
    if (!marks.some((m) => m.id === `turn-${firstAnchor}` && m.marked)) {
      fail(`turn-${firstAnchor} not marked`);
    }

    const visible = await page.$eval(`#turn-${firstAnchor}`, (el) => {
      const r = el.getBoundingClientRect();
      return r.top >= 0 && r.top < window.innerHeight;
    });
    console.log("turn in viewport:", visible);
    if (!visible) fail("fragment did not scroll to the match");

    console.log(process.exitCode ? "RESULT: FAILED" : "RESULT: ALL PASS");
  } finally {
    await browser.close();
    probe.stopProbe(serve);
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
