<script lang="ts">
  import { store } from "./store.svelte";
  import { projBase, relTime } from "./derive";
  import StatusBadge from "./StatusBadge.svelte";
  import TitleCell from "./TitleCell.svelte";
  import PinControl from "./PinControl.svelte";
  import type { Session } from "./types";

  let { session }: { session: Session } = $props();
  const selected = $derived(store.browseSelected.has(session.id));
</script>

<tr class:sel={selected}>
  <td class="col-pick">
    <input
      type="checkbox"
      checked={selected}
      onchange={(e) =>
        store.toggleBrowseSelect(session.id, e.currentTarget.checked)}
    />
  </td>
  <td class="col-status"><StatusBadge status={session.status} /></td>
  <td>
    <div class="proj">{projBase(session.cwd)}</div>
    <div class="cwd" title={session.cwd}>{session.cwd || ""}</div>
  </td>
  <td><TitleCell {session} renderingQuery={store.browseFilter} /></td>
  <td class="col-when"
    ><span class="when" title={session.lastActive}>{relTime(session.lastActive)}</span
    ></td
  >
  <td><PinControl {session} /></td>
</tr>
