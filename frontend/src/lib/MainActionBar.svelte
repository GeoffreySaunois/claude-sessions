<script lang="ts">
  import { store } from "./store.svelte";
  import Popover from "./Popover.svelte";

  const n = $derived(store.selected.size);
  // The Main action bar yields to the modal so the two never overlap.
  const show = $derived(n > 0 && !store.browseOpen);

  let catAnchor = $state<HTMLElement>();
  let catOpen = $state(false);
</script>

<div class="selbar-wrap" class:show>
<div class="selbar" role="region" aria-label="Selection actions">
  <span class="n"><span class="badge-n tnum">{n}</span> selected</span>
  <button class="iconbtn" onclick={() => void store.openIds([...store.selected])}
    ><span class="g">↗</span> Open</button
  >
  <button class="iconbtn" onclick={() => void store.runMainBulk("archive")}
    ><span class="g">⌗</span> Archive</button
  >
  <button class="iconbtn" onclick={() => void store.runMainBulk("unarchive")}
    ><span class="g">⊞</span> Unarchive</button
  >
  <button
    class="iconbtn"
    onclick={() => store.requestUnpin([...store.selected], "main")}
    ><span class="g">⊘</span> Unpin</button
  >
  <button
    class="iconbtn"
    bind:this={catAnchor}
    onclick={() => (catOpen = true)}><span class="g">⊕</span> Move to category…</button
  >
  <button class="linkbtn" onclick={() => store.clearMainSelection()}>Clear</button>
</div>
</div>

{#if catOpen && catAnchor}
  <Popover
    anchor={catAnchor}
    options={store.options.categories}
    isSelected={() => false}
    onpick={(value) => void store.runMainBulk("category", value)}
    showClear
    clearLabel="✕ Clear category"
    emptyLabel="No categories"
    onclose={() => (catOpen = false)}
  />
{/if}
