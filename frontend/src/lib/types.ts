// Mirrors the HTTP API contract in MIGRATION.md exactly.

export type SessionKind =
  | "main"
  | "worktree"
  | "example"
  | "gym"
  | "sdk"
  | "background";

export type SessionStatus = "busy" | "waiting" | "inactive";

export interface Session {
  id: string;
  path: string;
  projectDir: string;
  cwd: string;
  gitBranch: string;
  title: string;
  lastMessage: string;
  kind: SessionKind;
  status: SessionStatus;
  pid: number;
  lastActive: string; // RFC3339
  sizeBytes: number;
  version: string;
  pinned: boolean;
  category: string;
  tags: string[];
  archived: boolean;
}

export interface Options {
  categories: string[];
  tags: string[];
}

export interface SearchResult {
  matches: Record<string, string>;
}

export type BulkAction = "archive" | "unarchive" | "category" | "pin" | "unpin";

export interface MetaPatch {
  id: string;
  pinned?: boolean;
  category?: string;
  tags?: string[];
  archived?: boolean;
}
