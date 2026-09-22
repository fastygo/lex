<script lang="ts">
  import type { Snippet } from "svelte";
  import { Block } from "$ui8kit/ui";

  let { children }: { children?: Snippet } = $props();
</script>

<Block tag="main" id="main-content" class="flex flex-1 flex-col p-4">
  {@render children?.()}
</Block>
