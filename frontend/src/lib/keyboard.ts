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

// Escape unwinds the most specific layer first: conversation preview, unpin
// confirm, context menu, then popover, then help overlay, then browse modal,
// then selection, then search.
function handleEscape(): boolean {
  if (store.conversation.open) {
    store.closeConversation();
    return true;
  }
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

  // ⌘↵ / Ctrl+↵: open/resume in the terminal (the selection, else the focused
  // row). This is the action plain Enter used to perform.
  if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
    e.preventDefault();
    store.openActive();
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
    case " ": {
      // Space peeks the focused conversation (hold-to-show). Guard against the
      // OS key-repeat firing keydown repeatedly while held — the peek opens once.
      // preventDefault stops the page from scrolling.
      e.preventDefault();
      if (e.repeat) return;
      const focused = store.focusedSession;
      if (focused) store.openConversation(focused, { transient: true });
      return;
    }
    case "x":
      e.preventDefault();
      store.toggleFocusedSelect();
      return;
    case "Enter": {
      // Enter opens the conversation preview sticky. While a peek is showing,
      // it promotes the peek so releasing Space won't close it.
      e.preventDefault();
      if (store.conversation.open) {
        store.promoteConversation();
        return;
      }
      const focused = store.focusedSession;
      if (focused) store.openConversation(focused, { transient: false });
      return;
    }
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

// Releasing Space ends a hold-to-peek: it closes the conversation modal ONLY if
// it's still the transient peek (an Enter-promoted sticky preview survives). No
// text-entry guard is needed — the peek only opens from the full keymap, which
// already excludes text entry.
export function handleKeyup(e: KeyboardEvent): void {
  if (e.key === " ") store.closeConversationPeek();
}
