<script lang="ts">
  import { store } from "./store.svelte";
  import { projBase, relTime, tildePath } from "./derive";
  import { focusOnClick } from "./clickable";
  import StatusBadge from "./StatusBadge.svelte";
  import TitleCell from "./TitleCell.svelte";
  import PinControl from "./PinControl.svelte";
  import type { Session } from "./types";

  let { session }: { session: Session } = $props();
  const selected = $derived(store.browseSelected.has(session.id));
  const focused = $derived(store.focusedSession?.id === session.id);

  let rowEl = $state<HTMLTableRowElement>();
  $effect(() => {
    if (focused) rowEl?.scrollIntoView({ block: "nearest" });
  });
</script>

<tr
  bind:this={rowEl}
  use:focusOnClick={() => store.focusSessionId(session.id)}
  class:sel={selected}
  class:kfocus={focused}
  oncontextmenu={(e) => {
    e.preventDefault();
    store.openContextMenu(session, "browse", e.clientX, e.clientY);
  }}
>
  <td class="col-pick">
    <input
      type="checkbox"
      checked={selected}
      onchange={(e) =>
        store.toggleBrowseSelect(session.id, e.currentTarget.checked)}
    />
  </td>
  <td class="col-status"><StatusBadge status={session.status} variant="badge" /></td>
  <td>
    <div class="proj">{projBase(session.cwd)}</div>
    <div class="cwd" title={session.cwd}>{tildePath(session.cwd)}</div>
  </td>
  <td><TitleCell {session} renderingQuery={store.browseFilter} view="browse" /></td>
  <td class="col-when"
    ><span class="when" title={session.lastActive}>{relTime(session.lastActive)}</span
    ></td
  >
  <td><PinControl {session} /></td>
</tr>
