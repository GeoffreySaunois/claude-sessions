//! Meta carries the user-maintained organization data — pin, category, tags,
//! archive — in a sidecar file under `~/.claude/session-ui/meta.json`, alongside
//! the universe of category and tag options offered in the UI. It never touches
//! Claude Code's own state.

use std::collections::BTreeMap;
use std::fs;
use std::path::PathBuf;

use serde::{Deserialize, Deserializer, Serialize};

use crate::session::{claude_dir, Session};

/// null_as_default deserializes a JSON value into `T`, treating an explicit
/// `null` as `T::default()`. Go marshals nil maps and slices as `null`, so the
/// sidecar this dashboard inherits can carry `"tags": null` / `"sessions": null`
/// where serde would otherwise reject the type.
fn null_as_default<'de, D, T>(d: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Default + Deserialize<'de>,
{
    let opt = Option::<T>::deserialize(d)?;
    Ok(opt.unwrap_or_default())
}

/// SessionMeta is the user-maintained organization data for one session.
/// `pinned` marks a session as adopted into the curated dashboard;
/// category/tags/archived only carry meaning for pinned sessions. `title` is an
/// optional rename that overrides the transcript-derived title; it is
/// independent of pinning.
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
pub struct SessionMeta {
    #[serde(default, deserialize_with = "null_as_default")]
    pub pinned: bool,
    #[serde(default, deserialize_with = "null_as_default")]
    pub category: String,
    #[serde(default, deserialize_with = "null_as_default")]
    pub tags: Vec<String>,
    #[serde(default, deserialize_with = "null_as_default")]
    pub archived: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
}

/// MetaDoc is the on-disk shape of the sidecar: per-session metadata plus the
/// option universe.
#[derive(Default, Serialize, Deserialize)]
struct MetaDoc {
    #[serde(default, deserialize_with = "null_as_default")]
    sessions: BTreeMap<String, SessionMeta>,
    #[serde(default, deserialize_with = "null_as_default")]
    categories: Vec<String>,
    #[serde(default, deserialize_with = "null_as_default")]
    tags: Vec<String>,
}

/// MetaStore is the persistent collection of per-session organization metadata
/// plus the universe of category and tag options offered in the UI (so an option
/// can exist before any session uses it, Notion-style). Each handler loads a
/// fresh store so it never serves stale state; persistence is process-external,
/// so no in-process locking is needed.
pub struct MetaStore {
    path: PathBuf,
    data: BTreeMap<String, SessionMeta>,
    categories: Vec<String>,
    tags: Vec<String>,
}

fn meta_path() -> PathBuf {
    claude_dir().join("session-ui").join("meta.json")
}

impl MetaStore {
    /// load reads the sidecar file, returning an empty store if none exists. The
    /// option universe is seeded from any categories/tags already in use so the
    /// current data always appears as selectable options.
    pub fn load() -> std::io::Result<MetaStore> {
        let path = meta_path();
        let mut ms = MetaStore {
            path,
            data: BTreeMap::new(),
            categories: Vec::new(),
            tags: Vec::new(),
        };
        let b = match fs::read(&ms.path) {
            Ok(b) => b,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(ms),
            Err(e) => return Err(e),
        };
        let doc: MetaDoc = serde_json::from_slice(&b)
            .map_err(|e| std::io::Error::new(std::io::ErrorKind::InvalidData, e))?;
        ms.data = doc.sessions;
        ms.categories = doc.categories;
        ms.tags = doc.tags;
        ms.seed_options_from_usage();
        Ok(ms)
    }

    /// get returns the stored metadata for a session, or the default if unset.
    pub fn get(&self, id: &str) -> SessionMeta {
        self.data.get(id).cloned().unwrap_or_default()
    }

    /// options returns the category and tag universes for the UI selects, sorted.
    pub fn options(&self) -> (Vec<String>, Vec<String>) {
        (self.categories.clone(), self.tags.clone())
    }

    /// add_category registers a category option without assigning it to a session.
    pub fn add_category(&mut self, name: &str) -> std::io::Result<()> {
        insert_sorted(&mut self.categories, name);
        self.save()
    }

