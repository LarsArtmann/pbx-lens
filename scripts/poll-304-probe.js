#!/usr/bin/env node
// poll-304-probe — the 304-hit-rate harness behind the stamp-churn owner
// decision (TODO row; plan T18, 2026-10-26). Points at a RUNNING serve and
// replays the dashboard's 30 s htmx poll cadence against one or more ETagged
// routes, chaining If-None-Match the way the browser would. The printed
// 304-rate is the BEFORE number; after a client-side footer stamp lands,
// re-run and compare — that delta is the data the owner decides with.
//
// Usage: node scripts/poll-304-probe.js [--base http://127.0.0.1:8300]
//        [--polls 20] [--interval 2000] [--path /frag/dashboard]
//
// Exit 0 always (a measurement, not a gate); a non-200/304 answer fails
// loudly instead.
"use strict";

const http = require("http");

function arg(name, fallback) {
  const i = process.argv.indexOf(name);

  return i >= 0 && i + 1 < process.argv.length ? process.argv[i + 1] : fallback;
}

const BASE = arg("--base", "http://127.0.0.1:8300");
const POLLS = Number(arg("--polls", "20"));
const INTERVAL = Number(arg("--interval", "2000"));
const PATH = arg("--path", "/frag/dashboard");

function fetchOnce(etag) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      `${BASE}${PATH}`,
      { method: "GET", headers: etag ? { "If-None-Match": etag } : {} },
      (res) => {
        res.resume();
        resolve({ status: res.statusCode, etag: res.headers.etag || null });
      },
    );
    req.on("error", reject);
    req.setTimeout(5000, () => {
      req.destroy(new Error("timeout"));
    });
    req.end();
  });
}

(async () => {
  let etag = null;
  let hits304 = 0;
  const statuses = [];

  for (let i = 0; i < POLLS; i++) {
    const r = await fetchOnce(etag);
    if (r.status !== 200 && r.status !== 304) {
      console.error(`poll ${i}: unexpected status ${r.status}`);
      process.exit(1);
    }

    statuses.push(r.status);
    if (r.status === 304) hits304++;
    if (r.etag) etag = r.etag;
    if (i < POLLS - 1) await new Promise((res) => setTimeout(res, INTERVAL));
  }

  console.log(`route=${PATH} polls=${POLLS} interval=${INTERVAL}ms`);
  console.log(`statuses=${statuses.join(",")}`);
  console.log(`304-hit-rate=${((hits304 / POLLS) * 100).toFixed(1)}% (${hits304}/${POLLS})`);
  console.log(
    "note: the freshness pill is minute-quantized (docs/http-stack.md), so polls " +
      "crossing a minute boundary re-validate with 200s — compare rates across " +
      "the SAME minute span when reading before/after numbers.",
  );
})();
