<script lang="ts">
  import { store } from "./store.svelte";
  import { highlightSegments } from "./derive";
  import { onActivateKey } from "./clickable";
  import type { Session } from "./types";

  // renderingQuery is the active search box's query for THIS view, so a content
  // snippet only shows in the surface whose query produced it.
  let { session, renderingQuery }: { session: Session; renderingQuery: string } =
    $props();

  const open = $derived(store.expandedPreviews.has(session.id));

  // The content snippet shows only when the rendering query matches the
  // server-side search query that produced the current matches.
  const snippet = $derived.by(() => {
    const q = renderingQuery.trim().toLowerCase();
    if (!q || store.contentQuery !== q) return null;
    const snip = store.contentMatches.get(session.id);
    if (!snip) return null;
    const terms = q.split(/\s+/).filter(Boolean);
    return { raw: snip, segments: highlightSegments(snip, terms) };
  });
</script>

<div>
  <div>
    {#if session.title && session.title.trim()}
      <span class="title">{session.title}</span>
    {:else}
      <span class="title"><span class="muted">untitled</span></span>
    {/if}
    {#if session.kind && session.kind !== "main"}
      <span class="kindbadge">{session.kind}</span>
    {/if}
  </div>

  {#if session.lastMessage && session.lastMessage.trim()}
    <div class="preview" class:open>
      <span
        class="pcaret"
        role="button"
        tabindex="0"
        onkeydown={onActivateKey}
        onclick={() => store.togglePreview(session.id)}>▶</span
      >
      <span
        class="ptext"
        title="click to expand"
        role="button"
        tabindex="0"
        onkeydown={onActivateKey}
        onclick={() => store.togglePreview(session.id)}>{session.lastMessage}</span
      >
    </div>
  {/if}

  {#if snippet}
    <div class="search-hit">
      <span class="shicon">🔍</span>
      <span class="shtext" title={snippet.raw}
        >{#each snippet.segments as seg}{#if seg.hl}<mark class="hl"
              >{seg.text}</mark
            >{:else}{seg.text}{/if}{/each}</span
      >
    </div>
  {/if}
</div>
