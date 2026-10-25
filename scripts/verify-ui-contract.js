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

const probe = require("./lib/ui-probe");

const fail = probe.fail;

(async () => {
  const {
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    base: BASE,
  } = await probe.probeSession("/tmp/lens-verify");
  console.log("port:", PORT);

  const { serve, browser, page } = await probe.launchProbe({ outDir: OUT, dbPath: DB, port: PORT });

  try {
    // --- Contract D: keyboard walk (plan M5/F16) --------------------------
    // A keyboard user must reach the clickable table rows by tabbing, and
    // the row must show the app.css focus ring (inset teal outline) when
    // focused via keyboard — the library markup has NO visible focus style
    // of its own (upstream filing candidate #49). The copy affordances live
    // on the DETAIL page, so the copy reachability leg walks there.
    async function tabState(page) {
      await page.keyboard.press("Tab");
      return page.evaluate(() => {
        const el = document.activeElement;
        if (!el) return { isRow: false, isCopy: false };
        const cs = getComputedStyle(el);
        return {
          isRow: el.matches('tr[role="link"]'),
          isCopy: el.matches("[data-tc-copy]"),
          outline: `${cs.outlineStyle} ${cs.outlineColor}`,
        };
      });
    }

    {
      await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });
      let rowOutline = null;
      for (let i = 0; i < 120 && rowOutline === null; i++) {
        const st = await tabState(page);
        if (st.isRow) rowOutline = st.outline;
      }
      if (rowOutline === null) fail("tabbing never reached a clickable table row");
      else if (!/rgb\(15,\s*118,\s*110\)/.test(rowOutline)) {
        fail(`table-row focus ring is not the teal outline (got ${rowOutline})`);
      }
      console.log("row focus outline:", rowOutline);
    }

    // --- Contract A: copy affordances on the detail page ---------------
    await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });
    const detailHref = await page.$$eval('[data-tc-row-href], a[href^="/calls/"]', (els) =>
      els.map((el) => el.getAttribute("data-tc-row-href") || el.getAttribute("href")).find(Boolean),
    );
    console.log("detail href:", detailHref);
    if (!detailHref) fail("no detail link on list page");

    // Contract D, copy leg (M5/F16): the copy affordances are reachable by
    // keyboard from the top of the detail page.
    {
      await page.goto(`${BASE}${detailHref}`, { waitUntil: "networkidle0" });
      let reachedCopy = false;
      for (let i = 0; i < 120 && !reachedCopy; i++) {
        reachedCopy = (await tabState(page)).isCopy;
      }
      if (!reachedCopy) fail("tabbing never reached a copy affordance on the detail page");
      console.log("keyboard walk: row ring + copy reachable OK");
    }

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

    // --- Contract B2: transcript mode lands at the exact turn ---------
    // ?mode=transcript renders per-turn hits; each deep-links with the
    // at param (suppressing the jump redirect) plus its own fragment, so
    // a turn-2 hit lands at turn 2, not the call's first match.
    await page.goto(`${BASE}/search?q=thermostat&mode=transcript`, { waitUntil: "networkidle0" });
    const turnHitHref = await page
      .$eval('a[href*="#turn-"]', (el) => el.getAttribute("href"))
      .catch(() => null);
    if (!turnHitHref) fail("transcript mode rendered no per-turn deep link");
    console.log("turn hit href:", turnHitHref);
    if (!turnHitHref.includes("&at=")) fail("turn hit missing the at anchor param");

    const modeTabs = await page.$$eval("a[href^='/search?q=']", (els) =>
      els.map((el) => ({
        text: el.textContent.trim(),
        active: el.className.includes("border-tc-primary"),
      })),
    );
    console.log("mode tabs:", JSON.stringify(modeTabs));
    const turnsTab = modeTabs.find((t) => t.text === "Turns");
    if (!turnsTab || !turnsTab.active) fail("Turns tab not active in transcript mode");

    const resp3 = await page.goto(`${BASE}${turnHitHref}`, { waitUntil: "networkidle0" });
    // 304 is correct here: Contract B already loaded this exact detail URL,
    // and the ETag revalidation contract (no-cache) answers Not Modified.
    if (![200, 304].includes(resp3.status())) fail(`turn deep link status ${resp3.status()}`);
    if (!page.url().includes("#turn-")) fail(`turn deep link lost its fragment, url=${page.url()}`);

    // --- Contract E: a11y basics on /ops (2026-10-17 hardening) --------
    // Heading order starts at exactly one h1 and never skips a level, and
    // keyboard focus carries a visible indicator on the first nav stop.
    // The verdict-chip contrast rides the library tc-* tokens (the style
    // contract pins the token chain); the probe stack serves /ops unwired,
    // so the chip itself is covered by the ops goldens + page tests.
    await page.goto(`${BASE}/ops`, { waitUntil: "networkidle0" });
    const a11y = await page.evaluate(() => {
      const levels = [...document.querySelectorAll("h1, h2, h3, h4, h5, h6")].map((h) =>
        Number(h.tagName[1]),
      );
      let skipped = null;
      let prev = 0;
      for (const level of levels) {
        if (prev > 0 && level > prev + 1) {
          skipped = `h${prev}->h${level}`;
          break;
        }
        prev = level;
      }
      return { h1Count: levels.filter((l) => l === 1).length, skipped, levels: levels.join(",") };
    });
    console.log("ops headings:", a11y.levels);
    if (a11y.h1Count !== 1) fail(`/ops must have exactly one h1, got ${a11y.h1Count}`);
    if (a11y.skipped) fail(`/ops heading order skips a level (${a11y.skipped})`);

    await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });
    await page.keyboard.press("Tab");
    const focusVisible = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el) return false;
      const cs = getComputedStyle(el);
      return cs.outlineStyle !== "none" || cs.boxShadow !== "none";
    });
    if (!focusVisible) fail("first Tab stop has no visible focus indicator");

    console.log(process.exitCode ? "RESULT: FAILED" : "RESULT: ALL PASS");
  } finally {
    await browser.close();
    await probe.stopProbe(serve);
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
