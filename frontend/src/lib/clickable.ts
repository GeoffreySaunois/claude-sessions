// Keyboard activation for non-button elements styled as controls: Enter/Space
// fire a synthetic click so keyboard users operate them like a button. Wired as
// an inline onkeydown handler (recognized by the a11y linter, unlike an action).
export function onActivateKey(e: KeyboardEvent) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    (e.currentTarget as HTMLElement).click();
  }
}

// Action: move the keyboard focus cursor to this row when it's clicked. Attached
// imperatively (not an inline onclick) so a row stays a plain container rather
// than being mislabelled as a button — keyboard users navigate with the arrows.
export function focusOnClick(node: HTMLElement, fn: () => void) {
  const handler = () => fn();
  node.addEventListener("click", handler);
  return { destroy: () => node.removeEventListener("click", handler) };
}
