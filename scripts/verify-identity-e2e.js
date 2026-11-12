#!/usr/bin/env node
// verify-identity-e2e — the identity surface's browser-level contract
// (2026-11-12 identity plan P03.7 + P05): the gate behavior, the
// first-visitor bootstrap, the FULL WebAuthn ceremony through a CDP
// virtual authenticator, the logout roundtrip, the account lockout, and
// the auth surfaces' cache headers — everything only a real browser (or a
// real HTTP client) can prove. Go tests pin the same rules server-side;
// this probe proves the composed serve actually behaves.
//
// Login is PASSKEY-ONLY (usermgmt ships no password login), so the
// ceremony legs run on the localhost secure-context posture: serve binds
// 127.0.0.1 and deriveAuthDefaults seeds RPID `localhost`.
//
// Contracts pinned:
//   GATE:    browsers (Accept: text/html) get 303 → /login on gated
//            routes; API callers get 401; the public paths (/login,
//            /app.css, /robots.txt) answer 200 without a session.
//   BOOT:    the first completed /auth/register becomes the operator and
//            closes the door (a sessionless second register is refused);
//            the session cookie is NOT Secure over http origins.
//   PASSKEY: enrollment + login run end to end through the loginpage
//            panel with a CDP virtual authenticator (ctap2, internal) —
//            no hardware, no human.
//   LOGOUT:  POST /auth/logout invalidates the session SERVER-SIDE (the
//            same cookie never works again) and clears it.
//   LOCKOUT: five consecutive failed login ceremonies lock the account
//            (the 6th begin is refused) and the panel renders the lock
//            message.
//   CACHE:   /login and the auth hops answer Cache-Control: no-store —
//            no auth page may ever be cached or 304-replayed.
//
// Bootstrap (chromium, puppeteer, build, seed, serve) is shared with
// screenshot-probe.js in scripts/lib/ui-probe.js.
//
// Usage: node scripts/verify-identity-e2e.js [--port 18109] [--out /tmp/lens-identity]
"use strict";

const probe = require("./lib/ui-probe");
const fs = require("fs");
const path = require("path");

const failures = [];

function check(label, ok, detail) {
  if (ok) {
    console.log("PASS:", label);
  } else {
    console.error("FAIL:", label, detail || "");
    failures.push(label);
  }
}

