<script lang="ts">
  import { store } from "./store.svelte";
  import Popover from "./Popover.svelte";

  const n = $derived(store.selected.size);
  // The Main action bar yields to the modal so the two never overlap.
  const show = $derived(n > 0 && !store.browseOpen);

  let catAnchor = $state<HTMLElement>();
  let catOpen = $state(false);
</script>

<div class="actionbar" class:show>
  <span class="n"><b>{n}</b> selected</span>
  <button class="btn" onclick={() => void store.openIds([...store.selected])}
    >Open selected ({n})</button
  >
  <button class="iconbtn" onclick={() => void store.runMainBulk("unpin")}
    >Remove from dashboard</button
  >
  <button class="iconbtn" onclick={() => void store.runMainBulk("archive")}
    >Archive</button
  >
  <button class="iconbtn" onclick={() => void store.runMainBulk("unarchive")}
    >Unarchive</button
  >
  <button
    class="iconbtn"
    bind:this={catAnchor}
    onclick={() => (catOpen = true)}>Move to category…</button
  >
  <button class="linkbtn" onclick={() => store.clearMainSelection()}>clear</button>
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
