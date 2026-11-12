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
//            the session cookie rides non-Secure on http origins
//            (allOriginsHTTPS) and /auth/me answers it.
//   PASSKEY: enrollment + login run end to end with a CDP virtual
//            authenticator (ctap2, internal) — no hardware, no human,
//            exactly the panel's own ceremony sequence (loginpage
//            assets/login.js: register → begin{user_id} → create →
//            finish?user_id&credential_name; login → begin{email} → get
//            → finish?user_id=<session_key>).
//   LOGOUT:  POST /auth/logout invalidates the session SERVER-SIDE (the
//            same cookie never works again) and clears it.
//   LOCKOUT: five consecutive failed login ceremonies lock the account
//            (the 6th begin is refused) and the refusal names the lock.
//   CACHE:   /login answers Cache-Control: no-store — no auth page may
//            ever be cached or 304-replayed.
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

// The ceremony helpers below mirror loginpage assets/login.js byte for
// byte (envelope unwrap, base64url decode, credential serialization) —
// the panel's own sequence, driven by the probe.
const CEREMONY_JS = `
function b64uToBuf(b64u) {
  const pad = "===".slice((b64u.length + 3) % 4);
  const s = (b64u + pad).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(s);
  const buf = new Uint8Array(raw.length);
  for (let i = 0; i < raw.length; i++) buf[i] = raw.charCodeAt(i);
  return buf;
}
function bufToB64u(buf) {
  let s = "";
  const bytes = new Uint8Array(buf);
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
  return btoa(s).replace(/\\+/g, "-").replace(/\\//g, "_").replace(/=+$/, "");
}
function unwrapPublicKeyEnvelope(opts) {
  if (opts && opts.publicKey && typeof opts.publicKey === "object") return opts.publicKey;
  return opts;
}
function prepareLoginOptions(opts) {
  const src = unwrapPublicKeyEnvelope(opts);
  const p = Object.assign({}, src);
  p.challenge = b64uToBuf(src.challenge);
  if (src.allowCredentials) {
    p.allowCredentials = src.allowCredentials.map((c) => ({
      type: c.type, id: b64uToBuf(c.id), transports: c.transports,
    }));
  }
  return p;
}
function prepareRegOptions(opts) {
  const src = unwrapPublicKeyEnvelope(opts);
  const p = Object.assign({}, src);
  p.challenge = b64uToBuf(src.challenge);
  if (src.user) p.user = Object.assign({}, src.user, { id: b64uToBuf(src.user.id) });
  if (src.excludeCredentials) {
    p.excludeCredentials = src.excludeCredentials.map((c) => ({
      type: c.type, id: b64uToBuf(c.id), transports: c.transports,
    }));
  }
  return p;
}
function serializeAssertion(cred) {
  const r = cred.response;
  const out = {
    id: cred.id, rawId: bufToB64u(cred.rawId), type: cred.type,
    response: {
      authenticatorData: bufToB64u(r.authenticatorData),
      clientDataJSON: bufToB64u(r.clientDataJSON),
      signature: bufToB64u(r.signature),
    },
  };
  if (r.userHandle) out.response.userHandle = bufToB64u(r.userHandle);
  if (cred.getClientExtensionResults) out.clientExtensionResults = cred.getClientExtensionResults();
  return out;
}
function serializeAttestation(cred) {
  const r = cred.response;
  const out = {
    id: cred.id, rawId: bufToB64u(cred.rawId), type: cred.type,
    response: {
      attestationObject: bufToB64u(r.attestationObject),
      clientDataJSON: bufToB64u(r.clientDataJSON),
    },
  };
  if (cred.getClientExtensionResults) out.clientExtensionResults = cred.getClientExtensionResults();
  return out;
}
window.__ceremony = { prepareLoginOptions, prepareRegOptions, serializeAssertion, serializeAttestation };
`;

