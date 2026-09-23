<script lang="ts">
  import type { Snippet } from "svelte";
  import { Block } from "$ui8kit/ui";
  import { cn } from "$ui8kit/utils";

  let { children, class: className = "" }: { children?: Snippet; class?: string } = $props();
</script>

<Block tag="main" id="main-content" class={cn("flex min-h-0 w-full min-w-0 flex-1 flex-col overflow-x-clip", className)}>
  {@render children?.()}
</Block>
