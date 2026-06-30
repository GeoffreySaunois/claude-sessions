//! Server wires the dashboard's JSON API and the embedded SPA over axum, binding
//! to localhost only. Each handler loads a fresh `MetaStore` so it never serves
//! stale organization state. Heavy scans (the session list, full-text search)
//! run on the blocking pool so they never stall the async reactor.

use axum::extract::{Path, Query};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::{Json, Router};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

use crate::assets;
use crate::loader::load_sessions;
use crate::meta::{MetaStore, SessionMeta};
use crate::open::{open, OpenConfig};
use crate::search::search_transcripts;
use crate::session::Session;

/// router builds the full route table: SPA at `/`, JSON API under `/api`, and
/// every other path resolved against the embedded assets.
pub fn router() -> Router {
    Router::new()
        .route("/", get(assets::index))
        .route("/api/sessions", get(handle_sessions))
        .route("/api/search", get(handle_search))
        .route(
            "/api/options",
            get(handle_get_options).post(handle_add_option),
        )
        .route("/api/meta", post(handle_meta))
        .route("/api/pin", post(handle_pin))
        .route("/api/bulk", post(handle_bulk))
        .route("/api/open", post(handle_open))
        .route("/{*path}", get(handle_asset))
}

/// internal_error maps any IO/store failure to a 500, mirroring the Go handlers'
/// `http.Error(..., 500)`.
fn internal_error<E: std::fmt::Display>(e: E) -> Response {
    (StatusCode::INTERNAL_SERVER_ERROR, e.to_string()).into_response()
}

fn bad_request(msg: &str) -> Response {
    (StatusCode::BAD_REQUEST, msg.to_string()).into_response()
}

async fn handle_asset(Path(path): Path<String>) -> Response {
    assets::asset(&path).await
}

async fn handle_sessions() -> Response {
    match tokio::task::spawn_blocking(load_sessions).await {
        Ok(Ok(sessions)) => Json::<Vec<Session>>(sessions).into_response(),
        Ok(Err(e)) => internal_error(e),
        Err(e) => internal_error(e),
    }
}

#[derive(Deserialize)]
struct SearchParams {
    #[serde(default)]
    q: String,
}

/// handle_search runs a full-text search over every transcript and returns a map
/// of session id to a snippet around the first match.
async fn handle_search(Query(params): Query<SearchParams>) -> Response {
    let q = params.q;
    match tokio::task::spawn_blocking(move || search_transcripts(&q)).await {
        Ok(Ok(matches)) => Json(SearchResponse { matches }).into_response(),
        Ok(Err(e)) => internal_error(e),
        Err(e) => internal_error(e),
    }
}

#[derive(Serialize)]
struct SearchResponse {
    matches: HashMap<String, String>,
}

#[derive(Serialize)]
struct OptionsResponse {
    categories: Vec<String>,
    tags: Vec<String>,
}

async fn handle_get_options() -> Response {
    let store = match MetaStore::load() {
        Ok(s) => s,
        Err(e) => return internal_error(e),
    };
    let (categories, tags) = store.options();
    Json(OptionsResponse { categories, tags }).into_response()
}

#[derive(Deserialize)]
struct AddOptionRequest {
    /// "category" | "tag"
    #[serde(default)]
    kind: String,
    #[serde(default)]
    name: String,
}

async fn handle_add_option(
    body: Result<Json<AddOptionRequest>, axum::extract::rejection::JsonRejection>,
) -> Response {
    let Json(req) = match body {
        Ok(j) => j,
        Err(e) => return bad_request(&e.to_string()),
    };
    if req.name.is_empty() {
        return bad_request("missing name");
    }
    let mut store = match MetaStore::load() {
        Ok(s) => s,
        Err(e) => return internal_error(e),
    };
    let res = match req.kind.as_str() {
        "category" => store.add_category(&req.name),
        "tag" => store.add_tag(&req.name),
        _ => return bad_request("kind must be category or tag"),
    };
    match res {
        Ok(()) => ok_response(),
        Err(e) => internal_error(e),
    }
}

/// MetaRequest is the body of POST /api/meta. Each field is an `Option` so the
/// handler can tell "absent" from "set to zero value" and apply ONLY the fields
/// the client sent — an archive toggle must not wipe tags. (Matching the Go
/// reference, `pinned` is not part of this patch; pinning goes through /api/pin.)
#[derive(Deserialize)]
struct MetaRequest {
    #[serde(default)]
    id: String,
    #[serde(default)]
    category: Option<String>,
    #[serde(default)]
    tags: Option<Vec<String>>,
    #[serde(default)]
    archived: Option<bool>,
    /// Rename override: present+non-empty sets it, present+empty clears it back
    /// to the derived title, absent leaves it untouched.
    #[serde(default)]
    title: Option<String>,
}

