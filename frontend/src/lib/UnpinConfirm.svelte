<script lang="ts">
  import { store } from "./store.svelte";
  import { tick } from "svelte";

  // Confirmation for the destructive unpin (clears category/tags/archived).
  // Enter = confirm, Escape = cancel; the confirm button is focused by default
  // and focus is trapped to the two buttons.
  const pending = $derived(store.unpinConfirm);

  let confirmEl = $state<HTMLButtonElement>();
  let cancelEl = $state<HTMLButtonElement>();

  $effect(() => {
    if (pending) void tick().then(() => confirmEl?.focus());
  });

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      void store.confirmUnpin();
    } else if (e.key === "Escape") {
      e.preventDefault();
      store.cancelUnpin();
    } else if (e.key === "Tab") {
      // Trap focus between the two buttons.
      e.preventDefault();
      (document.activeElement === confirmEl ? cancelEl : confirmEl)?.focus();
    }
  }
</script>

{#if pending}
  <div
    class="overlay show confirm-overlay"
    role="presentation"
    onmousedown={(e) => {
      if (e.target === e.currentTarget) store.cancelUnpin();
    }}
  >
    <div
      class="confirm-modal"
      role="alertdialog"
      aria-modal="true"
      aria-label="Confirm unpin"
      tabindex="-1"
      onkeydown={onKeydown}
    >
      <h2>Remove {pending.label} from the dashboard?</h2>
      <p>This clears its category and tags.</p>
      <div class="confirm-actions">
        <button
          class="linkbtn"
          type="button"
          bind:this={cancelEl}
          onclick={() => store.cancelUnpin()}>Cancel</button
        >
        <button
          class="btn danger"
          type="button"
          bind:this={confirmEl}
          onclick={() => void store.confirmUnpin()}>Remove</button
        >
      </div>
    </div>
  </div>
{/if}
