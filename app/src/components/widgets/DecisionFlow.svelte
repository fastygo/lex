<script lang="ts">
  import { SvelteFlow, Background, Controls, type Edge, type Node } from "@xyflow/svelte";
  import "@xyflow/svelte/dist/style.css";
  import { Box } from "$ui8kit/ui";
  import FlowStep from "$components/widgets/FlowStep.svelte";
  import type { StepData } from "$lib/layout";

  let {
    nodes = $bindable([]),
    edges = $bindable([]),
    colorMode = "system",
  }: {
    nodes?: Node<StepData>[];
    edges?: Edge[];
    colorMode?: "light" | "dark" | "system";
  } = $props();

  const nodeTypes = { step: FlowStep };
</script>

<Box class="relative h-[36rem] overflow-hidden rounded-md border border-border bg-background" style="--xy-background-color: var(--muted); --xy-node-border-radius: 5px;">
  <Box class="absolute inset-0">
    <SvelteFlow
      bind:nodes
      bind:edges
      {nodeTypes}
      fitView
      {colorMode}
      nodesDraggable={false}
      nodesConnectable={false}
      elementsSelectable={false}
      class="h-full w-full"
    >
      <Background />
      <Controls />
    </SvelteFlow>
  </Box>
</Box>
