# README screenshot capture

`../screenshot.png` is generated from **fabricated** data (defined inline in
`capture.mjs`) — never from real `~/.claude` content — so it's deterministic and
safe to publish.

## Regenerate

```sh
# 1. build + run a server (any port)
cargo build --release --manifest-path backend/Cargo.toml --locked
./backend/target/release/ccs --addr 127.0.0.1:7799 &

# 2. make playwright-core available here, then capture
#    (uses your installed Google Chrome via channel: "chrome")
cd docs/capture && npm i playwright-core && cd ../..
node docs/capture/capture.mjs        # writes docs/screenshot.png
```

The script intercepts `/api/sessions`, `/api/options`, and `/api/search` with the
fake fixtures, so the running server's real data is never shown. Edit the
`SESSIONS` / `OPTIONS` arrays in `capture.mjs` to change what's pictured. The
`agoMin` offsets are converted to timestamps at capture time, so relative times
("just now", "8m ago") always look fresh.

`node_modules/` here is git-ignored.
