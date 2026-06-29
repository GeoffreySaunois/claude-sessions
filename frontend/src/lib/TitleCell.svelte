<script lang="ts">
  import { store } from "./store.svelte";
  import { highlightSegments } from "./derive";
  import { onActivateKey } from "./clickable";
  import type { Session } from "./types";
  import type { Snippet } from "svelte";

  // renderingQuery is the active search box's query for THIS view, so a content
  // snippet only shows in the surface whose query produced it. editable turns on
  // the inline rename affordance (Main rows only); Browse rows stay read-only.
  // `view` scopes the preview-expand state so the same session toggles
  // independently in Main vs Browse. `trailing` is an optional snippet rendered
  // on the title line after the title (the card rows use it for the status pill).
  let {
    session,
    renderingQuery,
    view,
    editable = false,
    trailing,
  }: {
    session: Session;
    renderingQuery: string;
    view: "main" | "browse";
    editable?: boolean;
    trailing?: Snippet;
  } = $props();

  const open = $derived(store.isPreviewOpen(view, session.id));

  // Inline rename state. `draft` seeds from the current title when editing opens.
  let editing = $state(false);
  let draft = $state("");
  let inputEl = $state<HTMLInputElement>();

  function startEdit() {
    if (!editable) return;
    draft = session.title || "";
    editing = true;
    // Suppress the refresh re-render while editing, like the popover controls.
    store.popoverOpen = true;
  }

  function stopEdit() {
    editing = false;
    store.popoverOpen = false;
  }

  function commit() {
    if (!editing) return;
    void store.commitTitle(session, draft);
    stopEdit();
  }

  function cancel() {
    stopEdit();
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      commit();
    } else if (e.key === "Escape") {
      e.preventDefault();
      cancel();
    }
  }

  $effect(() => {
    if (editing) inputEl?.focus();
  });

  // The context menu's Rename action bumps store.renameRequest with this id;
  // honoring it opens the inline editor (Main rows only, which are editable).
  $effect(() => {
    if (editable && store.renameRequest === session.id) {
      store.renameRequest = null;
      startEdit();
    }
  });

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
  <div class="title-line">
    <div class="titlerow">
      {#if editing}
        <input
          class="title-edit"
          type="text"
          bind:this={inputEl}
          bind:value={draft}
          onkeydown={onKeydown}
          onblur={commit}
          aria-label="Rename session"
        />
      {:else}
        {#if session.title && session.title.trim()}
          <span class="title">{session.title}</span>
        {:else}
          <span class="title"><span class="muted">untitled</span></span>
        {/if}
        {#if session.kind && session.kind !== "main"}
          <span class="kindbadge">{session.kind}</span>
        {/if}
        {#if editable}
          <button
            class="title-edit-btn"
            type="button"
            title="Rename"
            aria-label="Rename session"
            onclick={startEdit}>✎</button
          >
        {/if}
      {/if}
    </div>
    {#if trailing}{@render trailing()}{/if}
  </div>

  {#if session.lastMessage && session.lastMessage.trim()}
    <div class="preview" class:open>
      <span
        class="pcaret"
        role="button"
        tabindex="0"
        onkeydown={onActivateKey}
        onclick={() => store.togglePreview(view, session.id)}>▶</span
      >
      <span
        class="ptext"
        title="click to expand"
        role="button"
        tabindex="0"
        onkeydown={onActivateKey}
        onclick={() => store.togglePreview(view, session.id)}>{session.lastMessage}</span
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
