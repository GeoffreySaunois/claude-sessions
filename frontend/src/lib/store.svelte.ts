import {
  fetchOptions,
  fetchSearch,
  fetchSessions,
  postBulk,
  postMeta,
  postOpen,
  postPin,
} from "./api";
import { projBase, textMatch } from "./derive";
import type { BulkAction, Options, Session } from "./types";

export const ALL_KINDS: Session["kind"][] = [
  "main",
  "worktree",
  "background",
  "sdk",
  "example",
  "gym",
];
const DEFAULT_KINDS: Session["kind"][] = ["main", "worktree", "background"];

const LS = {
  kinds: "ccs-browse-kinds",
  collapsed: "ccs-collapsed",
};

export type GroupMode = "project" | "category" | "none";

function loadStringSet(key: string, fallback: string[]): Set<string> {
  try {
    const raw = localStorage.getItem(key);
    if (raw) {
      const a = JSON.parse(raw);
      if (Array.isArray(a)) return new Set(a);
    }
  } catch (e) {
    /* ignore */
  }
  return new Set(fallback);
}

function saveStringSet(key: string, set: Set<string>): void {
  try {
    localStorage.setItem(key, JSON.stringify([...set]));
  } catch (e) {
    /* ignore */
  }
}

// Toast notification surfaced by the App shell.
export interface Toast {
  msg: string;
  err: boolean;
  seq: number;
}

class SessionStore {
  sessions = $state<Session[]>([]);
  options = $state<Options>({ categories: [], tags: [] });

  // ---- Main (pinned) view ----
  selected = $state<Set<string>>(new Set());
  filter = $state("");
  group = $state<GroupMode>("project");
  showArchived = $state(false);
  statusFilter = $state("");
  categoryFilter = $state("");
  tagFilter = $state("");
  collapsed = $state<Set<string>>(loadStringSet(LS.collapsed, []));
  expandedPreviews = $state<Set<string>>(new Set());

  // ---- Browse-all modal ----
  browseOpen = $state(false);
  browseSelected = $state<Set<string>>(new Set());
  browseFilter = $state("");
  browseStatusFilter = $state("");
  browseProjectFilter = $state("");
  browseKinds = $state<Set<string>>(loadStringSet(LS.kinds, DEFAULT_KINDS));

  // ---- shared ----
  // While a popover is up we suppress the refresh re-render so the user is not
  // interrupted mid-edit.
  popoverOpen = $state(false);
  contentQuery = $state("");
  contentMatches = $state<Map<string, string>>(new Map());
  updatedAt = $state("");
  toast = $state<Toast | null>(null);

  // renderingQuery decides which view's search box drove the highlighted
  // snippet; a content snippet only shows in the view whose query produced it.
  private toastSeq = 0;
  private searchTimer: ReturnType<typeof setTimeout> | null = null;

  // ---- derived: Main view ----
  get pinned(): Session[] {
    return this.sessions.filter((s) => s.pinned);
  }

  private passesMainFilters(s: Session): boolean {
    if (!this.showArchived && s.archived) return false;
    if (this.statusFilter && s.status !== this.statusFilter) return false;
    if (this.categoryFilter && (s.category || "") !== this.categoryFilter)
      return false;
    if (this.tagFilter && !(s.tags || []).includes(this.tagFilter)) return false;
    return textMatch(s, this.filter, this.contentQuery, this.contentMatches);
  }

  get visibleMain(): Session[] {
    return this.pinned.filter((s) => this.passesMainFilters(s));
  }

  private groupKey(s: Session): string {
    if (this.group === "project") return projBase(s.cwd);
    if (this.group === "category") return s.category || "(uncategorized)";
    return "";
  }

  get mainGroups(): [string, Session[]][] {
    const sessions = this.visibleMain;
    if (this.group === "none") return [["", sessions]];
    const map = new Map<string, Session[]>();
    for (const s of sessions) {
      const k = this.groupKey(s);
      if (!map.has(k)) map.set(k, []);
      map.get(k)!.push(s);
    }
    return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  }

  // ---- derived: Browse view ----
  private passesBrowseFilters(s: Session): boolean {
    if (!this.browseKinds.has(s.kind)) return false;
    if (this.browseStatusFilter && s.status !== this.browseStatusFilter)
      return false;
    if (
      this.browseProjectFilter &&
      projBase(s.cwd) !== this.browseProjectFilter
    )
      return false;
    return textMatch(
      s,
      this.browseFilter,
      this.contentQuery,
      this.contentMatches,
    );
  }

