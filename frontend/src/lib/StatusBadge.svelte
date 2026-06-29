<script lang="ts">
  import type { SessionStatus } from "./types";
  let { status }: { status: SessionStatus } = $props();
</script>

<span class="badge {status}">
  <span class="dot {status}"></span>{status}
</span>
