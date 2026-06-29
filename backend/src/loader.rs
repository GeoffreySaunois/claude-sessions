//! Loader assembles the full session list: discovered transcripts enriched with
//! live status and the user's organization metadata, sorted most-recently-active
//! first.

use crate::discover::discover_transcripts;
use crate::meta::MetaStore;
use crate::session::Session;
use crate::status::resolve_live_statuses;

/// load_sessions returns every discovered session, enriched with live status and
/// the user's organization metadata, sorted most-recently-active first.
/// Intended to run on a blocking pool — it does synchronous disk I/O.
pub fn load_sessions() -> std::io::Result<Vec<Session>> {
    let mut sessions = discover_transcripts()?;
    let live = resolve_live_statuses();
    let store = MetaStore::load()?;
    for s in &mut sessions {
        if let Some(info) = live.get(&s.id) {
            s.status = info.status;
            s.pid = info.pid;
        }
        store.apply(s);
    }
    // Newest active first, by the numeric mtime so variable-width RFC3339
    // fractional seconds can't perturb the order.
    sessions.sort_by(|a, b| b.last_active_nanos.cmp(&a.last_active_nanos));
    Ok(sessions)
}
