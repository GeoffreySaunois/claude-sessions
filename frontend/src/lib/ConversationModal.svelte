<script lang="ts">
  import { store } from "./store.svelte";
  import { parseMarkdown } from "./markdown";
  import { tick } from "svelte";

  const conv = $derived(store.conversation);
  const session = $derived(conv.session);

  function onOverlayDown(e: MouseEvent) {
    // Outside-click on the dim backdrop closes; clicks inside the modal don't.
    if (e.target === e.currentTarget) store.closeConversation();
  }

  function resume() {
    if (session) void store.openIds([session.id]);
  }

  // Auto-scroll to the newest turn (bottom) once the modal opens and whenever the
  // messages change (the fetch resolving). `tick` waits for the rows to render.
  let bodyEl = $state<HTMLDivElement>();
  $effect(() => {
    // Touch the reactive deps that should trigger a scroll-to-bottom.
    void conv.messages.length;
    void conv.open;
    if (!conv.open) return;
    void tick().then(() => {
      if (bodyEl) bodyEl.scrollTop = bodyEl.scrollHeight;
    });
  });
</script>

<div
  class="overlay conv-overlay"
  class:show={conv.open}
  onmousedown={onOverlayDown}
  role="presentation"
>
  <div
    class="conv-modal"
    role="dialog"
    aria-modal="true"
    aria-label="Conversation preview"
  >
    <div class="conv-head">
      <h2 class="conv-title">{session?.title || "Conversation"}</h2>
      <button class="conv-resume" type="button" onclick={resume}>
        Resume <span class="k">⌘↵</span>
      </button>
      <button
        class="modal-close"
        type="button"
        title="Close (Esc)"
        aria-label="Close"
        onclick={() => store.closeConversation()}>✕</button
      >
    </div>

    <div class="conv-body" bind:this={bodyEl}>
      {#if conv.loading}
        <div class="conv-state">Loading conversation…</div>
      {:else if conv.error}
        <div class="conv-state err">Couldn't load conversation: {conv.error}</div>
      {:else if !conv.messages.length}
        <div class="conv-state">No conversation text to preview.</div>
      {:else}
        {#each conv.messages as m, i (i)}
          <div class="conv-turn" class:assistant={m.role === "assistant"}>
            <div class="conv-role">{m.role === "assistant" ? "Claude" : "You"}</div>
            {#if m.text}
              <div class="conv-text">
                {#each parseMarkdown(m.text) as block (block)}
                  {#if block.kind === "code"}
                    <pre class="conv-code"><code>{block.text}</code></pre>
                  {:else}
                    <p class="conv-para">{#each block.spans as span (span)}{#if span.code}<code class="conv-inline">{span.text}</code>{:else}{span.text}{/if}{/each}</p>
                  {/if}
                {/each}
              </div>
            {/if}
            {#if m.tools.length}
              <div class="conv-tools">⚙ {m.tools.join(" · ")}</div>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>