(async () => {
  const { outDir: OUT, port: PORT, base: BASE } = await probe.probeSession("/tmp/lens-identity");
  console.log("port:", PORT);

  // The ceremonies run on the LOCALHOST origin: deriveAuthDefaults
  // collapses the IP bind to RPID `localhost`, and WebAuthn refuses a
  // ceremony whose origin host is not the RPID (127.0.0.1 ≠ localhost →
  // SecurityError). Browsing localhost is exactly what a real operator
  // does on the localhost posture; Node-level fetches keep BASE.
  const LB = BASE.replace("127.0.0.1", "localhost");

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
    await page.goto(`${LB}/login`, { waitUntil: "networkidle0" });
    const gateResp = await page.goto(`${LB}/`, { waitUntil: "networkidle0" });
    check(
      "gate: authed-less browser lands on /login (200 after redirect chain)",
      page.url().endsWith("/login") && gateResp.status() === 200,
      `url=${page.url()} status=${gateResp.status()}`,
    );

    const apiResp = await fetch(`${BASE}/`, { redirect: "manual" });
    check(
      "gate: API caller gets 401 (no redirect)",
      apiResp.status === 401,
      `status=${apiResp.status}`,
    );

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
    await page.goto(`${LB}/login`, { waitUntil: "networkidle0" });
    await page.evaluate(CEREMONY_JS);

    async function pageFetch(url, opts) {
      return page.evaluate(
        async (url, opts) => {
          const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
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

    const cookies = await page.cookies();
    const sessionCookie = cookies.find((c) => c.name === "session_token");
    check(
      "bootstrap: session cookie rides non-Secure on the http origin",
      sessionCookie && sessionCookie.secure === false,
      `secure=${sessionCookie && sessionCookie.secure}`,
    );

    const me = await pageFetch("/auth/me", { method: "GET" });
    check(
      "bootstrap: /auth/me answers the operator session",
      me.status === 200,
      `status=${me.status} body=${JSON.stringify(me.body)}`,
    );

    const userID = reg.body && reg.body.user && reg.body.user.id;

    // --- DOOR: second registration refused (P05.3) ------------------------
    const ctx2 = await browser.createBrowserContext();
    const page2 = await ctx2.newPage();
    await page2.goto(`${LB}/login`, { waitUntil: "networkidle0" });
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

    // --- PASSKEY enrollment through the panel's ceremony (P05.6) ----------
    const beginReg = await pageFetch("/auth/webauthn/register/begin", {
      method: "POST",
      body: JSON.stringify({ user_id: userID }),
    });
    const enrolled = await page
      .evaluate(
        async (begin, uid) => {
          const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
          const credential = await navigator.credentials.create({
            publicKey: window.__ceremony.prepareRegOptions(begin.body.options),
          });
          const finishURL = new URL("/auth/webauthn/register/finish", window.location.href);
          finishURL.searchParams.set("user_id", uid);
          finishURL.searchParams.set("credential_name", "Probe Passkey");
          const res = await fetch(finishURL, {
            method: "POST",
            credentials: "same-origin",
            headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
            body: JSON.stringify(window.__ceremony.serializeAttestation(credential)),
          });
          return res.status;
        },
        beginReg,
        userID,
      )
      .catch((err) => `crash: ${err.message}`);
    check(
      "passkey: enrollment ceremony completes through the virtual authenticator",
      enrolled === 200,
      `status=${enrolled} beginStatus=${beginReg.status}`,
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
    await page.goto(`${LB}/login`, { waitUntil: "networkidle0" });
    await page.evaluate(CEREMONY_JS);

    const beginLogin = await pageFetch("/auth/webauthn/login/begin", {
      method: "POST",
      body: JSON.stringify({ email: "operator@pbx.lens" }),
    });
    const loggedIn = await page
      .evaluate(async (begin) => {
        const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
        const assertion = await navigator.credentials.get({
          publicKey: window.__ceremony.prepareLoginOptions(begin.body.options),
        });
        const finishURL = new URL("/auth/webauthn/login/finish", window.location.href);
        finishURL.searchParams.set("user_id", begin.body.session_key);
        const res = await fetch(finishURL, {
          method: "POST",
          credentials: "same-origin",
          headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
          body: JSON.stringify(window.__ceremony.serializeAssertion(assertion)),
        });
        return res.status;
      }, beginLogin)
      .catch((err) => `crash: ${err.message}`);
    check(
      "passkey: login ceremony mints a session through the virtual authenticator",
      loggedIn === 200,
      `status=${loggedIn} beginStatus=${beginLogin.status}`,
    );

    const meAfterLogin = await pageFetch("/auth/me", { method: "GET" });
    check(
      "passkey: the minted session answers /auth/me",
      meAfterLogin.status === 200,
      `status=${meAfterLogin.status}`,
    );

    // --- LOCKOUT: five failed ceremonies lock the account (P05.8) ----------
    // Strip the authenticator's credential so every finish fails — the
    // exact upstream path (finishLoginFailed → RecordFailure), driven by
    // the same begin+finish requests the panel makes.
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
      const finishStatus = await page.evaluate(
        async (sessionKey) => {
          const token = document.querySelector('meta[name="csrf-token"]')?.content || "";
          const finishURL = new URL("/auth/webauthn/login/finish", window.location.href);
          finishURL.searchParams.set("user_id", sessionKey);
          const res = await fetch(finishURL, {
            method: "POST",
            credentials: "same-origin",
            headers: { "X-Csrf-Token": token, "Content-Type": "application/json" },
            body: JSON.stringify({ garbage: true }),
          });
          return res.status;
        },
        String(begin.body?.session_key ?? ""),
      );
      lockHits = attempt + 1;
      if (finishStatus === 200) break;
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

    const lockText = JSON.stringify(lockedBegin.body || {}).toLowerCase();
    check(
      "lockout: the refusal names the lock, not a user fault",
      lockText.includes("lock") || lockText.includes("wait") || lockText.includes("try"),
      `status=${lockedBegin.status} body=${JSON.stringify(lockedBegin.body)}`,
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
