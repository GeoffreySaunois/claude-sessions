<script lang="ts">
  import { store } from "./store.svelte";
  import { onActivateKey } from "./clickable";
  import Popover from "./Popover.svelte";

  // Right-click menu for a session row. The menu acts on the right-clicked row,
  // unless that row is part of the current multi-selection — then the
  // bulk-capable actions (Open, Archive, Unpin, Move to category) apply to the
  // whole selection. Built on the same fluo surface as Popover.

  const ctx = $derived(store.contextMenu);
  const session = $derived(ctx?.session ?? null);
  const isBrowse = $derived(ctx?.surface === "browse");

  // The selection on the active surface, and whether THIS row is part of it.
  const selection = $derived(isBrowse ? store.browseSelected : store.selected);
  const inSelection = $derived(!!session && selection.has(session.id));
  const bulkN = $derived(inSelection ? selection.size : 1);
  const bulkSuffix = $derived(bulkN > 1 ? ` (${bulkN})` : "");

  let menuEl = $state<HTMLDivElement>();
  let style = $state("");

  let catAnchor = $state<HTMLElement>();
  let catOpen = $state(false);

  // Clamp the menu to the viewport so it never overflows off-screen.
  function position() {
    if (!menuEl || !ctx) return;
    const r = menuEl.getBoundingClientRect();
    const x = Math.min(ctx.x, window.innerWidth - r.width - 8);
    const y = Math.min(ctx.y, window.innerHeight - r.height - 8);
    style = `top:${Math.max(8, y)}px;left:${Math.max(8, x)}px`;
  }

  $effect(() => {
    if (!ctx) return;
    position();
    const onOutside = (e: MouseEvent) => {
      if (menuEl && !menuEl.contains(e.target as Node)) store.closeContextMenu();
    };
    // Defer attach so the opening contextmenu event doesn't immediately close us.
    const t = setTimeout(() => {
      document.addEventListener("mousedown", onOutside, true);
    }, 0);
    return () => {
      clearTimeout(t);
      document.removeEventListener("mousedown", onOutside, true);
    };
  });

  // ---- actions ----
  // Bulk-capable: run over the whole selection when the row is part of it.
  function openTargets(): string[] {
    if (inSelection) return [...selection];
    return session ? [session.id] : [];
  }

  function close() {
    store.closeContextMenu();
  }

  function doOpen() {
    void store.openIds(openTargets());
    close();
  }
  // Pin is immediate; unpin (destructive) routes through the confirm dialog.
  function doPinToggle() {
    if (!session) return;
    if (session.pinned) doUnpin();
    else {
      void store.setPinned(session, true);
      close();
    }
  }
  function doArchiveToggle() {
    if (!session) return;
    const action = session.archived ? "unarchive" : "archive";
    if (inSelection) void store.runMainBulk(action);
    else void store.commitMeta(session, { archived: !session.archived });
    close();
  }
  function doUnpin() {
    if (!session) return;
    const ids = inSelection ? [...selection] : [session.id];
    store.requestUnpin(ids, isBrowse ? "browse" : "main");
    close();
  }
  function doRename() {
    if (session) store.requestRename(session.id);
    close();
  }
  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      store.notify("Copied");
    } catch (e) {
      store.notify("Copy failed: " + (e as Error).message, true);
    }
    close();
  }

  function onMoveCategory(e: MouseEvent) {
    catAnchor = e.currentTarget as HTMLElement;
    catOpen = true;
  }
  function pickCategory(value: string) {
    if (!session) return;
    if (inSelection) void store.runMainBulk("category", value);
    else void store.setCategory(session, value);
    catOpen = false;
    close();
  }
</script>

{#if ctx && session}
  <div class="ctxmenu" bind:this={menuEl} {style} role="menu" tabindex="-1">
    <div
      class="ctxitem"
      role="menuitem"
      tabindex="0"
      onclick={doOpen}
      onkeydown={onActivateKey}
    >
      <span class="cg">↗</span> Open{bulkSuffix}
    </div>
    <div
      class="ctxitem"
      role="menuitem"
      tabindex="0"
      onclick={doPinToggle}
      onkeydown={onActivateKey}
    >
      <span class="cg">{session.pinned ? "☆" : "★"}</span>
      {session.pinned ? "Unpin" : "Pin"}
    </div>

    {#if !isBrowse}
      <div
        class="ctxitem"
        role="menuitem"
        tabindex="0"
        onclick={doArchiveToggle}
        onkeydown={onActivateKey}
      >
        <span class="cg">{session.archived ? "⊞" : "⌗"}</span>
        {session.archived ? "Unarchive" : "Archive"}{inSelection ? bulkSuffix : ""}
      </div>
      <div
        class="ctxitem"
        role="menuitem"
        tabindex="0"
        bind:this={catAnchor}
        onclick={onMoveCategory}
        onkeydown={onActivateKey}
      >
        <span class="cg">⊕</span> Move to category…{inSelection ? bulkSuffix : ""}
      </div>
      <div
        class="ctxitem"
        role="menuitem"
        tabindex="0"
        onclick={doRename}
        onkeydown={onActivateKey}
      >
        <span class="cg">✎</span> Rename
      </div>
    {/if}

    {#if isBrowse}
      <div
        class="ctxitem"
        role="menuitem"
        tabindex="0"
        onclick={doUnpin}
        onkeydown={onActivateKey}
      >
        <span class="cg">⊘</span> Unpin{inSelection ? bulkSuffix : ""}
      </div>
    {/if}

    <div class="ctxdiv"></div>
    <div
      class="ctxitem"
      role="menuitem"
      tabindex="0"
      onclick={() => copy(session.cwd)}
      onkeydown={onActivateKey}
    >
      <span class="cg">⧉</span> Copy path
    </div>
    <div
      class="ctxitem"
      role="menuitem"
      tabindex="0"
      onclick={() => copy(session.id)}
      onkeydown={onActivateKey}
    >
      <span class="cg">#</span> Copy session id
    </div>
  </div>
{/if}

{#if catOpen && catAnchor}
  <Popover
    anchor={catAnchor}
    options={store.options.categories}
    isSelected={() => false}
    onpick={pickCategory}
    showClear
    clearLabel="✕ Clear category"
    emptyLabel="No categories"
    onclose={() => (catOpen = false)}
  />
{/if}
