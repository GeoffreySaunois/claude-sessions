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
      <span class="group-name">{name}</span>
      <span class="group-count">{sessions.length}</span>
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
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th class="col-pick"></th>
            <th class="col-status">Status</th>
            <th>Project</th>
            <th>Title</th>
            <th class="col-when">Active</th>
            <th>Branch</th>
            <th>Category</th>
            <th>Tags</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each sessions as s (s.id)}
            <MainRow session={s} />
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
