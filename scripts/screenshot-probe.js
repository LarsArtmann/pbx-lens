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
//        [--themes light,dark] [--only calls,dashboard,print]
"use strict";

const fs = require("fs");
const path = require("path");
const probe = require("./lib/ui-probe");

const args = process.argv.slice(2);
const arg = (name, dflt) => probe.arg(args, name, dflt);

const OUT = arg("--out", "/tmp/lens-shots");
const DB = path.join(OUT, "lens.db");

// fail records a contract violation (the checks below have a right answer,
// unlike the alert-only pixel diff). Collected rather than thrown so one bad
// surface does not cost the rest of the corpus; the run still ends red.
const failures = [];
function fail(message) {
  failures.push(message);
  console.error(`FAIL: ${message}`);
}

// --only <name[,name…]> (plan 05-40 M12/F33): capture a subset instead of
// the full sweep — an eyeball pass on one page shouldn't pay for the whole
// corpus. Names are the capture labels (calls, dashboard, detail-a,
// empty-calls, print, …).
const KNOWN_SURFACES = [
  "calls",
  "search",
  "dashboard",
  "detail-a",
  "detail-b",
  "detail-q",
  "notfound",
  "detail-rec",
  "print",
  "empty-calls",
  "empty-search",
  "empty-dashboard",
  "long-excerpt",
];
const ONLY = args.includes("--only")
  ? arg("--only", "")
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean)
  : null;
