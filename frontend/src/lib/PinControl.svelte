<script lang="ts">
  import { store } from "./store.svelte";
  import type { Session } from "./types";

  let { session }: { session: Session } = $props();
</script>

<button
  class="pinbtn"
  class:pinned={session.pinned}
  type="button"
  title={session.pinned ? "Remove from dashboard" : "Pin to dashboard"}
  onclick={(e) => {
    e.stopPropagation();
    void store.setPinned(session, !session.pinned);
  }}
>
  <span class="ic">{session.pinned ? "★" : "☆"}</span>{session.pinned
    ? "Pinned"
    : "Pin"}
</button>
