<script lang="ts">
  import { store } from "./store.svelte";
  import { projBase, relTime } from "./derive";
  import StatusBadge from "./StatusBadge.svelte";
  import TitleCell from "./TitleCell.svelte";
  import CategoryControl from "./CategoryControl.svelte";
  import TagsControl from "./TagsControl.svelte";
  import type { Session } from "./types";

  let { session }: { session: Session } = $props();

  const selected = $derived(store.selected.has(session.id));
</script>

<tr class:sel={selected} class:archived={session.archived}>
  <td class="col-pick">
    <input
      type="checkbox"
      checked={selected}
      onchange={(e) =>
        store.toggleMainSelect(session.id, e.currentTarget.checked)}
    />
  </td>
  <td class="col-status"><StatusBadge status={session.status} /></td>
  <td>
    <div class="proj">{projBase(session.cwd)}</div>
    <div class="cwd" title={session.cwd}>{session.cwd || ""}</div>
  </td>
  <td><TitleCell {session} renderingQuery={store.filter} /></td>
  <td class="col-when"
    ><span class="when" title={session.lastActive}>{relTime(session.lastActive)}</span
    ></td
  >
  <td>
    {#if session.gitBranch}
      <span class="branch">{session.gitBranch}</span>
    {:else}
      <span class="muted"></span>
    {/if}
  </td>
  <td class="cat-cell"><CategoryControl {session} /></td>
  <td class="tags-cell"><TagsControl {session} /></td>
  <td>
    <div class="tags">
      <button
        class="iconbtn"
        class:on={session.archived}
        title={session.archived ? "Unarchive" : "Archive (set aside, keep pinned)"}
        onclick={() => void store.commitMeta(session, { archived: !session.archived })}
        >{session.archived ? "archived" : "archive"}</button
      >
      <button
        class="iconbtn"
        title="Remove from dashboard"
        onclick={() => void store.setPinned(session, false)}>unpin</button
      >
    </div>
  </td>
</tr>
