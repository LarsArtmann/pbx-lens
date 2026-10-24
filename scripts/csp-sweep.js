#!/usr/bin/env node
// csp-sweep — the P1-T7 report-only dry run as a durable instrument: boots
// the serve chain with CSP_NONCE=1 + CSP_REPORT_ONLY=1, drives every UI
// surface (plus the interactive handlers: theme toggle, copy buttons), and
// collects Content-Security-Policy violations the way only a real browser
// reports them — via `securitypolicyviolation` events and the console.
//
// Why both env flags: CSP_REPORT_ONLY stamps the nonce-based policy, and
// the nonce only exists under CSP_NONCE=1 — report-only alone would report
// every inline script against an empty 'nonce-' and bury the real signal.
//
// Verdict protocol (TODO row "CSP report-only dry run"): exit 0 with zero
// violations = the enforced rollout is safe to flip; any violation lands
// here with directive + sample for the enforce/adjust call.
//
// Usage: node scripts/csp-sweep.js [--port 18108] [--out /tmp/lens-csp]
"use strict";

const probe = require("./lib/ui-probe");

(async () => {
  const {
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    base: BASE,
  } = await probe.probeSession("/tmp/lens-csp");
  console.log("port:", PORT);

  const { serve, browser } = await probe.launchProbe({
    outDir: OUT,
    dbPath: DB,
    port: PORT,
    serveEnv: { CSP_NONCE: "1", CSP_REPORT_ONLY: "1" },
  });

  const violations = [];
  let surfaces = 0;

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 900 });

    await page.evaluateOnNewDocument(() => {
      window.__cspViolations = [];
      document.addEventListener("securitypolicyviolation", (e) => {
        window.__cspViolations.push({
          directive: e.effectiveDirective,
          blocked: e.blockedURI,
          sample: e.sample,
          source: e.sourceFile,
        });
      });
    });

    let consoleSurface = "(before first navigation)";

    page.on("console", (msg) => {
      const text = msg.text();
      if (/\[Report Only\]|Refused to |Content Security Policy/i.test(text)) {
        violations.push({ surface: `${consoleSurface} (console)`, text: text.slice(0, 300) });
      }
    });

    async function sweep(surface, { clickToggle = false } = {}) {
      surfaces++;
      consoleSurface = surface;
      const resp = await page.goto(BASE + surface, { waitUntil: "networkidle2" });
      const reportOnly = resp.headers()["content-security-policy-report-only"];
      console.log(
        `${surface} -> ${resp.status()} report-only=${reportOnly ? "present" : "MISSING"}`,
      );
      if (!reportOnly) {
        violations.push({ surface, text: "Content-Security-Policy-Report-Only header MISSING" });
      }
      if (clickToggle) {
        const toggle = await page.$('[role="switch"], [aria-checked]');
        if (toggle) await toggle.click();
      }
      const found = await page.evaluate(() => window.__cspViolations || []);
      for (const v of found) {
        violations.push({
          surface,
          text: `${v.directive} blocked=${v.blocked} sample=${JSON.stringify(v.sample)}`,
        });
      }
      await page.evaluate(() => {
        window.__cspViolations = [];
      });
    }

    await sweep("/", { clickToggle: true });
    await sweep("/dashboard");
    await sweep("/ops");
    await sweep("/search");
    await sweep("/search?q=thermostat");

    await page.goto(BASE + "/", { waitUntil: "networkidle2" });
    const detailHref = await page.evaluate(() => {
      const a = document.querySelector('a[href^="/calls/"]');
      if (a) return a.getAttribute("href");
      const row = document.querySelector("tr[data-tc-row-href]");
      return row ? row.dataset.tcRowHref : null;
    });
    if (detailHref) {
      await sweep(detailHref, { clickToggle: true });
      consoleSurface = `${detailHref} (copy click)`;
      const copy = await page.$("[data-tc-copy]");
      if (copy) await copy.click();
      const found = await page.evaluate(() => window.__cspViolations || []);
      for (const v of found) {
        violations.push({
          surface: detailHref + " (copy click)",
          text: `${v.directive} blocked=${v.blocked} sample=${JSON.stringify(v.sample)}`,
        });
      }
    } else {
      console.log("detail page: no /calls/ link found in the seeded corpus — skipped");
    }
  } finally {
    await browser.close();
    await probe.stopProbe(serve);
  }

  if (violations.length > 0) {
    console.error(`\nFAIL: ${violations.length} CSP report(s) across ${surfaces} surfaces:`);
    for (const v of violations) console.error(`  [${v.surface}] ${v.text}`);
    process.exitCode = 1;
    return;
  }

  console.log(
    `\nPASS: zero CSP violations across ${surfaces} surfaces (nonce policy is enforce-clean).`,
  );
  console.log(
    "Enforce/adjust call: flip CSP_REPORT_ONLY to enforcing (CSP_NONCE=1 stays opt-in) per docs/http-stack.md.",
  );
})();
