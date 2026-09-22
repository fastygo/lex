<script lang="ts">
  import { Block, Box, Button, Input, Stack, Text, Title } from "$ui8kit/ui";
  import DecisionFlow from "$components/widgets/DecisionFlow.svelte";
  import JsonEditor from "$components/widgets/JsonEditor.svelte";
  import type { StepData } from "$lib/layout";
  import type { Playground } from "$lib/playground.svelte";
  import type { Edge, Node } from "@xyflow/svelte";

  let {
    heading,
    jsonLabel,
    runLabel,
    locked,
    trapLabel,
    themeMode,
    board,
  }: {
    heading: string;
    jsonLabel: string;
    runLabel: string;
    locked: string;
    trapLabel: string;
    themeMode: "light" | "dark";
    board: Playground;
  } = $props();

  let nodes = $state<Node<StepData>[]>([]);
  let edges = $state<Edge[]>([]);

  $effect(() => {
    const next = board.graph;
    nodes = next.nodes.map((node) => ({
      ...node,
      data: { ...node.data, ontoggle: (id: string) => board.toggle(id) },
    }));
    edges = next.edges;
  });
</script>

<Block tag="section" class="flex min-h-0 flex-1 flex-col gap-4">
  <Stack class="gap-2">
    <Title as={2}>{heading}</Title>
    <Text class="text-muted-foreground">{board.slice.title}. {board.slice.domain}.</Text>
  </Stack>
  <Box class="grid min-h-0 flex-1 gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(18rem,0.8fr)]">
    <DecisionFlow bind:nodes bind:edges colorMode={themeMode} />
    <Stack class="min-h-0 gap-3">
      <Text class="text-sm font-medium">{jsonLabel}</Text>
      <JsonEditor value={board.requestText} revision={board.revision} onchange={(text) => board.editRequest(text)} />
      <Button type="button" disabled={!board.canRun} onclick={() => board.run()}>
        {runLabel}
      </Button>
      {#if board.status}
        <Text role="status">{board.status}</Text>
      {:else if !board.canRun}
        <Text class="text-sm text-muted-foreground">{locked}</Text>
      {/if}
      {#each board.reasons() as reason (reason)}
        <Text class="text-sm text-muted-foreground">{reason}</Text>
      {/each}
      <Input
        type="text"
        name="company"
        tabindex="-1"
        autocomplete="off"
        class="sr-only"
        aria-label={trapLabel}
        oninput={(event: Event) => {
          const target = event.currentTarget as HTMLInputElement;
          board.trap = target.value;
        }}
      />
    </Stack>
  </Box>
</Block>
