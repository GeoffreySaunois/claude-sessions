<script lang="ts">
  import { store } from "./store.svelte";
  import Popover from "./Popover.svelte";
  import { onActivateKey } from "./clickable";
  import type { Session } from "./types";

  let { session }: { session: Session } = $props();

  let anchor = $state<HTMLElement>();
  let open = $state(false);
</script>

<div class="tags">
  {#each session.tags || [] as t (t)}
    <span class="tag"
      >{t}<span
        class="x"
        title="remove"
        role="button"
        tabindex="0"
        onkeydown={onActivateKey}
        onclick={(e) => {
          e.stopPropagation();
          void store.removeTag(session, t);
        }}>×</span
      ></span
    >
  {/each}
  <span
    bind:this={anchor}
    class="selctl"
    role="button"
    tabindex="0"
    onkeydown={onActivateKey}
    onclick={(e) => {
      e.stopPropagation();
      open = true;
    }}>＋ tag</span
  >
</div>

{#if open && anchor}
  <Popover
    {anchor}
    options={store.options.tags}
    isSelected={(t) => (session.tags || []).includes(t)}
    onpick={(value) => {
      if (value === "") return;
      void store.toggleTag(session, value);
    }}
    keepOpen
    emptyLabel="No tags"
    onclose={() => (open = false)}
  />
{/if}
