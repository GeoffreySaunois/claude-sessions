import type {
  BulkAction,
  MetaPatch,
  Options,
  SearchResult,
  Session,
  Transcript,
} from "./types";

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(url + " " + res.status);
  return res.json() as Promise<T>;
}

export async function fetchSessions(): Promise<Session[]> {
  const res = await fetch("/api/sessions");
  if (!res.ok) throw new Error("sessions " + res.status);
  return res.json() as Promise<Session[]>;
}

export async function fetchOptions(): Promise<Options> {
  const res = await fetch("/api/options");
  if (!res.ok) throw new Error("options " + res.status);
  const o = (await res.json()) as Partial<Options>;
  // The API encodes empty option universes as JSON null; normalize to arrays.
  return { categories: o.categories || [], tags: o.tags || [] };
}

export async function fetchTranscript(
  id: string,
  limit: number,
): Promise<Transcript> {
  const res = await fetch(
    "/api/sessions/" + encodeURIComponent(id) + "/transcript?limit=" + limit,
  );
  if (!res.ok) throw new Error("transcript " + res.status);
  const data = (await res.json()) as Partial<Transcript>;
  return { messages: data.messages || [] };
}

export async function fetchSearch(query: string): Promise<SearchResult> {
  const res = await fetch("/api/search?q=" + encodeURIComponent(query));
  if (!res.ok) throw new Error("search " + res.status);
  const data = (await res.json()) as Partial<SearchResult>;
  return { matches: data.matches || {} };
}

export function postMeta(patch: MetaPatch): Promise<{ ok: true }> {
  return postJSON("/api/meta", patch);
}

export function postPin(id: string, pinned: boolean): Promise<{ ok: true }> {
  return postJSON("/api/pin", { id, pinned });
}

export function postBulk(body: {
  ids: string[];
  action: BulkAction;
  value?: string;
}): Promise<{ updated: number }> {
  return postJSON("/api/bulk", body);
}

export function postOpen(ids: string[]): Promise<{ opened: number }> {
  return postJSON("/api/open", { ids });
}
