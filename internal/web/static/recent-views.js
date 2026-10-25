// Recent views (plan P14): the calls page records every FILTERED view in
// localStorage and renders the five most recent as one-click links. The
// server ships only the stable container, so the page body stays
// byte-stable for the ETag contract and no-JS visitors lose nothing they
// had before. External same-origin script: loads under both CSP postures
// (script-src 'self' covers it; no nonce needed).
(() => {
  "use strict";

  const STORAGE_KEY = "lens.recentViews";
  const CAP = 5;

  const host = document.getElementById("recent-views");
  if (!host) return;

  const readAll = () => {
    try {
      const parsed = JSON.parse(window.localStorage.getItem(STORAGE_KEY) || "[]");

      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  };

  const writeAll = (entries) => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(entries));
    } catch {
      // storage unavailable (private mode, quota): the list simply
      // stays session-local
    }
  };

  const labelFor = (filters) => {
    const parts = [];

    if (filters.direction) parts.push(filters.direction);
    if (filters.ai) parts.push(filters.ai === "yes" ? "AI" : "Human");
    if (filters.number) parts.push(filters.number);
    if (filters.from || filters.to) parts.push(`${filters.from || "…"} – ${filters.to || "…"}`);

    return parts.length ? parts.join(" · ") : "Filtered view";
  };

  const render = (entries) => {
    host.textContent = "";

    const title = document.createElement("p");
    title.className = "dark:text-gray-400 mb-1 text-gray-500 text-xs";
    title.textContent = "Recent views";
    host.appendChild(title);

    if (!entries.length) {
      const empty = document.createElement("p");
      empty.className = "dark:text-gray-400 text-gray-500 text-sm";
      empty.textContent = "Filtered views you open will appear here.";
      host.appendChild(empty);

      return;
    }

    const list = document.createElement("ul");
    list.className = "flex flex-wrap gap-2";

    for (const entry of entries) {
      const item = document.createElement("li");
      const link = document.createElement("a");
      link.href = entry.u;
      link.className =
        "rounded-full border border-gray-200 px-3 py-1 text-gray-500 text-sm transition-colors hover:text-gray-700 motion-reduce:transition-none dark:border-gray-800 dark:text-gray-400";
      link.textContent = entry.l;
      item.appendChild(link);
      list.appendChild(item);
    }

    host.appendChild(list);
  };

  const raw = host.getAttribute("data-filters");
  if (raw && raw !== "{}") {
    try {
      const filters = JSON.parse(raw);
      const url = `${window.location.pathname}${window.location.search}`;
      const entries = readAll().filter((entry) => entry.u !== url);

      entries.unshift({ u: url, l: labelFor(filters), at: Date.now() });
      writeAll(entries.slice(0, CAP));
    } catch {
      // a malformed payload must never break the page
    }
  }

  render(readAll());
})();
