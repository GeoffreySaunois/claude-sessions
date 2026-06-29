<script lang="ts">
  import { store } from "./store.svelte";
  import { relTime, tildePath } from "./derive";
  import StatusBadge from "./StatusBadge.svelte";
  import TitleCell from "./TitleCell.svelte";
  import CategoryControl from "./CategoryControl.svelte";
  import TagsControl from "./TagsControl.svelte";
  import type { Session, SessionStatus } from "./types";

  let { session }: { session: Session } = $props();

  const selected = $derived(store.selected.has(session.id));
  const focused = $derived(store.focusedSession?.id === session.id);
  const live = $derived(session.status === "busy");

  let rowEl = $state<HTMLElement>();
  // Scroll the keyboard-focused row into view when focus lands on it.
  $effect(() => {
    if (focused) rowEl?.scrollIntoView({ block: "nearest" });
  });

  // The left-rail status glyph: filled dot when busy, hollow circle when
  // waiting, a faint mid-dot when inactive.
  const GLYPH: Record<SessionStatus, string> = {
    busy: "●",
    waiting: "○",
    inactive: "·",
  };
</script>

<article
  bind:this={rowEl}
  class="row"
  class:selected
  class:kfocus={focused}
  class:archived={session.archived}
  oncontextmenu={(e) => {
    e.preventDefault();
    store.openContextMenu(session, "main", e.clientX, e.clientY);
  }}
>
  <label class="row-pick">
    <input
      type="checkbox"
      checked={selected}
      aria-label="Select session"
      onchange={(e) =>
        store.toggleMainSelect(session.id, e.currentTarget.checked)}
    />
  </label>

  <span class="status-glyph {session.status}" aria-hidden="true"
    >{GLYPH[session.status]}</span
  >

  <div class="row-body">
    <TitleCell {session} renderingQuery={store.filter} view="main" editable>
      {#snippet trailing()}
        <StatusBadge status={session.status} />
      {/snippet}
    </TitleCell>
    <div class="meta">
      <span class="cwd" title={session.cwd}>{tildePath(session.cwd)}</span>
      {#if session.gitBranch}
        <span class="sep">│</span>
        <span class="branch">{session.gitBranch}</span>
      {/if}
    </div>
  </div>

  <div class="row-chips">
    <CategoryControl {session} />
    <TagsControl {session} />
  </div>

  <div class="row-when" class:live title={session.lastActive}>
    {relTime(session.lastActive)}
  </div>
</article>
