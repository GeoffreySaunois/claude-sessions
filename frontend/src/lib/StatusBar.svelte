<script lang="ts">
  import { store } from "./store.svelte";

  // The statusline mirrors the live pinned-workspace state: a coral mode block,
  // count segments, the pinned total, and the keybind legend.
  const counts = $derived(store.counts);
  const pinned = $derived(store.pinned.length);
  const total = $derived(store.sessions.length);
</script>

<div class="statusline" role="status" aria-label="Status line">
  <span class="sl-mode">NORMAL</span>
  <span class="sl-seg">
    <span class="accent tnum">{pinned}</span> pinned
    <span class="muted">·</span>
    <span class="tnum">{total}</span> total
  </span>
  <span class="sl-seg optional">
    <span class="sl-counts">
      <span class="c"><span class="dot busy"></span><b class="tnum">{counts.busy}</b> <span class="muted">busy</span></span>
      <span class="c"><span class="dot waiting"></span><b class="tnum">{counts.waiting}</b> <span class="muted">waiting</span></span>
      <span class="c"><span class="dot inactive"></span><b class="tnum">{counts.inactive}</b> <span class="muted">inactive</span></span>
    </span>
  </span>
  <span class="sl-keys">
    <span class="pair"><span class="k">↑↓</span>navigate</span>
    <span class="pair"><span class="k">x</span>select</span>
    <span class="pair"><span class="k">⌘K</span>search</span>
    <span class="pair"><span class="k">p</span>pin</span>
    <span class="pair"><span class="k">⌫</span>unpin</span>
    <span class="pair"><span class="k">o</span>open</span>
    <span class="pair"><span class="k">↵</span>resume</span>
    <button
      class="pair sl-help"
      type="button"
      title="All shortcuts"
      onclick={() => (store.shortcutsOpen = !store.shortcutsOpen)}
      ><span class="k">?</span>help</button
    >
  </span>
</div>