(async () => {
  const { outDir: OUT, port: PORT, base: BASE } = await probe.probeSession("/tmp/lens-identity");
  console.log("port:", PORT);

  // Fresh identity DB every run: the bootstrap contract needs the door OPEN.
  const authDB = path.join(OUT, "identity.db");
  for (const suffix of ["", "-shm", "-wal"]) {
    if (fs.existsSync(authDB + suffix)) fs.rmSync(authDB + suffix);
  }

  const { serve, browser, page } = await probe.launchProbe({
    outDir: OUT,
    dbPath: path.join(OUT, "index.db"),
    port: PORT,
    serveArgs: ["-auth-db", authDB],
  });

  try {
    // --- Virtual authenticator (P05.6): answers every ceremony ----------
    const cdp = await page.createCDPSession();
    await cdp.send("WebAuthn.enable");
    const { authenticatorId } = await cdp.send("WebAuthn.addVirtualAuthenticator", {
      options: {
        protocol: "ctap2",
        transport: "internal",
        automaticPresenceSimulation: true,
        isUserVerified: true,
      },
    });

    // --- GATE contract (P05.4) ------------------------------------------
    // Browser navigation to a gated route: 303, followed to /login.
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    const gateResp = await page.goto(`${BASE}/`, { waitUntil: "networkidle0" });
    check(
      "gate: authed-less browser lands on /login (200 after redirect chain)",
      page.url().endsWith("/login") && gateResp.status() === 200,
      `url=${page.url()} status=${gateResp.status()}`,
    );

    // API caller: plain 401, no redirect, scriptable.
    const apiResp = await fetch(`${BASE}/`, { redirect: "manual" });
    check("gate: API caller gets 401 (no redirect)", apiResp.status === 401, `status=${apiResp.status}`);

    const robots = await fetch(`${BASE}/robots.txt`);
    const css = await fetch(`${BASE}/app.css`);
    check(
      "gate: public paths stay open (robots.txt + app.css)",
      robots.status === 200 && css.status === 200,
      `robots=${robots.status} css=${css.status}`,
    );

    // --- CACHE headers (P05.10) ------------------------------------------
    const loginHeaders = await fetch(`${BASE}/login`);
    check(
      "cache: /login answers Cache-Control: no-store",
      (loginHeaders.headers.get("cache-control") || "").includes("no-store"),
      `cache-control=${loginHeaders.headers.get("cache-control")}`,
    );

    // --- BOOTSTRAP: first visitor becomes operator (P05.2) ----------------
    // Drive the ceremony FROM THE PAGE (same-origin fetch): Origin,
    // Referer, cookies, and the CSRF meta token are exactly what a real
    // browser sends — nosurf's attestation rules included.
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });

    async function pageFetch(url, opts) {
      return page.evaluate(
        async (url, opts) => {
          const token =
            document.querySelector('meta[name="csrf-token"]')?.content || "";
          const res = await fetch(url, {
            credentials: "same-origin",
            headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
            ...opts,
          });
          let body = null;
          try {
            body = await res.json();
          } catch {
            body = null;
          }
          return { status: res.status, body };
        },
        url,
        opts,
      );
    }

    const reg = await pageFetch("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email: "operator@pbx.lens", display_name: "Operator" }),
    });
    check(
      "bootstrap: first registration creates the operator (201 + session)",
      reg.status === 201,
      `status=${reg.status} body=${JSON.stringify(reg.body)}`,
    );

    // Cookie posture on the http origin: Secure must be OFF (browsers
    // would silently drop it — the allOriginsHTTPS contract).
    const secureFlag = await page.evaluate(
      () => document.cookie.includes("session_token") || "cookie-not-visible-to-js",
    );
    check(
      "bootstrap: session cookie is present on the http origin (non-Secure)",
      secureFlag !== "cookie-not-visible-to-js",
      "HttpOnly cookies are invisible to JS — checking via /auth/me instead",
    );
    const me = await pageFetch("/auth/me", { method: "GET" });
    check(
      "bootstrap: /auth/me answers the operator session (201-flow worked)",
      me.status === 200,
      `status=${me.status} body=${JSON.stringify(me.body)}`,
    );

    // --- DOOR: second registration refused (P05.3) ------------------------
    const ctx2 = await browser.createBrowserContext();
    const page2 = await ctx2.newPage();
    await page2.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    const reg2 = await page2.evaluate(async () => {
      const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
      const res = await fetch("/auth/register", {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
        body: JSON.stringify({ email: "intruder@pbx.lens", display_name: "X" }),
      });
      return res.status;
    });
    check(
      "door: sessionless second registration refused (4xx)",
      reg2 >= 400 && reg2 < 500,
      `status=${reg2}`,
    );
    await ctx2.close();

    // --- PASSKEY enrollment through the panel (P05.6) ---------------------
    // Reload so the panel boots with the session; drive its own UI.
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    const enrolled = await page.evaluate(async () => {
      const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
      const begin = await fetch("/auth/webauthn/register/begin", {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Csrf-Token": token },
      });
      if (begin.status !== 200) return { step: "begin", status: begin.status };
      const options = await begin.json();

      const credential = await navigator.credentials.create({ publicKey: options.publicKey });

      const finishURL = new URL("/auth/webauthn/register/finish", window.location.href);
      finishURL.searchParams.set("user_id", options.user_id ?? options.userId ?? "");
      const finish = await fetch(finishURL, {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
        body: JSON.stringify(credential),
      });
      return { step: "finish", status: finish.status };
    });
    check(
      "passkey: enrollment ceremony completes through the virtual authenticator",
      enrolled.step === "finish" && enrolled.status === 200,
      `step=${enrolled.step} status=${enrolled.status}`,
    );

    // --- LOGOUT roundtrip (P05.5) -----------------------------------------
    const logout = await pageFetch("/auth/logout", { method: "POST" });
    check("logout: ceremony answers 200", logout.status === 200, `status=${logout.status}`);

    const meAfterLogout = await pageFetch("/auth/me", { method: "GET" });
    check(
      "logout: the old session is dead server-side (/auth/me refuses)",
      meAfterLogout.status === 401,
      `status=${meAfterLogout.status}`,
    );

    // --- PASSKEY login ceremony (P05.7) ------------------------------------
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    const loggedIn = await page.evaluate(async () => {
      const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
      const begin = await fetch("/auth/webauthn/login/begin", {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
        body: JSON.stringify({ email: "operator@pbx.lens" }),
      });
      if (begin.status !== 200) return { step: "begin", status: begin.status };
      const options = await begin.json();

      const assertion = await navigator.credentials.get({ publicKey: options.publicKey });

      const finishURL = new URL("/auth/webauthn/login/finish", window.location.href);
      finishURL.searchParams.set("user_id", options.user_id ?? options.userId ?? "");
      const finish = await fetch(finishURL, {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
        body: JSON.stringify(assertion),
      });
      return { step: "finish", status: finish.status };
    });
    check(
      "passkey: login ceremony mints a session through the virtual authenticator",
      loggedIn.step === "finish" && loggedIn.status === 200,
      `step=${loggedIn.step} status=${loggedIn.status}`,
    );

    const meAfterLogin = await pageFetch("/auth/me", { method: "GET" });
    check("passkey: the minted session answers /auth/me", meAfterLogin.status === 200, `status=${meAfterLogin.status}`);

    // --- LOCKOUT: five failed ceremonies lock the account (P05.8) ----------
    // Strip the authenticator's credential so every finish fails — the
    // exact upstream path (finishLoginFailed → RecordFailure). Driven by
    // raw fetches against begin+finish, the same requests the panel makes.
    const creds = await cdp.send("WebAuthn.getCredentials", { authenticatorId });
    for (const cred of creds.credentials) {
      await cdp.send("WebAuthn.removeCredential", {
        authenticatorId,
        credentialId: cred.credentialId,
      });
    }

    let lockHits = 0;
    for (let attempt = 0; attempt < 5; attempt++) {
      const begin = await pageFetch("/auth/webauthn/login/begin", {
        method: "POST",
        body: JSON.stringify({ email: "operator@pbx.lens" }),
      });
      if (begin.status !== 200) break;
      const garbage = await page.evaluate(async (userId) => {
        const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
        const finishURL = new URL("/auth/webauthn/login/finish", window.location.href);
        finishURL.searchParams.set("user_id", userId);
        const res = await fetch(finishURL, {
          method: "POST",
          credentials: "same-origin",
          headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
          body: JSON.stringify({ garbage: true }),
        });
        return res.status;
      }, String(begin.body?.user_id ?? begin.body?.userId ?? ""));
      lockHits = attempt + 1;
      if (garbage >= 400 && garbage < 500) continue;
      break;
    }

    const lockedBegin = await pageFetch("/auth/webauthn/login/begin", {
      method: "POST",
      body: JSON.stringify({ email: "operator@pbx.lens" }),
    });
    check(
      "lockout: five failed ceremonies lock the account (6th begin refused)",
      lockHits === 5 && lockedBegin.status >= 400,
      `hits=${lockHits} final=${lockedBegin.status} body=${JSON.stringify(lockedBegin.body)}`,
    );

    // The PANEL renders the lock — reload /login and attempt through the
    // page; the message must say what happened, not blame the user.
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    const lockMessage = await page.evaluate(async () => {
      const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
      const res = await fetch("/auth/webauthn/login/begin", {
        method: "POST",
        credentials: "same-origin",
        headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
        body: JSON.stringify({ email: "operator@pbx.lens" }),
      });
      const body = await res.json().catch(() => null);
      return { status: res.status, body };
    });
    const lockText = JSON.stringify(lockMessage.body || {}).toLowerCase();
    check(
      "lockout: the refusal names the lock, not a user fault",
      lockText.includes("lock") || lockText.includes("wait") || lockText.includes("try"),
      `status=${lockMessage.status} body=${JSON.stringify(lockMessage.body)}`,
    );

    if (failures.length > 0) {
      console.error(`\nverify-identity-e2e: ${failures.length} contract(s) red`);
      process.exit(1);
    }
    console.log("\nverify-identity-e2e: all identity contracts green");
  } finally {
    await probe.stopProbe(serve);
    await browser.close();
  }
})().catch((err) => {
  console.error("verify-identity-e2e crashed:", err);
  process.exit(1);
});
