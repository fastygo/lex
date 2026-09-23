<script lang="ts">
  import {
    SvelteFlow,
    Background,
    Controls,
    type Connection,
    type Edge,
    type Node,
  } from "@xyflow/svelte";
  import "@xyflow/svelte/dist/style.css";
  import { Box, Button } from "$ui8kit/ui";
  import FlowStep from "$components/widgets/FlowStep.svelte";
  import QuestionGroup from "$components/widgets/QuestionGroup.svelte";

  let {
    nodes = $bindable([]),
    edges = $bindable([]),
    colorMode = "system",
    deleteLabel,
    onconnect,
    ondelete,
    onplace,
    isValidConnection,
  }: {
    nodes?: Node[];
    edges?: Edge[];
    colorMode?: "light" | "dark" | "system";
    deleteLabel: string;
    onconnect?: (connection: Connection) => void;
    ondelete?: (deleted: { nodes: Node[]; edges: Edge[] }) => void;
    onplace?: (id: string, x: number, y: number) => void;
    isValidConnection?: (connection: Connection | Edge) => boolean;
  } = $props();

  const nodeTypes = { step: FlowStep, group: QuestionGroup };
  let menu = $state<{ x: number; y: number; source: string; target: string } | null>(null);
  let onEscape: ((event: KeyboardEvent) => void) | null = null;

  function stopEscape() {
    if (!onEscape) return;
    window.removeEventListener("keydown", onEscape);
    onEscape = null;
  }

  function closeMenu() {
    menu = null;
    stopEscape();
  }

  function openMenu(edge: Edge, event: MouseEvent) {
    event.preventDefault();
    const frame = event.target instanceof Element ? event.target.closest("[data-flow-frame]") : null;
    const rect = frame?.getBoundingClientRect();
    menu = {
      x: event.clientX - (rect?.left ?? 0),
      y: event.clientY - (rect?.top ?? 0),
      source: edge.source,
      target: edge.target,
    };
    stopEscape();
    onEscape = (keyEvent: KeyboardEvent) => {
      if (keyEvent.key === "Escape") closeMenu();
    };
    window.addEventListener("keydown", onEscape);
  }

  function removeEdge() {
    if (!menu) return;
    ondelete?.({ nodes: [], edges: [{ id: `${menu.source}->${menu.target}`, source: menu.source, target: menu.target }] });
    closeMenu();
  }
</script>

<Box
  data-flow-frame
  class="relative h-full min-h-0 w-full overflow-hidden rounded-md border border-border bg-background"
  style="--xy-background-color: var(--muted); --xy-node-border-radius: 5px;"
>
  <Box class="absolute inset-0">
    <SvelteFlow
      bind:nodes
      bind:edges
      {nodeTypes}
      fitView
      {colorMode}
      {onconnect}
      {ondelete}
      {isValidConnection}
      connectionRadius={24}
      nodesDraggable={true}
      nodesConnectable={true}
      elementsSelectable={true}
      onedgecontextmenu={({ edge, event }) => openMenu(edge, event)}
      onpaneclick={() => closeMenu()}
      onnodedragstop={({ nodes: moved }) => {
        for (const node of moved) onplace?.(node.id, node.position.x, node.position.y);
      }}
      class="h-full w-full"
    >
      <Background />
      <Controls />
    </SvelteFlow>
  </Box>
  {#if menu}
    <Box
      role="menu"
      class="absolute z-50 rounded-md border border-border bg-popover p-1 shadow-md"
      style="left: {menu.x}px; top: {menu.y}px"
    >
      <Button type="button" size="sm" variant="ghost" role="menuitem" onclick={removeEdge}>
        {deleteLabel}
      </Button>
    </Box>
  {/if}
</Box>
