//! Assets embeds the built frontend (`../frontend/dist`) into the binary at
//! compile time, so the shipped artifact is a single executable. `/` serves
//! `index.html`; every other path resolves to its embedded file with a guessed
//! content-type.

use axum::body::Body;
use axum::http::{header, StatusCode};
use axum::response::{IntoResponse, Response};
use rust_embed::RustEmbed;

#[derive(RustEmbed)]
#[folder = "../frontend/dist"]
struct Frontend;

/// index serves the SPA entrypoint at `/`.
pub async fn index() -> Response {
    serve("index.html")
}

/// asset serves an embedded file by path, falling back to a 404 when absent.
pub async fn asset(path: &str) -> Response {
    serve(path.trim_start_matches('/'))
}

fn serve(path: &str) -> Response {
    match Frontend::get(path) {
        Some(file) => {
            let mime = mime_guess::from_path(path).first_or_octet_stream();
            (
                [(header::CONTENT_TYPE, mime.as_ref().to_string())],
                Body::from(file.data.into_owned()),
            )
                .into_response()
        }
        None => (StatusCode::NOT_FOUND, "not found").into_response(),
    }
}