  get visibleBrowse(): Session[] {
    return this.sessions
      .filter((s) => this.passesBrowseFilters(s))
      .sort(
        (a, b) =>
          new Date(b.lastActive).getTime() - new Date(a.lastActive).getTime(),
      );
  }

  get distinctProjects(): string[] {
    const set = new Set(this.sessions.map((s) => projBase(s.cwd)));
    return [...set].sort((a, b) => a.localeCompare(b));
  }

  get counts(): { busy: number; waiting: number; inactive: number } {
    const c = { busy: 0, waiting: 0, inactive: 0 };
    for (const s of this.pinned) {
      if (s.archived && !this.showArchived) continue;
      c[s.status]++;
    }
    return c;
  }

  // ---- toast ----
  notify(msg: string, err = false): void {
    this.toast = { msg, err, seq: ++this.toastSeq };
  }

  // ---- full-text content search (debounced) ----
  scheduleContentSearch(query: string): void {
    if (this.searchTimer) clearTimeout(this.searchTimer);
    if (query.trim().length < 2) {
      this.contentQuery = "";
      this.contentMatches = new Map();
      return;
    }
    this.searchTimer = setTimeout(() => this.runContentSearch(query.trim()), 280);
  }

  private async runContentSearch(query: string): Promise<void> {
    try {
      const data = await fetchSearch(query);
      this.contentQuery = query.toLowerCase();
      this.contentMatches = new Map(Object.entries(data.matches));
    } catch (e) {
      /* leave prior results in place on a failed search */
    }
  }

  // ---- refresh loop ----
  async refresh(): Promise<void> {
    // Don't clobber an open popover — the user may be mid-edit.
    if (this.popoverOpen) return;
    try {
      const [sessions, options] = await Promise.all([
        fetchSessions(),
        fetchOptions(),
      ]);
      this.sessions = sessions;
      this.options = options;
      this.pruneSelection();
      this.pruneBrowseSelection();
      this.updatedAt = new Date().toLocaleTimeString();
    } catch (e) {
      this.notify("Refresh failed: " + (e as Error).message, true);
    }
  }

  startRefreshLoop(): () => void {
    void this.refresh();
    const id = setInterval(() => void this.refresh(), 4000);
    return () => clearInterval(id);
  }

  private pruneSelection(): void {
    const ids = new Set(this.sessions.map((s) => s.id));
    const next = new Set([...this.selected].filter((id) => ids.has(id)));
    this.selected = next;
  }
  pruneBrowseSelection(): void {
    const ids = new Set(this.sessions.map((s) => s.id));
    const next = new Set([...this.browseSelected].filter((id) => ids.has(id)));
    this.browseSelected = next;
  }

  // ---- selection mutations ----
  toggleMainSelect(id: string, on: boolean): void {
    const next = new Set(this.selected);
    if (on) next.add(id);
    else next.delete(id);
    this.selected = next;
  }
  toggleBrowseSelect(id: string, on: boolean): void {
    const next = new Set(this.browseSelected);
    if (on) next.add(id);
    else next.delete(id);
    this.browseSelected = next;
  }
  setGroupSelected(sessions: Session[], on: boolean): void {
    const next = new Set(this.selected);
    for (const s of sessions) {
      if (on) next.add(s.id);
      else next.delete(s.id);
    }
    this.selected = next;
  }
  clearMainSelection(): void {
    this.selected = new Set();
  }
  clearBrowseSelection(): void {
    this.browseSelected = new Set();
  }

  togglePreview(id: string): void {
    const next = new Set(this.expandedPreviews);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    this.expandedPreviews = next;
  }

  // ---- collapse ----
  toggleCollapsed(name: string): void {
    const next = new Set(this.collapsed);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    this.collapsed = next;
    saveStringSet(LS.collapsed, this.collapsed);
  }
  collapseAll(): void {
    const next = new Set(this.collapsed);
    for (const [name] of this.mainGroups) if (name) next.add(name);
    this.collapsed = next;
    saveStringSet(LS.collapsed, this.collapsed);
  }
  expandAll(): void {
    this.collapsed = new Set();
    saveStringSet(LS.collapsed, this.collapsed);
  }

