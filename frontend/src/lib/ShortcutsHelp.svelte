<script lang="ts">
  import { store } from "./store.svelte";

  // Keyboard shortcuts reference, toggled by `?`. A light overlay listing the
  // real keymap wired in keyboard.ts.
  const KEYS: { keys: string[]; label: string }[] = [
    { keys: ["↑", "k"], label: "Move focus up" },
    { keys: ["↓", "j"], label: "Move focus down" },
    { keys: ["x"], label: "Toggle selection of focused row" },
    { keys: ["Space"], label: "Peek conversation (hold to show)" },
    { keys: ["↵"], label: "Preview conversation" },
    { keys: ["⌘↵"], label: "Open / resume in terminal" },
    { keys: ["p"], label: "Pin focused row" },
    { keys: ["⌫"], label: "Unpin focused row (confirms first)" },
    { keys: ["/", "⌘K"], label: "Focus search" },
    { keys: ["b"], label: "Open Browse all" },
    { keys: ["Esc"], label: "Close conversation / popover / modal, else clear selection" },
    { keys: ["?"], label: "Toggle this help" },
  ];
</script>

{#if store.shortcutsOpen}
  <div
    class="overlay show help-overlay"
    role="presentation"
    onmousedown={(e) => {
      if (e.target === e.currentTarget) store.shortcutsOpen = false;
    }}
  >
    <div class="help-modal" role="dialog" aria-modal="true" aria-label="Keyboard shortcuts">
      <div class="help-head">
        <h2>Keyboard shortcuts</h2>
        <button
          class="modal-close"
          type="button"
          aria-label="Close"
          onclick={() => (store.shortcutsOpen = false)}>✕</button
        >
      </div>
      <div class="help-list">
        {#each KEYS as row (row.label)}
          <div class="help-row">
            <span class="help-keys">
              {#each row.keys as k (k)}<span class="k">{k}</span>{/each}
            </span>
            <span class="help-label">{row.label}</span>
          </div>
        {/each}
      </div>
    </div>
  </div>
{/if}
