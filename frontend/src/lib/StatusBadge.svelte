<script lang="ts">
  import type { SessionStatus } from "./types";
  // A vivid status pill (BUSY / WAITING / INACTIVE). `variant="badge"` is the
  // boxier form used in the Browse table; the default `tag` form is the compact
  // pill that sits inline on a card's title line.
  let {
    status,
    variant = "tag",
  }: { status: SessionStatus; variant?: "tag" | "badge" } = $props();
</script>

<span class="{variant === 'badge' ? 'badge' : 'status-tag'} {status}">{status}</span>
