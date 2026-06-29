<script lang="ts">
  import { store } from "./store.svelte";
  import Popover from "./Popover.svelte";
  import { onActivateKey } from "./clickable";
  import { tokenVar } from "./derive";
  import type { Session } from "./types";

  let { session }: { session: Session } = $props();

  let anchor = $state<HTMLElement>();
  let open = $state(false);
</script>

<span
  bind:this={anchor}
  class={session.category ? "cat-pill" : "selctl"}
  style={session.category ? `--tok: ${tokenVar(session.category)}` : undefined}
  role="button"
  tabindex="0"
  onkeydown={onActivateKey}
  onclick={(e) => {
    e.stopPropagation();
    open = true;
  }}
>
  {session.category ? session.category : "＋ category"}
</span>

{#if open && anchor}
  <Popover
    {anchor}
    options={store.options.categories}
    isSelected={(c) => c === session.category}
    onpick={(value) => void store.setCategory(session, value)}
    showClear={!!session.category}
    clearLabel="✕ Clear category"
    emptyLabel="No categories"
    onclose={() => (open = false)}
  />
{/if}
