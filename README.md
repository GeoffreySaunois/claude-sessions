# claude-sessions

A local dashboard to browse, organize, and resume your Claude Code sessions.

It reads Claude Code's own files under `~/.claude` (`projects/` transcripts and
`sessions/` live status) read-only, and stores your organization (pins,
categories, tags, archive) in a separate sidecar at `~/.claude/session-ui/meta.json`.
It never modifies Claude Code's data.

## Stack

- **Backend** — Rust + axum. Ports the discovery, status, metadata, full-text
  search, and Ghostty "open" logic; serves the JSON API and the embedded SPA as a
  single binary. (`backend/`)
- **Frontend** — Svelte 5 + Vite + TypeScript + Tailwind, built to `frontend/dist`
  and embedded into the Rust binary. (`frontend/`)

## Features

- Curated **pinned workspace** (home) + a **Browse all** modal to discover and pin.
- Notion-style **category** (single-select) and **tags** (multi-select).
- **Archive** (pinned-but-set-aside), collapsible groups, bulk actions.
- Live **status** (busy / waiting / inactive) and last-message previews.
- **Full-text search** across full conversation content, with highlighted snippets.
- **Open** selected sessions as native Ghostty splits (`claude --resume`).
- Light / dark / system theme.

## Build & run

```sh
# build the SPA (only needed when the frontend changes), then the binary
pnpm -C frontend install
pnpm -C frontend build
cargo build --release --manifest-path backend/Cargo.toml --locked

# run (serves http://127.0.0.1:7799)
./backend/target/release/ccs            # or --addr 127.0.0.1:PORT
```

Frontend dev with hot reload (proxies `/api` to a running backend on :7799):

```sh
pnpm -C frontend dev
```

## Opening sessions (macOS / Ghostty)

Ghostty has no CLI to create a split running a command, so "Open" drives the
native splits via AppleScript keystrokes. The first use will prompt to allow the
controlling app under **System Settings → Privacy & Security → Accessibility**.
