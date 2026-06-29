<script lang="ts">
  import { store } from "./store.svelte";

  let show = $state(false);
  let msg = $state("");
  let err = $state(false);
  let timer: ReturnType<typeof setTimeout> | null = null;

  $effect(() => {
    const t = store.toast;
    if (!t) return;
    msg = t.msg;
    err = t.err;
    show = true;
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => (show = false), 2600);
  });
</script>

<div id="toast" class:show class:err>{msg}</div>
