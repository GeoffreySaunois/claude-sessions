import { store } from "./store.svelte";

// Global keyboard model for the dashboard. The active list is the Browse list
// while the modal is open, otherwise the Main (pinned) list — see
// store.activeList. Typing in an input/textarea/contenteditable is left alone
// (except Escape), as is the case while a popover is open.

function isTextEntry(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) return false;
  const tag = el.tagName;
  return (
    tag === "INPUT" ||
    tag === "TEXTAREA" ||
    tag === "SELECT" ||
    el.isContentEditable
  );
}

function focusActiveSearch(): void {
  const sel = store.browseOpen
    ? ".overlay.show .modal-controls .searchbox"
    : "header .search-wrap .searchbox";
  const el = document.querySelector<HTMLInputElement>(sel);
  el?.focus();
  el?.select();
}

function blurActiveElement(): void {
  const el = document.activeElement;
  if (el instanceof HTMLElement) el.blur();
}

// Escape unwinds the most specific layer first: unpin confirm, context menu,
// then popover, then help overlay, then modal, then selection, then search.
function handleEscape(): boolean {
  if (store.unpinConfirm) {
    store.cancelUnpin();
    return true;
  }
  if (store.contextMenu) {
    store.closeContextMenu();
    return true;
  }
  if (store.popoverOpen) {
    // The popover owns its own Escape handler (capture phase); don't double-act.
    return false;
  }
  if (store.shortcutsOpen) {
    store.shortcutsOpen = false;
    return true;
  }
  if (store.browseOpen) {
    store.closeBrowse();
    return true;
  }
  const sel = store.browseOpen ? store.browseSelected : store.selected;
  if (sel.size) {
    if (store.browseOpen) store.clearBrowseSelection();
    else store.clearMainSelection();
    return true;
  }
  blurActiveElement();
  return true;
}

export function handleKeydown(e: KeyboardEvent): void {
  if (e.key === "Escape") {
    handleEscape();
    return;
  }

  // While a popover or the unpin-confirm dialog is open, only Escape is honored
  // (handled above) — the dialog owns Enter/Escape itself.
  if (store.popoverOpen || store.unpinConfirm) return;

  // Don't steal keystrokes from text entry, but DO handle the search-focus and
  // help shortcuts that should work from anywhere.
  const typing = isTextEntry(e.target);

  // ⌘K / Ctrl+K: focus the active search box (works even while typing).
  if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) {
    e.preventDefault();
    focusActiveSearch();
    return;
  }

  // Ignore other modified chords so browser/OS shortcuts pass through.
  if (e.metaKey || e.ctrlKey || e.altKey) return;

  // The Browse modal auto-focuses its search box on open; arrow keys must still
  // move the focused row from there (they don't insert text). Arrowing means
  // "navigate now", so we blur the input — that hands subsequent keys
  // (x/p/Enter/⌫…) to the full keymap below. j/k/x/etc. stay literal while the
  // input keeps focus. This is what makes navigation work inside the modal.
  if (typing) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      blurActiveElement();
      store.moveFocus(e.key === "ArrowDown" ? 1 : -1);
    }
    return;
  }

  switch (e.key) {
    case "ArrowDown":
    case "j":
      e.preventDefault();
      store.moveFocus(1);
      return;
    case "ArrowUp":
    case "k":
      e.preventDefault();
      store.moveFocus(-1);
      return;
    case " ":
    case "x":
      e.preventDefault();
      store.toggleFocusedSelect();
      return;
    case "Enter":
    case "o":
      e.preventDefault();
      store.openActive();
      return;
    case "p":
      // Pin only — safe. Unpin (destructive) is Backspace/Delete + confirm.
      e.preventDefault();
      store.pinFocused();
      return;
    case "Backspace":
    case "Delete":
      e.preventDefault();
      store.unpinFocused();
      return;
    case "/":
      e.preventDefault();
      focusActiveSearch();
      return;
    case "b":
      if (!store.browseOpen) {
        e.preventDefault();
        store.openBrowse();
      }
      return;
    case "?":
      e.preventDefault();
      store.shortcutsOpen = !store.shortcutsOpen;
      return;
  }
}
