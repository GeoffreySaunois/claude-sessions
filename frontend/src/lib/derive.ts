import type { Session } from "./types";

// A curated ~8-color token palette. Each entry names a CSS variable holding a
// hue that has light- and dark-theme values defined in app.css, so a chip reads
// well on both themes. The order is fixed: changing it would reshuffle which
// name maps to which color.
export const TOKEN_PALETTE = [
  "blue",
  "green",
  "violet",
  "amber",
  "cyan",
  "rose",
  "teal",
  "slate",
] as const;

export type Token = (typeof TOKEN_PALETTE)[number];

// fnv1a is a tiny deterministic string hash. We only need a stable, well-mixed
// integer to index the palette — not cryptographic strength.
function fnv1a(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

// tokenFor maps a category/tag name deterministically onto a palette token, so
// the same name is always the same color across sessions and reloads.
export function tokenFor(name: string): Token {
  return TOKEN_PALETTE[fnv1a(name) % TOKEN_PALETTE.length]!;
}

// tokenVar is the CSS variable a chip should use as its hue. Style rules then
// derive the tinted background and border from it via color-mix.
export function tokenVar(name: string): string {
  return `var(--tok-${tokenFor(name)})`;
}

export function projBase(cwd: string): string {
  if (!cwd) return "(no cwd)";
  const parts = cwd.split("/").filter(Boolean);
  return parts.length ? parts[parts.length - 1]! : cwd;
}

/** Abbreviate a home-directory path for display: /Users/foo/x -> ~/x */
export function tildePath(cwd: string): string {
  if (!cwd) return "";
  return cwd.replace(/^\/(?:Users|home)\/[^/]+\//, "~/");
}

export function relTime(iso: string): string {
  const t = new Date(iso).getTime();
  if (!t) return "—";
  const s = Math.round((Date.now() - t) / 1000);
  if (s < 45) return "just now";
  const m = Math.round(s / 60);
  if (m < 60) return m + "m ago";
  const h = Math.round(m / 60);
  if (h < 24) return h + "h ago";
  const d = Math.round(h / 24);
  if (d < 30) return d + "d ago";
  return new Date(t).toLocaleDateString();
}

// A session matches the quick filter if every lowercased term is found in its
// metadata haystack, OR the server-side full-text search matched its content
// for the current query.
export function textMatch(
  s: Session,
  query: string,
  contentQuery: string,
  contentMatches: Map<string, string>,
): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  const hay = [
    s.title,
    projBase(s.cwd),
    s.cwd,
    (s.tags || []).join(" "),
    s.category,
    s.lastMessage,
  ]
    .join(" ")
    .toLowerCase();
  if (q.split(/\s+/).every((term) => hay.includes(term))) return true;
  return contentQuery === q && contentMatches.has(s.id);
}

// Splits text into segments for highlighting: each segment is either plain text
// or a matched term to wrap in <mark>.
export interface HlSegment {
  text: string;
  hl: boolean;
}

export function highlightSegments(text: string, terms: string[]): HlSegment[] {
  const escaped = terms
    .map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
    .filter(Boolean);
  if (!escaped.length) return [{ text, hl: false }];
  const re = new RegExp("(" + escaped.join("|") + ")", "ig");
  const out: HlSegment[] = [];
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) out.push({ text: text.slice(last, m.index), hl: false });
    out.push({ text: m[0], hl: true });
    last = m.index + m[0].length;
    if (m.index === re.lastIndex) re.lastIndex++;
  }
  if (last < text.length) out.push({ text: text.slice(last), hl: false });
  return out;
}
