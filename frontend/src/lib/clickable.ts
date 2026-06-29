// Keyboard activation for non-button elements styled as controls: Enter/Space
// fire a synthetic click so keyboard users operate them like a button. Wired as
// an inline onkeydown handler (recognized by the a11y linter, unlike an action).
export function onActivateKey(e: KeyboardEvent) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    (e.currentTarget as HTMLElement).click();
  }
}