    /// add_tag registers a tag option without assigning it to a session.
    pub fn add_tag(&mut self, name: &str) -> std::io::Result<()> {
        insert_sorted(&mut self.tags, name);
        self.save()
    }

    /// update applies `f` to a session's metadata, folds any newly-used
    /// category/tags into the option universe, and persists the store atomically.
    pub fn update<F: FnOnce(&mut SessionMeta)>(&mut self, id: &str, f: F) -> std::io::Result<()> {
        self.apply_update(id, f);
        self.save()
    }

    /// update_many applies `f` to several sessions and persists once. Used for
    /// bulk actions (archive all, move to category).
    pub fn update_many<F: Fn(&mut SessionMeta)>(
        &mut self,
        ids: &[String],
        f: F,
    ) -> std::io::Result<()> {
        for id in ids {
            self.apply_update(id, &f);
        }
        self.save()
    }

    /// apply_update mutates one session's metadata and registers its options.
    fn apply_update<F: FnOnce(&mut SessionMeta)>(&mut self, id: &str, f: F) {
        let mut meta = self.data.get(id).cloned().unwrap_or_default();
        f(&mut meta);
        if !meta.category.is_empty() {
            insert_sorted(&mut self.categories, &meta.category);
        }
        for t in &meta.tags {
            insert_sorted(&mut self.tags, t);
        }
        self.data.insert(id.to_string(), meta);
    }

    /// set_pinned pins a session, or unpins it. Unpinning removes it from the
    /// curated dashboard, so its dashboard organization (category, tags,
    /// archived) is dropped.
    pub fn set_pinned(&mut self, id: &str, pinned: bool) -> std::io::Result<()> {
        self.update(id, pin_fn(pinned))
    }

    /// set_pinned_many applies set_pinned to several sessions in one persisted
    /// write.
    pub fn set_pinned_many(&mut self, ids: &[String], pinned: bool) -> std::io::Result<()> {
        self.update_many(ids, pin_fn(pinned))
    }

    /// apply overlays stored metadata onto a session. A non-empty title override
    /// replaces the transcript-derived title; the derived title stays the fallback.
    pub fn apply(&self, s: &mut Session) {
        let meta = self.get(&s.id);
        s.pinned = meta.pinned;
        s.category = meta.category;
        s.tags = meta.tags;
        s.archived = meta.archived;
        if let Some(title) = meta.title.filter(|t| !t.is_empty()) {
            s.title = title;
        }
    }

    /// seed_options_from_usage folds every category/tag currently assigned to a
    /// session into the option universe.
    fn seed_options_from_usage(&mut self) {
        let used: Vec<(String, Vec<String>)> = self
            .data
            .values()
            .map(|m| (m.category.clone(), m.tags.clone()))
            .collect();
        for (cat, tags) in used {
            if !cat.is_empty() {
                insert_sorted(&mut self.categories, &cat);
            }
            for t in &tags {
                insert_sorted(&mut self.tags, t);
            }
        }
    }

    /// save writes the store via a temp file + rename so a crash never leaves a
    /// half-written sidecar.
    fn save(&self) -> std::io::Result<()> {
        if let Some(parent) = self.path.parent() {
            fs::create_dir_all(parent)?;
        }
        let doc = MetaDoc {
            sessions: self.data.clone(),
            categories: self.categories.clone(),
            tags: self.tags.clone(),
        };
        let b = serde_json::to_vec_pretty(&doc)
            .map_err(|e| std::io::Error::new(std::io::ErrorKind::InvalidData, e))?;
        let mut tmp = self.path.clone().into_os_string();
        tmp.push(".tmp");
        let tmp = PathBuf::from(tmp);
        fs::write(&tmp, &b)?;
        fs::rename(&tmp, &self.path)
    }
}

/// pin_fn returns a mutator that pins (or unpins, clearing organization) a
/// session's metadata.
fn pin_fn(pinned: bool) -> impl Fn(&mut SessionMeta) {
    move |meta: &mut SessionMeta| {
        meta.pinned = pinned;
        if !pinned {
            meta.category = String::new();
            meta.tags = Vec::new();
            meta.archived = false;
        }
    }
}