  // ---- kind chips ----
  toggleKind(kind: string): void {
    const next = new Set(this.browseKinds);
    if (next.has(kind)) next.delete(kind);
    else next.add(kind);
    this.browseKinds = next;
    saveStringSet(LS.kinds, this.browseKinds);
    this.pruneBrowseSelection();
  }

  // ---- option universe ----
  private registerOption(kind: "categories" | "tags", value: string): void {
    if (value && !this.options[kind].includes(value)) {
      this.options = {
        ...this.options,
        [kind]: [...this.options[kind], value].sort((a, b) =>
          a.localeCompare(b),
        ),
      };
    }
  }

  // ---- pin / unpin ----
  async setPinned(s: Session, pinned: boolean): Promise<void> {
    // Optimistic update so both surfaces react immediately. Unpin clears the
    // curated metadata, mirroring the server-side behavior.
    s.pinned = pinned;
    if (!pinned) {
      s.category = "";
      s.tags = [];
      s.archived = false;
    }
    this.selected.delete(s.id);
    this.selected = new Set(this.selected);
    try {
      await postPin(s.id, pinned);
      this.notify(pinned ? "Pinned" : "Removed from dashboard");
    } catch (e) {
      this.notify(
        (pinned ? "Pin" : "Remove") + " failed: " + (e as Error).message,
        true,
      );
    }
  }

  // ---- per-session meta (category / tags / archived) ----
  async commitMeta(
    s: Session,
    patch: { category?: string; tags?: string[]; archived?: boolean },
  ): Promise<void> {
    if (patch.category !== undefined) s.category = patch.category;
    if (patch.tags !== undefined) s.tags = patch.tags;
    if (patch.archived !== undefined) s.archived = patch.archived;
    this.sessions = [...this.sessions];
    const body = { id: s.id, ...patch };
    try {
      await postMeta(body);
      if (patch.category) this.registerOption("categories", patch.category);
      if (patch.tags) for (const t of patch.tags) this.registerOption("tags", t);
      this.notify("Saved");
    } catch (e) {
      this.notify("Save failed: " + (e as Error).message, true);
    }
  }

  setCategory(s: Session, value: string): Promise<void> {
    return this.commitMeta(s, { category: value });
  }
  addTag(s: Session, tag: string): Promise<void> {
    const cur = new Set(s.tags || []);
    cur.add(tag);
    return this.commitMeta(s, { tags: [...cur] });
  }
  removeTag(s: Session, tag: string): Promise<void> {
    return this.commitMeta(s, { tags: (s.tags || []).filter((t) => t !== tag) });
  }
  toggleTag(s: Session, tag: string): Promise<void> {
    const cur = new Set(s.tags || []);
    if (cur.has(tag)) cur.delete(tag);
    else cur.add(tag);
    return this.commitMeta(s, { tags: [...cur] });
  }

  // ---- bulk actions ----
  private bulkToast(action: BulkAction, n: number): string {
    const plural = n === 1 ? "session" : "sessions";
    if (action === "unpin") return "Removed " + n + " " + plural + " from dashboard";
    if (action === "pin") return "Pinned " + n + " " + plural;
    return "Updated " + n + " " + plural;
  }

  async runMainBulk(action: BulkAction, value?: string): Promise<void> {
    const ids = [...this.selected];
    if (!ids.length) return;
    try {
      const res = await postBulk({ ids, action, value: value || "" });
      if (value) this.registerOption("categories", value);
      this.notify(this.bulkToast(action, res.updated));
      await this.refresh();
    } catch (e) {
      this.notify("Bulk action failed: " + (e as Error).message, true);
    }
  }

  async runBrowseBulk(action: BulkAction): Promise<void> {
    const ids = [...this.browseSelected];
    if (!ids.length) return;
    try {
      const res = await postBulk({ ids, action, value: "" });
      this.notify(this.bulkToast(action, res.updated));
      await this.refresh();
    } catch (e) {
      this.notify("Bulk action failed: " + (e as Error).message, true);
    }
  }

  // ---- open ----
  async openIds(ids: string[]): Promise<void> {
    if (!ids.length) return;
    try {
      const res = await postOpen(ids);
      this.notify("Opened " + res.opened + " session" + (res.opened === 1 ? "" : "s"));
    } catch (e) {
      this.notify("Open failed: " + (e as Error).message, true);
    }
  }

  // ---- modal open/close ----
  openBrowse(): void {
    this.browseOpen = true;
  }
  closeBrowse(): void {
    this.browseOpen = false;
  }
}

export const store = new SessionStore();
