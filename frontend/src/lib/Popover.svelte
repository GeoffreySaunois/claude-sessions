<script lang="ts">
  import { store } from "./store.svelte";
  import { onActivateKey } from "./clickable";
  import { tick } from "svelte";

  interface Props {
    anchor: HTMLElement;
    options: string[];
    // Which options are currently checked (for the ✓ marks).
    isSelected: (name: string) => boolean;
    // Picking an existing or freshly-typed value.
    onpick: (value: string) => void;
    // Keep the popover open after a pick (tag multi-select); single-select closes.
    keepOpen?: boolean;
    // Optional "Clear" row at the bottom (category single-select).
    showClear?: boolean;
    clearLabel?: string;
    emptyLabel: string;
    onclose: () => void;
  }

  let {
    anchor,
    options,
    isSelected,
    onpick,
    keepOpen = false,
    showClear = false,
    clearLabel = "✕ Clear",
    emptyLabel,
    onclose,
  }: Props = $props();

  let query = $state("");
  let popEl = $state<HTMLDivElement>();
  let inputEl = $state<HTMLInputElement>();
  let style = $state("");

  const filtered = $derived(
    options.filter((o) => o.toLowerCase().includes(query.toLowerCase())),
  );
  const typed = $derived(query.trim());
  const exact = $derived(
    options.some((o) => o.toLowerCase() === typed.toLowerCase()),
  );
  const showCreate = $derived(typed !== "" && !exact);
  const showEmpty = $derived(filtered.length === 0 && !showCreate);

  // Positioned with `position: fixed`, so coordinates are viewport-relative
  // (no scroll offset). This keeps the popover anchored to its trigger even when
  // it lives inside a `position: relative` row or the modal overlay.
  function position() {
    if (!popEl) return;
    const rect = anchor.getBoundingClientRect();
    const top = rect.bottom + 4;
    const left = rect.left;
    style = `top:${top}px;left:${left}px`;
    // Clamp within the viewport (both axes) once the box has a measured size.
    requestAnimationFrame(() => {
      if (!popEl) return;
      const pr = popEl.getBoundingClientRect();
      let clampedLeft = left;
      if (pr.right > window.innerWidth - 8) {
        clampedLeft = Math.max(8, window.innerWidth - pr.width - 8);
      }
      let clampedTop = top;
      if (pr.bottom > window.innerHeight - 8) {
        // Flip above the anchor when it would overflow the bottom edge.
        clampedTop = Math.max(8, rect.top - pr.height - 4);
      }
      style = `top:${clampedTop}px;left:${clampedLeft}px`;
    });
  }

  // Move the popover to <body> so a row's `:hover { transform }` (which would
  // otherwise become the containing block for this position:fixed element and
  // make it jump as the mouse moves between rows) can never capture it.
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.parentNode?.removeChild(node) };
  }

  function pick(value: string) {
    onpick(value);
    if (!keepOpen) onclose();
  }

  function onOutside(ev: MouseEvent) {
    if (popEl && !popEl.contains(ev.target as Node)) onclose();
  }
  function onKey(ev: KeyboardEvent) {
    if (ev.key === "Escape") {
      ev.stopPropagation();
      onclose();
    }
  }

  $effect(() => {
    store.popoverOpen = true;
    position();
    void tick().then(() => inputEl?.focus());
    // Defer listener attach so the opening click doesn't immediately close us.
    const t = setTimeout(() => {
      document.addEventListener("mousedown", onOutside, true);
      document.addEventListener("keydown", onKey, true);
    }, 0);
    return () => {
      clearTimeout(t);
      document.removeEventListener("mousedown", onOutside, true);
      document.removeEventListener("keydown", onKey, true);
      store.popoverOpen = false;
    };
  });
</script>

<div class="popover" use:portal bind:this={popEl} {style}>
  <input
    class="psearch"
    type="text"
    placeholder="Search or create…"
    bind:value={query}
    bind:this={inputEl}
    autocomplete="off"
    spellcheck="false"
  />
  <div class="poplist">
    {#each filtered as opt (opt)}
      <div class="popitem" onclick={() => pick(opt)} onkeydown={onActivateKey} role="button" tabindex="0">
        <span class="check">{isSelected(opt) ? "✓" : ""}</span>
        <span class="name">{opt}</span>
      </div>
    {/each}
    {#if showCreate}
      <div
        class="popitem create"
        onclick={() => pick(typed)}
        onkeydown={onActivateKey}
        role="button"
        tabindex="0"
      >
        <span class="name">➕ Create "{typed}"</span>
      </div>
    {/if}
    {#if showEmpty}
      <div class="popempty">{emptyLabel}</div>
    {/if}
  </div>
  {#if showClear}
    <div class="popdiv"></div>
    <div
      class="popitem clear"
      onclick={() => pick("")}
      onkeydown={onActivateKey}
      role="button"
      tabindex="0"
    >
      <span class="name">{clearLabel}</span>
    </div>
  {/if}
</div>
