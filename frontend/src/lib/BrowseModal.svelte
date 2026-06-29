<script lang="ts">
  import { store, ALL_KINDS } from "./store.svelte";
  import BrowseRow from "./BrowseRow.svelte";
  import { onActivateKey } from "./clickable";

  const sessions = $derived(store.visibleBrowse);
  const sub = $derived(
    `${sessions.length} of ${store.sessions.length} shown · ${store.pinned.length} pinned`,
  );
  const n = $derived(store.browseSelected.size);

  function onOverlayDown(e: MouseEvent) {
    // Outside-click on the dim backdrop closes; clicks inside the modal don't.
    if (e.target === e.currentTarget) store.closeBrowse();
  }

  // Focus the search box when the modal opens.
  let searchEl = $state<HTMLInputElement>();
  $effect(() => {
    if (store.browseOpen) searchEl?.focus();
  });
</script>

<div
  class="overlay"
  class:show={store.browseOpen}
  onmousedown={onOverlayDown}
  role="presentation"
>
  <div class="modal" role="dialog" aria-modal="true" aria-label="Browse all sessions">
    <div class="modal-head">
      <h2>Browse all sessions</h2>
      <span class="modsub">{sub}</span>
      <button
        class="modal-close"
        type="button"
        title="Close (Esc)"
        aria-label="Close"
        onclick={() => store.closeBrowse()}>✕</button
      >
    </div>

    <div class="modal-controls">
      <input
        type="text"
        class="searchbox"
        placeholder="Search every session — incl. full conversation text…"
        autocomplete="off"
        spellcheck="false"
        bind:this={searchEl}
        bind:value={store.browseFilter}
        oninput={() => store.scheduleContentSearch(store.browseFilter)}
      />
      <span class="ctl"
        ><span class="lbl">Status</span>
        <select class="filter" bind:value={store.browseStatusFilter}>
          <option value="">all</option>
          <option value="busy">busy</option>
          <option value="waiting">waiting</option>
          <option value="inactive">inactive</option>
        </select></span
      >
      <span class="ctl"
        ><span class="lbl">Project</span>
        <select class="filter" bind:value={store.browseProjectFilter}>
          <option value="">all</option>
          {#each store.distinctProjects as p (p)}
            <option value={p}>{p}</option>
          {/each}
        </select></span
      >
      <span class="ctl"
        ><span class="lbl">Kinds</span>
        <span class="kindchips">
          {#each ALL_KINDS as k (k)}
            <span
              class="kindchip"
              class:on={store.browseKinds.has(k)}
              role="button"
              tabindex="0"
              onkeydown={onActivateKey}
              onclick={() => store.toggleKind(k)}>{k}</span
            >
          {/each}
        </span></span
      >
    </div>

    <div class="modal-body">
      {#if !sessions.length}
        <div class="empty">No sessions match these filters.</div>
      {:else}
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th class="col-pick"></th>
                <th class="col-status">Status</th>
                <th>Project</th>
                <th>Title</th>
                <th class="col-when">Active</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {#each sessions as s (s.id)}
                <BrowseRow session={s} />
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  </div>
</div>

<div class="actionbar" class:show={n > 0 && store.browseOpen}>
  <span class="n"><b>{n}</b> selected</span>
  <button
    class="btn"
    onclick={() => void store.openIds([...store.browseSelected])}
    >Open selected ({n})</button
  >
  <button class="iconbtn" onclick={() => void store.runBrowseBulk("pin")}
    >Pin</button
  >
  <button class="iconbtn" onclick={() => void store.runBrowseBulk("unpin")}
    >Unpin</button
  >
  <button class="linkbtn" onclick={() => store.clearBrowseSelection()}
    >clear</button
  >
</div>
