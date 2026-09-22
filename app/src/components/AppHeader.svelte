<script lang="ts">
  import { Block, Title } from "$ui8kit/ui";

  let { title }: { title: string } = $props();
</script>

<Block tag="header" class="border-b border-border px-4 py-4">
  <Title as={1} class="text-sm font-semibold">{title}</Title>
</Block>
