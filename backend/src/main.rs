//! ccs serves the Claude Code sessions dashboard: a JSON API over `~/.claude`
//! plus an embedded SPA, bound to localhost only. The bind address comes from
//! `--addr`, then `CCS_ADDR`, then the default `127.0.0.1:7799`.

mod assets;
mod config;
mod discover;
mod loader;
mod meta;
mod open;
mod search;
mod server;
mod session;
mod status;
mod timefmt;
mod transcript;

const DEFAULT_ADDR: &str = "127.0.0.1:7799";

#[tokio::main]
async fn main() {
    let addr = resolve_addr();
    let listener = match tokio::net::TcpListener::bind(&addr).await {
        Ok(l) => l,
        Err(e) => {
            eprintln!("ccs: failed to bind {addr}: {e}");
            std::process::exit(1);
        }
    };
    eprintln!("ccs: listening on http://{addr}");
    if let Err(e) = axum::serve(listener, server::router()).await {
        eprintln!("ccs: server error: {e}");
        std::process::exit(1);
    }
}

/// resolve_addr reads the bind address from `--addr <value>` (or `--addr=value`),
/// then `CCS_ADDR`, then the default.
fn resolve_addr() -> String {
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        if let Some(value) = arg.strip_prefix("--addr=") {
            return value.to_string();
        }
        if arg == "--addr" {
            if let Some(value) = args.next() {
                return value;
            }
        }
    }
    if let Ok(addr) = std::env::var("CCS_ADDR") {
        if !addr.is_empty() {
            return addr;
        }
    }
    DEFAULT_ADDR.to_string()
}
