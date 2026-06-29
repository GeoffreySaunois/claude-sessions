<script lang="ts">
  import { store } from "./store.svelte";
  import MainRow from "./MainRow.svelte";
  import type { Session } from "./types";

  let { name, sessions }: { name: string; sessions: Session[] } = $props();

  const collapsed = $derived(store.collapsed.has(name));
  const allSel = $derived(
    sessions.length > 0 && sessions.every((s) => store.selected.has(s.id)),
  );
</script>

<div class="group">
  {#if name}
    <div class="group-head" class:collapsed>
      <button
        class="group-caret"
        type="button"
        title={collapsed ? "Expand group" : "Collapse group"}
        aria-label={collapsed ? "Expand group" : "Collapse group"}
        onclick={() => store.toggleCollapsed(name)}>▾</button
      >
      <span class="group-name mono">{name}</span>
      <span class="group-count tnum">{sessions.length}</span>
      <label class="group-selall" title="Select every session in this group">
        <input
          type="checkbox"
          checked={allSel}
          onchange={(e) =>
            store.setGroupSelected(sessions, e.currentTarget.checked)}
        />
        select all
      </label>
    </div>
  {/if}

  {#if !(name && collapsed)}
    <div class="rows">
      {#each sessions as s (s.id)}
        <MainRow session={s} />
      {/each}
    </div>
  {/if}
</div>