if (ONLY) {
  const unknown = ONLY.filter((n) => !KNOWN_SURFACES.includes(n));
  if (unknown.length > 0) {
    console.error(
      `--only: unknown surface(s) ${unknown.join(", ")}; known: ${KNOWN_SURFACES.join(", ")}`,
    );
    process.exit(1);
  }
}
const wanted = (name) => !ONLY || ONLY.includes(name);
// Surfaces that consume the DISCOVERED detail hrefs (detail-a/b/q and the
// print check ride detail1). When none are selected, the discovery page
// loads are skipped entirely (the --only dashboard eyeball pass no longer
// pays for two scrapes it never uses).
const NEEDS_DISCOVERY =
  !ONLY ||
  ONLY.some((n) => n === "detail-a" || n === "detail-b" || n === "detail-q" || n === "print");

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
    // anchors — scrape the attribute (a[href] kept as fallback). Skipped
    // when no discovered-detail surface is selected (see NEEDS_DISCOVERY).
    let detail1 = "/calls/telnyx-conversation/none";
    let detail2 = detail1;
    let detailQ = detail1;

    if (NEEDS_DISCOVERY) {
      const seedPage = await browser.newPage();
      await seedPage.goto(`${BASE}/`, { waitUntil: "networkidle0" });
      const hrefs = await seedPage.$$eval('[data-tc-row-href], a[href^="/calls/"]', (els) => [
        ...new Set(els.map((el) => el.getAttribute("data-tc-row-href") || el.getAttribute("href"))),
      ]);
      await seedPage.close();
      detail1 = hrefs[0] || detail1;
      detail2 = hrefs[1] || detail1;

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
      detailQ = hitHrefs[0]
        ? `${hitHrefs[0]}${hitHrefs[0].includes("?") ? "&" : "?"}q=thermostat`
        : detail1;
    }

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
    // Drift guard (2026-10-17): a page added to the capture loop without a
    // KNOWN_SURFACES label (or vice versa) is a silent --only hole — fail
    // loudly instead.
    const drift = pages.map(([name]) => name).filter((n) => !KNOWN_SURFACES.includes(n));
    if (drift.length > 0) {
      console.error(`surface drift: ${drift.join(", ")} missing from KNOWN_SURFACES`);
      process.exit(1);
    }
    // --themes dark|light|light,dark (plan M21/F64): the exhaustive dark
    // sweep after a color change is one command without re-capturing light.
    const themes = (arg("--themes", "light,dark") || "light,dark")
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t === "light" || t === "dark");
    if (themes.length === 0) {
      console.error("no valid themes after --themes (use light and/or dark)");
      process.exit(1);
    }
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
        for (const [name, urlPath] of pages.filter(([n]) => wanted(n))) {
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
    if (wanted("print")) {
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
      // The ops cards carry break-inside:avoid under print (2026-10-17
      // print-breaks nit); assert the computed result on every card, not
      // just the CSS text, and capture the page's print posture as evidence.
      await page.goto(`${BASE}/ops`, { waitUntil: "networkidle0" });
      const breakInside = await page.evaluate(() => {
        const cards = document.querySelectorAll(".lens-ops-card");
        const values = [...cards].map((el) => getComputedStyle(el).breakInside);
        return { count: cards.length, values };
      });
      console.log(
        `print break-inside on .lens-ops-card (${breakInside.count} cards):`,
        JSON.stringify(breakInside.values),
      );
      if (breakInside.count === 0) fail("print ops posture: no .lens-ops-card found on /ops");
      if (breakInside.values.some((v) => v !== "avoid")) {
        fail("print media did not keep break-inside:avoid on every .lens-ops-card");
      }
      await page.screenshot({ path: path.join(OUT, "print-ops.png"), fullPage: true });
      n++;
      await ctx.close();
    }

    // --- Empty-index phase (plan M3/F8): first-run corpus variant. The
    // smoke corpus never shows the "no calls yet" ladders, so restart the
    // serve against a truly empty DB and capture the three index pages.
    if (["empty-calls", "empty-search", "empty-dashboard"].some(wanted)) {
      await probe.stopProbe(serve);
      const emptyDB = path.join(OUT, "empty.db");
      if (fs.existsSync(emptyDB)) fs.rmSync(emptyDB);
      // Own free port: the drained serve holds PORT through its 5s
      // graceful shutdown, so a same-port phase server either fails to
      // bind or its ping is answered by the DYING server (2026-10-15:
      // long-excerpt captures rendered the previous phase's DB exactly
      // this way). A phase-local port removes the race entirely.
      const emptyPort = await probe.findFreePort();
      const emptyServe = await probe.serveProbe(bin, emptyDB, emptyPort);
      const emptyBase = `http://127.0.0.1:${emptyPort}`;
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
            for (const [name, urlPath] of emptyPages.filter(([n]) => wanted(n))) {
              const resp = await page.goto(`${emptyBase}${urlPath}`, {
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
    }

    // --- Long-excerpt phase (plan 05-40 M13/F42, folded from the one-off
    // /tmp capture): a transcript turn at/past the 200-rune excerpt
    // ceiling renders in the calls table — the truncation eyeball gets a
    // permanent corpus page instead of a lost one-off script. Own DB: the
    // main corpus must stay deterministic for the other captures.
    if (wanted("long-excerpt")) {
      // The empty phase normally stops the main serve; when --only skipped
      // it, the main serve still holds the port (stopProbe is idempotent).
      await probe.stopProbe(serve);
      const longDB = path.join(OUT, "long-excerpt.db");
      for (const suffix of ["", "-shm", "-wal"]) {
        if (fs.existsSync(longDB + suffix)) fs.rmSync(longDB + suffix);
      }
      const longDir = path.join(OUT, "long-fixture");
      fs.mkdirSync(longDir, { recursive: true });
      fs.writeFileSync(
        path.join(longDir, "long-1.jsonl"),
        [
          JSON.stringify({
            type: "start",
            ts: Math.floor(Date.now() / 1000),
            uuid: "probe-long-1",
            models: ["gpt-rt"],
          }),
          JSON.stringify({
            type: "turn",
            role: "caller",
            text: "The customer reported that the heating system in the upstairs unit has been making a loud banging noise every morning around six, and additionally the hot water pressure drops noticeably whenever the kitchen tap is opened at the same time, which suggests a shared pipe issue that needs a plumber's assessment before winter arrives in earnest. We scheduled the earliest available appointment.",
          }),
          JSON.stringify({ type: "end", reason: "normal" }),
        ].join("\n") + "\n",
      );
      const { execFileSync } = require("child_process");
      execFileSync(bin, ["ingest-agent", "-db", longDB, "-dir", longDir], { stdio: "ignore" });
      const longPort = await probe.findFreePort();
      const longServe = await probe.serveProbe(bin, longDB, longPort);
      const longBase = `http://127.0.0.1:${longPort}`;
      try {
        for (const theme of themes) {
          const ctx = await browser.createBrowserContext();
          const page = await ctx.newPage();
          await page.emulateMediaFeatures([{ name: "prefers-color-scheme", value: theme }]);
          await page.setViewport({ width: viewports[1].w, height: viewports[1].h });
          await page.goto(`${longBase}/`, { waitUntil: "networkidle0", timeout: 20000 });
          const file = path.join(OUT, `long-excerpt-${theme}.png`);
          await page.screenshot({ path: file, fullPage: true });
          n++;
          console.log("captured", file);
          await ctx.close();
        }
      } finally {
        probe.stopProbe(longServe);
      }
    }

    await browser.close();

    // --- Alert-only pixel diff (plan 05-40 M19/F59-F60): compare the
    // fresh captures against a baseline directory. The report NEVER fails
    // the run — it is an alert for eyeballs, not a gate.
    //
    // Volatile-pixel budget (practiced 2026-10-17 against a 41-PNG
    // baseline, zero-tolerance measured): the only run-to-run churn is
    // time text — the footer "Updated <time>" stamp (dashboard, 36-63 px)
    // and a RelativeTime chip on detail pages (80-81 px) — ≤81 px in every
    // measured pair, at content-dependent positions, so a geometric mask
    // would be brittle. Instead any capture at or under --volatile-px
    // (default 128) is reported as absorbed; the % tolerance for what
    // remains tightens to 0.05%. The baseline STRATEGY itself (in-repo vs
    // machine-local) stays the ROADMAP owner question; the practiced
    // corpus lives at ~/lens-baselines/<gen>/ outside the repo.
    if (args.includes("--diff") || args.includes("--save-baseline")) {
      const captured = fs
        .readdirSync(OUT)
        .filter((f) => f.endsWith(".png"))
        .sort();
      const saveDir = arg("--save-baseline", null);

      if (saveDir) {
        fs.mkdirSync(saveDir, { recursive: true });
        for (const f of captured) fs.copyFileSync(path.join(OUT, f), path.join(saveDir, f));
        console.log(`baseline saved: ${captured.length} PNGs -> ${saveDir}`);
      }

      const baselineDir = arg("--diff", null);
      if (baselineDir) {
        if (!fs.existsSync(baselineDir)) {
          console.error(
            `pixel-diff: baseline dir missing: ${baselineDir} (run --save-baseline first)`,
          );
          process.exit(1);
        }

        const tolerancePct = parseFloat(arg("--tolerance", "0.05"));
        const volatilePx = parseInt(arg("--volatile-px", "128"), 10);
        const mods = probe.ensureScratchDeps(["pixelmatch@5", "pngjs@7"]);
        const pixelmatch = mods.pixelmatch;
        const { PNG } = mods.pngjs;
        let flagged = 0;

        for (const f of captured) {
          const base = path.join(baselineDir, f);
          if (!fs.existsSync(base)) {
            console.log(`PIXELDIFF NEW  ${f} (no baseline capture)`);
            continue;
          }

          const a = PNG.sync.read(fs.readFileSync(base));
          const b = PNG.sync.read(fs.readFileSync(path.join(OUT, f)));
          if (a.width !== b.width || a.height !== b.height) {
            console.log(`PIXELDIFF SIZE ${f}: ${a.width}x${a.height} -> ${b.width}x${b.height}`);
            flagged++;
            continue;
          }

          const diff = new PNG({ width: a.width, height: a.height });
          const changed = pixelmatch(a.data, b.data, diff.data, a.width, a.height, {
            threshold: 0.1,
          });
          const pct = (100 * changed) / (a.width * a.height);
          if (changed <= volatilePx) {
            console.log(
              `PIXELDIFF VOLATILE ${f}: ${changed} px (≤ ${volatilePx} time-text budget) absorbed`,
            );
          } else if (pct > tolerancePct) {
            console.log(
              `PIXELDIFF ALERT ${f}: ${changed} px (${pct.toFixed(3)}% > ${tolerancePct}%)`,
            );
            flagged++;
          }
        }

        console.log(
          `pixel-diff: ${captured.length} compared against ${baselineDir}, ${flagged} alert(s) (alert-only, never a gate)`,
        );
      }
    }

    console.log(`DONE: ${n} PNGs in ${OUT}`);
  } finally {
    await probe.stopProbe(serve);
  }

  if (failures.length > 0) {
    console.error(`${failures.length} contract failure(s): ${failures.join("; ")}`);
    process.exitCode = 1;
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
