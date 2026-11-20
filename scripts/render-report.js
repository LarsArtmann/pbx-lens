#!/usr/bin/env node
// render-report.js — render one report HTML to a full-page PNG.
//
// Usage: node scripts/render-report.js <report.html> [out.png]
//   out.png defaults to <report>.png beside the input.
//
// Browser decision (mirrors scripts/lib/ui-probe.js's record): the
// cached puppeteer chrome-headless-shell exits 127 on this NixOS host —
// it links libnss3, which a plain store path does not carry — so the
// renderer drives NixOS's chromium through puppeteer-core (the same
// discovery + scratch-install the screenshot probe uses). chromium
// honors prefers-color-scheme from its flags, so reports render in the
// scheme their CSS defaults to; pass CHROMIUM=/path to override the
// discovery.

const fs = require("fs");
const path = require("path");
const probe = require("./lib/ui-probe");

const positional = process.argv.slice(2);
const input = positional[0];
if (!input) {
  console.error("usage: node scripts/render-report.js <report.html> [out.png]");
  process.exit(2);
}

const source = path.resolve(input);
if (!fs.existsSync(source)) {
  console.error(`render-report: no such file: ${source}`);
  process.exit(2);
}

const out = path.resolve(positional[1] || source.replace(/\.html$/i, "") + ".png");

(async () => {
  const chromium = probe.findChromium();
  const puppeteer = probe.ensurePuppeteer();

  const browser = await puppeteer.launch({
    executablePath: chromium,
    args: ["--no-sandbox", "--disable-gpu", "--hide-scrollbars"],
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 900 });
    await page.goto("file://" + source, { waitUntil: "networkidle0" });
    await page.screenshot({ path: out, fullPage: true });
    console.log(`render-report: ${out}`);
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(`render-report: ${err.message}`);
  process.exit(1);
});
