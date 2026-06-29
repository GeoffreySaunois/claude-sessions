<script lang="ts">
  import { store, type GroupMode } from "./lib/store.svelte";
  import { theme } from "./lib/theme.svelte";
  import GroupBlock from "./lib/GroupBlock.svelte";
  import MainActionBar from "./lib/MainActionBar.svelte";
  import BrowseModal from "./lib/BrowseModal.svelte";
  import Toast from "./lib/Toast.svelte";

  const counts = $derived(store.counts);
  const anyPinned = $derived(store.pinned.length > 0);
  const groups = $derived(store.mainGroups);
  const hasVisible = $derived(store.visibleMain.length > 0);

  const GROUP_MODES: { id: GroupMode; label: string }[] = [
    { id: "project", label: "Group: Project" },
    { id: "category", label: "Category" },
    { id: "none", label: "None" },
  ];

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && store.browseOpen && !store.popoverOpen) {
      store.closeBrowse();
    }
  }

  $effect(() => store.startRefreshLoop());
</script>

<svelte:window onkeydown={onKeydown} />

<header>
  <div class="titlebar">
    <h1>Claude Sessions</h1>
    <span class="sub">{store.updatedAt ? "updated " + store.updatedAt : "—"}</span>
    <div class="counts">
      {#each [["busy", counts.busy], ["waiting", counts.waiting], ["inactive", counts.inactive]] as [kind, n] (kind)}
        <span class="count"
          ><span class="dot {kind}"></span><b>{n}</b>{kind}</span
        >
      {/each}
    </div>
    <button class="browseBtn" type="button" onclick={() => store.openBrowse()}
      >Browse all</button
    >
    <button
      id="themeBtn"
      type="button"
      title={"Theme: " + theme.meta.label + " (click to cycle)"}
      onclick={() => theme.cycle()}
    >
      <span class="ic">{theme.meta.ic}</span><span>{theme.meta.label}</span>
    </button>
  </div>

  <div class="controls">
    <input
      type="text"
      class="searchbox"
      placeholder="Search pinned — title, project, tags, message text…"
      autocomplete="off"
      spellcheck="false"
      bind:value={store.filter}
      oninput={() => store.scheduleContentSearch(store.filter)}
    />
    <div class="seg">
      {#each GROUP_MODES as g (g.id)}
        <button
          class:active={store.group === g.id}
          onclick={() => (store.group = g.id)}>{g.label}</button
        >
      {/each}
    </div>
    <span class="ctl"
      ><span class="lbl">Status</span>
      <select class="filter" bind:value={store.statusFilter}>
        <option value="">all</option>
        <option value="busy">busy</option>
        <option value="waiting">waiting</option>
        <option value="inactive">inactive</option>
      </select></span
    >
    <span class="ctl"
      ><span class="lbl">Category</span>
      <select class="filter" bind:value={store.categoryFilter}>
        <option value="">all</option>
        {#each store.options.categories as c (c)}
          <option value={c}>{c}</option>
        {/each}
      </select></span
    >
    <span class="ctl"
      ><span class="lbl">Tag</span>
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
    <button class="linkbtn" onclick={() => store.expandAll()}>expand all</button>
    <button class="linkbtn" onclick={() => store.collapseAll()}>collapse all</button>
  </div>
</header>

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

<MainActionBar />
<BrowseModal />
<Toast />
