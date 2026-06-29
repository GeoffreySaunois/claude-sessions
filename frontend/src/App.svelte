<script lang="ts">
  import { store, type GroupMode } from "./lib/store.svelte";
  import { theme, type ThemeMode } from "./lib/theme.svelte";
  import GroupBlock from "./lib/GroupBlock.svelte";
  import MainActionBar from "./lib/MainActionBar.svelte";
  import BrowseModal from "./lib/BrowseModal.svelte";
  import StatusBar from "./lib/StatusBar.svelte";
  import ContextMenu from "./lib/ContextMenu.svelte";
  import ShortcutsHelp from "./lib/ShortcutsHelp.svelte";
  import UnpinConfirm from "./lib/UnpinConfirm.svelte";
  import Toast from "./lib/Toast.svelte";
  import { handleKeydown } from "./lib/keyboard";

  const counts = $derived(store.counts);
  const anyPinned = $derived(store.pinned.length > 0);
  const groups = $derived(store.mainGroups);
  const hasVisible = $derived(store.visibleMain.length > 0);

  const GROUP_MODES: { id: GroupMode; label: string }[] = [
    { id: "project", label: "Folder" },
    { id: "category", label: "Category" },
    { id: "none", label: "None" },
  ];

  const THEME_MODES: { id: ThemeMode; label: string }[] = [
    { id: "system", label: "sys" },
    { id: "light", label: "light" },
    { id: "dark", label: "dark" },
  ];

  $effect(() => store.startRefreshLoop());

  // Keep the keyboard focus index inside the active list as it changes (filter,
  // group, modal open/close, refresh). Touching the reactive deps re-runs this.
  $effect(() => {
    void store.activeList.length;
    void store.browseOpen;
    store.clampFocus();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<header>
  <div class="titlebar">
    <div class="brand">
      <span class="prompt">❯</span>
      <h1>claude-sessions</h1>
      <span class="cursor" aria-hidden="true"></span>
    </div>

    <div class="counts">
      {#each [["busy", counts.busy], ["waiting", counts.waiting], ["inactive", counts.inactive]] as [kind, n] (kind)}
        <span class="count"><span class="dot {kind}"></span><b>{n}</b> {kind}</span>
      {/each}
    </div>

    <div class="header-right">
      <span class="pinned-badge"><span class="pin">📌</span> {store.pinned.length} pinned</span>
      <div id="themeBtn" role="group" aria-label="Theme">
        {#each THEME_MODES as t (t.id)}
          <button
            class="seg-opt"
            class:active={theme.mode === t.id}
            type="button"
            title={"Theme: " + t.label}
            onclick={() => theme.apply(t.id)}>{t.label}</button
          >
        {/each}
      </div>
      <button class="browseBtn" type="button" onclick={() => store.openBrowse()}
        >Browse all <span style="opacity:.7">→</span></button
      >
    </div>
  </div>

  <div class="controls">
    <div class="search-wrap">
      <span class="glyph">⌕</span>
      <input
        type="text"
        class="searchbox"
        placeholder="Search pinned — title, folder, tags, message text…"
        autocomplete="off"
        spellcheck="false"
        bind:value={store.filter}
        oninput={() => store.scheduleContentSearch(store.filter)}
      />
      <span class="kbd">⌘K</span>
    </div>

    <div class="seg">
      {#each GROUP_MODES as g (g.id)}
        <button
          class:active={store.group === g.id}
          onclick={() => (store.group = g.id)}>{g.label}</button
        >
      {/each}
    </div>

    <span class="ctl"
      ><span class="lbl">status</span>
      <select class="filter" bind:value={store.statusFilter}>
        <option value="">all</option>
        <option value="busy">busy</option>
        <option value="waiting">waiting</option>
        <option value="inactive">inactive</option>
      </select></span
    >
    <span class="ctl"
      ><span class="lbl">category</span>
      <select class="filter" bind:value={store.categoryFilter}>
        <option value="">all</option>
        {#each store.options.categories as c (c)}
          <option value={c}>{c}</option>
        {/each}
      </select></span
    >
    <span class="ctl"
      ><span class="lbl">tag</span>
      <select class="filter" bind:value={store.tagFilter}>
        <option value="">all</option>
        {#each store.options.tags as t (t)}
          <option value={t}>{t}</option>
        {/each}
      </select></span
    >
    <label class="toggle"
      ><input type="checkbox" bind:checked={store.showArchived} /> Show archived</label
    >
    <button class="linkbtn" onclick={() => store.expandAll()}>⇕ expand all</button>
    <button class="linkbtn" onclick={() => store.collapseAll()}>collapse all</button>
  </div>
</header>

<MainActionBar />

<main>
  {#if !anyPinned}
    <div class="empty-hero">
      <div class="glyph">★</div>
      <h2>Your pinned workspace is empty</h2>
      <p>
        Pin sessions to track them here. Browse all your sessions to find the
        ones worth keeping close.
      </p>
      <button class="btn" type="button" onclick={() => store.openBrowse()}
        >Browse all</button
      >
    </div>
  {:else if !hasVisible}
    <div class="empty">No pinned sessions match these filters.</div>
  {:else}
    {#each groups as [name, group] (name)}
      <GroupBlock {name} sessions={group} />
    {/each}
  {/if}
</main>

<BrowseModal />
<ContextMenu />
<ShortcutsHelp />
<UnpinConfirm />
<StatusBar />
<Toast />
