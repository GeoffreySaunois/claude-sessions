// Regenerates the README screenshots with FAKE data (never real ~/.claude
// content). It runs a real ccs server but intercepts the API with the fixtures
// below, so the captures are deterministic and privacy-safe.
//
// Regen:
//   1. start a server:   ./backend/target/release/ccs --addr 127.0.0.1:7799
//   2. have playwright-core available (e.g. `npm i playwright-core` here)
//   3. node docs/capture/capture.mjs
//
// Env: BASE (default http://127.0.0.1:7799), OUTDIR (default docs/).
import { chromium } from "playwright-core";

const BASE = process.env.BASE || "http://127.0.0.1:7799";
const OUTDIR = process.env.OUTDIR || "docs";
const now = Date.now();
const iso = (agoMin) => new Date(now - agoMin * 60000).toISOString();

// Fake pinned sessions across two folders, with categories / tags / statuses.
const mk = (o) => ({
  id: o.id, path: "", projectDir: "", cwd: o.cwd, gitBranch: o.branch,
  title: o.title, lastMessage: o.msg, kind: "main", status: o.status,
  pid: 0, lastActive: iso(o.agoMin), sizeBytes: 0, version: "2.1.0",
  pinned: true, category: o.category, tags: o.tags, archived: false,
});
const SESSIONS = [
  mk({ id: "a1", cwd: "/Users/dev/Projects/acme-api", branch: "feat/ratelimit", title: "Rate limiter middleware", category: "Backend", tags: ["perf", "wip"], status: "busy", agoMin: 0, msg: "Implemented a token-bucket limiter; the p99 stays under 2 ms in the load test — wiring it into the gateway next." }),
  mk({ id: "a2", cwd: "/Users/dev/Projects/acme-api", branch: "fix/jwt-refresh", title: "Fix JWT refresh race", category: "Backend", tags: ["auth", "bug"], status: "waiting", agoMin: 8, msg: "Reproduced the double-refresh under concurrent requests; proposing a mutex around the token swap. Does that match what you saw?" }),
  mk({ id: "a3", cwd: "/Users/dev/Projects/acme-api", branch: "main", title: "Postgres migration 0007", category: "Infra", tags: ["ci"], status: "inactive", agoMin: 124, msg: "Migration applied to staging and the rollback path is tested — ready for prod when you are." }),
  mk({ id: "w1", cwd: "/Users/dev/Projects/web-app", branch: "feat/dashboard", title: "Dashboard redesign", category: "Frontend", tags: ["wip"], status: "busy", agoMin: 1, msg: "Switched the card grid to CSS subgrid so the columns line up across rows — pushing a preview." }),
  mk({ id: "w2", cwd: "/Users/dev/Projects/web-app", branch: "main", title: "Debounce search input", category: "Frontend", tags: ["perf"], status: "inactive", agoMin: 1440, msg: "Added a 250 ms debounce on the search box; network calls dropped about 80%." }),
  mk({ id: "w3", cwd: "/Users/dev/Projects/web-app", branch: "main", title: "API client types RFC", category: "Research", tags: ["rfc"], status: "inactive", agoMin: 4320, msg: "Comparing codegen options for the OpenAPI schema — leaning toward a typed fetch wrapper." }),
];
const OPTIONS = {
  categories: ["Backend", "Frontend", "Infra", "Research"],
  tags: ["auth", "bug", "ci", "perf", "rfc", "wip"],
};

const browser = await chromium.launch({ channel: "chrome", headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 940 }, deviceScaleFactor: 2 });
await page.route("**/api/sessions", (r) => r.fulfill({ contentType: "application/json", body: JSON.stringify(SESSIONS) }));
await page.route("**/api/options", (r) => r.fulfill({ contentType: "application/json", body: JSON.stringify(OPTIONS) }));
await page.route("**/api/search**", (r) => r.fulfill({ contentType: "application/json", body: JSON.stringify({ matches: {} }) }));
await page.goto(BASE, { waitUntil: "networkidle" });
await page.waitForTimeout(700);
await page.screenshot({ path: `${OUTDIR}/screenshot.png`, fullPage: true });
console.log(`wrote ${OUTDIR}/screenshot.png`);
await browser.close();