/// insert_sorted adds `name` to a sorted vec if absent, keeping it sorted.
/// An empty name is a no-op, matching the Go option universe.
fn insert_sorted(xs: &mut Vec<String>, name: &str) {
    if name.is_empty() {
        return;
    }
    match xs.binary_search_by(|x| x.as_str().cmp(name)) {
        Ok(_) => {}
        Err(i) => xs.insert(i, name.to_string()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::session::{Kind, Status};
    use std::sync::Mutex;
    use std::time::{SystemTime, UNIX_EPOCH};

    // insert_sorted guards dedup, ordering, and the empty-name no-op the option
    // universe relies on.
    #[test]
    fn insert_sorted_dedups_and_orders() {
        let mut xs = Vec::new();
        insert_sorted(&mut xs, "work");
        insert_sorted(&mut xs, "admin");
        insert_sorted(&mut xs, "work"); // duplicate ignored
        insert_sorted(&mut xs, ""); // empty ignored
        assert_eq!(xs, vec!["admin".to_string(), "work".to_string()]);
    }

    // CLAUDE_CONFIG_DIR is process-global, so env-touching tests serialize on
    // this lock and always point at a unique temp dir — never the real ~/.claude.
    static ENV_LOCK: Mutex<()> = Mutex::new(());

    /// with_temp_config runs `f` with CLAUDE_CONFIG_DIR pointed at a fresh temp
    /// dir, restoring the prior value afterwards. The lock keeps concurrent tests
    /// from racing on the shared env var.
    fn with_temp_config(f: impl FnOnce(&std::path::Path)) {
        let _guard = ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner());
        let nanos = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        let dir = std::env::temp_dir().join(format!("ccs-meta-test-{nanos}"));
        fs::create_dir_all(&dir).unwrap();
        let prev = std::env::var_os("CLAUDE_CONFIG_DIR");
        std::env::set_var("CLAUDE_CONFIG_DIR", &dir);
        f(&dir);
        match prev {
            Some(v) => std::env::set_var("CLAUDE_CONFIG_DIR", v),
            None => std::env::remove_var("CLAUDE_CONFIG_DIR"),
        }
        let _ = fs::remove_dir_all(&dir);
    }

    fn session_with_derived_title(id: &str, derived: &str) -> Session {
        Session {
            id: id.to_string(),
            path: String::new(),
            project_dir: String::new(),
            cwd: String::new(),
            git_branch: String::new(),
            title: derived.to_string(),
            last_message: String::new(),
            kind: Kind::Main,
            status: Status::Inactive,
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

    // The title override drives the whole rename feature: set → displayed title
    // is the override; clear → reverts to derived; an archived-only patch and an
    // unpin must both leave the override intact (rename is independent of both).
    #[test]
    fn title_override_set_clear_and_survives_archive_and_unpin() {
        with_temp_config(|_dir| {
            let id = "sess-1";

            // No override: apply leaves the derived title.
            let mut store = MetaStore::load().unwrap();
            store.set_pinned(id, true).unwrap();
            let mut s = session_with_derived_title(id, "derived title");
            MetaStore::load().unwrap().apply(&mut s);
            assert_eq!(s.title, "derived title");

            // Set the override: apply now shows it.
            store
                .update(id, |m| m.title = Some("custom name".into()))
                .unwrap();
            let mut s = session_with_derived_title(id, "derived title");
            MetaStore::load().unwrap().apply(&mut s);
            assert_eq!(s.title, "custom name");

            // An archived-only mutation must not disturb the override.
            store.update(id, |m| m.archived = true).unwrap();
            let mut s = session_with_derived_title(id, "derived title");
            MetaStore::load().unwrap().apply(&mut s);
            assert_eq!(s.title, "custom name");
            assert!(s.archived);

            // Unpin clears category/tags/archived but preserves the rename.
            store.set_pinned(id, false).unwrap();
            let mut s = session_with_derived_title(id, "derived title");
            MetaStore::load().unwrap().apply(&mut s);
            assert_eq!(s.title, "custom name");
            assert!(!s.pinned);
            assert!(!s.archived);

            // Clearing the override (empty → None) reverts to the derived title.
            store.update(id, |m| m.title = None).unwrap();
            let mut s = session_with_derived_title(id, "derived title");
            MetaStore::load().unwrap().apply(&mut s);
            assert_eq!(s.title, "derived title");
        });
    }
}