async fn handle_meta(
    body: Result<Json<MetaRequest>, axum::extract::rejection::JsonRejection>,
) -> Response {
    let Json(req) = match body {
        Ok(j) => j,
        Err(e) => return bad_request(&e.to_string()),
    };
    if req.id.is_empty() {
        return bad_request("missing id");
    }
    let mut store = match MetaStore::load() {
        Ok(s) => s,
        Err(e) => return internal_error(e),
    };
    let id = req.id.clone();
    let res = store.update(&id, |m: &mut SessionMeta| apply_meta_patch(&req, m));
    match res {
        Ok(()) => ok_response(),
        Err(e) => internal_error(e),
    }
}

/// apply_meta_patch sets only the fields present in `req`, leaving every absent
/// field untouched.
fn apply_meta_patch(req: &MetaRequest, m: &mut SessionMeta) {
    if let Some(category) = &req.category {
        m.category = category.clone();
    }
    if let Some(tags) = &req.tags {
        m.tags = tags.clone();
    }
    if let Some(archived) = req.archived {
        m.archived = archived;
    }
    if let Some(title) = &req.title {
        // Present+empty clears the override; present+non-empty sets it.
        m.title = if title.is_empty() {
            None
        } else {
            Some(title.clone())
        };
    }
}

#[derive(Deserialize)]
struct PinRequest {
    #[serde(default)]
    id: String,
    #[serde(default)]
    pinned: bool,
}

async fn handle_pin(
    body: Result<Json<PinRequest>, axum::extract::rejection::JsonRejection>,
) -> Response {
    let Json(req) = match body {
        Ok(j) => j,
        Err(e) => return bad_request(&e.to_string()),
    };
    if req.id.is_empty() {
        return bad_request("missing id");
    }
    let mut store = match MetaStore::load() {
        Ok(s) => s,
        Err(e) => return internal_error(e),
    };
    match store.set_pinned(&req.id, req.pinned) {
        Ok(()) => ok_response(),
        Err(e) => internal_error(e),
    }
}

#[derive(Deserialize)]
struct BulkRequest {
    #[serde(default)]
    ids: Vec<String>,
    /// "pin" | "unpin" | "archive" | "unarchive" | "category"
    #[serde(default)]
    action: String,
    /// category name when action == "category"
    #[serde(default)]
    value: String,
}

async fn handle_bulk(
    body: Result<Json<BulkRequest>, axum::extract::rejection::JsonRejection>,
) -> Response {
    let Json(req) = match body {
        Ok(j) => j,
        Err(e) => return bad_request(&e.to_string()),
    };
    let mut store = match MetaStore::load() {
        Ok(s) => s,
        Err(e) => return internal_error(e),
    };
    match apply_bulk(&mut store, &req) {
        Ok(()) => Json(BulkResponse {
            updated: req.ids.len(),
        })
        .into_response(),
        Err(BulkError::UnknownAction(a)) => bad_request(&format!("unknown action: {a}")),
        Err(BulkError::Store(e)) => internal_error(e),
    }
}

#[derive(Serialize)]
struct BulkResponse {
    updated: usize,
}

enum BulkError {
    UnknownAction(String),
    Store(std::io::Error),
}

/// apply_bulk dispatches a bulk action. Pin/unpin adopt or release sessions (and
/// unpin clears their organization), so they go through set_pinned_many; the
/// remaining actions edit existing metadata.
fn apply_bulk(store: &mut MetaStore, req: &BulkRequest) -> Result<(), BulkError> {
    match req.action.as_str() {
        "pin" => store
            .set_pinned_many(&req.ids, true)
            .map_err(BulkError::Store),
        "unpin" => store
            .set_pinned_many(&req.ids, false)
            .map_err(BulkError::Store),
        "archive" => store
            .update_many(&req.ids, |m| m.archived = true)
            .map_err(BulkError::Store),
        "unarchive" => store
            .update_many(&req.ids, |m| m.archived = false)
            .map_err(BulkError::Store),
        "category" => {
            let value = req.value.clone();
            store
                .update_many(&req.ids, |m| m.category = value.clone())
                .map_err(BulkError::Store)
        }
        other => Err(BulkError::UnknownAction(other.to_string())),
    }
}

#[derive(Deserialize)]
struct OpenRequest {
    #[serde(default)]
    ids: Vec<String>,
}

async fn handle_open(
    body: Result<Json<OpenRequest>, axum::extract::rejection::JsonRejection>,
) -> Response {
    let Json(req) = match body {
        Ok(j) => j,
        Err(e) => return bad_request(&e.to_string()),
    };
    let sessions = match tokio::task::spawn_blocking(load_sessions).await {
        Ok(Ok(s)) => s,
        Ok(Err(e)) => return internal_error(e),
        Err(e) => return internal_error(e),
    };
    let matched = select_by_ids(&sessions, &req.ids);
    let opened = matched.len();
    if let Err(e) = open(&matched, &OpenConfig::from_config()) {
        return internal_error(e);
    }
    Json(OpenResponse { opened }).into_response()
}

#[derive(Serialize)]
struct OpenResponse {
    opened: usize,
}

/// select_by_ids returns the sessions whose id is in `ids`, preserving the order
/// of `ids` and skipping any that don't resolve.
fn select_by_ids(sessions: &[Session], ids: &[String]) -> Vec<Session> {
    let by_id: HashMap<&str, &Session> = sessions.iter().map(|s| (s.id.as_str(), s)).collect();
    ids.iter()
        .filter_map(|id| by_id.get(id.as_str()).map(|s| (*s).clone()))
        .collect()
}

fn ok_response() -> Response {
    Json(serde_json::json!({ "ok": true })).into_response()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn session(id: &str) -> Session {
        Session {
            id: id.to_string(),
            path: String::new(),
            project_dir: String::new(),
            cwd: String::new(),
            git_branch: String::new(),
            title: String::new(),
            last_message: String::new(),
            kind: crate::session::Kind::Main,
            status: crate::session::Status::Inactive,
            pid: 0,
            last_active: String::new(),
            last_active_nanos: 0,
            size_bytes: 0,
            version: String::new(),
            pinned: false,
            category: String::new(),
            tags: Vec::new(),
            archived: false,
        }
    }

    // select_by_ids preserves request order and silently drops unknown ids — a
    // bug here would open the wrong sessions or in the wrong split order.
    #[test]
    fn select_by_ids_preserves_order_and_drops_unknown() {
        let sessions = vec![session("a"), session("b"), session("c")];
        let got = select_by_ids(&sessions, &["c".into(), "missing".into(), "a".into()]);
        assert_eq!(got.len(), 2);
        assert_eq!(got[0].id, "c");
        assert_eq!(got[1].id, "a");
    }

    // apply_meta_patch must touch ONLY the fields present, so an archive toggle
    // doesn't wipe tags and a tag edit doesn't reset the category.
    #[test]
    fn meta_patch_only_touches_present_fields() {
        let mut m = SessionMeta {
            category: "work".into(),
            tags: vec!["x".into(), "y".into()],
            archived: false,
            pinned: true,
            title: Some("renamed".into()),
        };
        apply_meta_patch(
            &MetaRequest {
                id: "s".into(),
                category: None,
                tags: None,
                archived: Some(true),
                title: None,
            },
            &mut m,
        );
        assert!(m.archived);
        assert_eq!(m.category, "work");
        assert_eq!(m.tags.len(), 2);
        // An archived-only patch must leave the title override intact.
        assert_eq!(m.title.as_deref(), Some("renamed"));

        // Clearing a category sends an empty string (present), not absent.
        apply_meta_patch(
            &MetaRequest {
                id: "s".into(),
                category: Some(String::new()),
                tags: None,
                archived: None,
                title: None,
            },
            &mut m,
        );
        assert_eq!(m.category, "");
        assert_eq!(m.tags.len(), 2);
    }

    // A present+empty title clears the override (reverts to derived); a
    // present+non-empty title sets it; absent leaves it untouched.
    #[test]
    fn meta_patch_title_set_clear_and_untouched() {
        let mut m = SessionMeta::default();

        apply_meta_patch(
            &MetaRequest {
                id: "s".into(),
                category: None,
                tags: None,
                archived: None,
                title: Some("My rename".into()),
            },
            &mut m,
        );
        assert_eq!(m.title.as_deref(), Some("My rename"));

        // Absent title leaves the override untouched.
        apply_meta_patch(
            &MetaRequest {
                id: "s".into(),
                category: Some("work".into()),
                tags: None,
                archived: None,
                title: None,
            },
            &mut m,
        );
        assert_eq!(m.title.as_deref(), Some("My rename"));

        // Present+empty clears it back to the derived title.
        apply_meta_patch(
            &MetaRequest {
                id: "s".into(),
                category: None,
                tags: None,
                archived: None,
                title: Some(String::new()),
            },
            &mut m,
        );
        assert_eq!(m.title, None);
    }
}
