<script lang="ts">
  import { Block, Box, Button, Group, Stack, Text } from "$ui8kit/ui";
  import DecisionFlow from "$components/widgets/DecisionFlow.svelte";
  import JsonEditor from "$components/widgets/JsonEditor.svelte";
  import { legalLink } from "$lib/layout";
  import type { Playground } from "$lib/playground.svelte";
  import type { Connection, Edge } from "@xyflow/svelte";

  let {
    jsonLabel,
    runLabel,
    hideJson,
    showJson,
    splitLabel,
    deleteEdge,
    locked,
    trapLabel,
    themeMode,
    board,
  }: {
    jsonLabel: string;
    runLabel: string;
    hideJson: string;
    showJson: string;
    splitLabel: string;
    deleteEdge: string;
    locked: string;
    trapLabel: string;
    themeMode: "light" | "dark";
    board: Playground;
  } = $props();

  let jsonOpen = $state(true);
  let jsonWidth = $state(360);
  const graph = $derived(board.graph);

  function clampWidth(value: number, limit: number) {
    return Math.min(limit, Math.max(220, value));
  }

  function startDrag(event: PointerEvent) {
    const handle = event.currentTarget as HTMLElement;
    const row = handle.parentElement;
    const limit = Math.max(220, Math.min(640, (row?.clientWidth ?? 800) - 180));
    const startX = event.clientX;
    const startWidth = jsonWidth;
    handle.setPointerCapture(event.pointerId);
    const move = (ev: PointerEvent) => {
      jsonWidth = clampWidth(startWidth - (ev.clientX - startX), limit);
    };
    const end = () => {
      handle.removeEventListener("pointermove", move);
      handle.removeEventListener("pointerup", end);
      handle.removeEventListener("pointercancel", end);
    };
    handle.addEventListener("pointermove", move);
    handle.addEventListener("pointerup", end);
    handle.addEventListener("pointercancel", end);
  }

  function resizeKey(event: KeyboardEvent) {
    const handle = event.currentTarget as HTMLElement;
    const row = handle.parentElement;
    const limit = Math.max(220, Math.min(640, (row?.clientWidth ?? 800) - 180));
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      jsonWidth = clampWidth(jsonWidth + 16, limit);
    }
    if (event.key === "ArrowRight") {
      event.preventDefault();
      jsonWidth = clampWidth(jsonWidth - 16, limit);
    }
  }

  function acceptConnection(connection: Connection | Edge) {
    if (!legalLink(board.slice, connection.source, connection.target)) return false;
    return !board.links.some((link) => link.source === connection.source && link.target === connection.target);
  }
</script>

<Block tag="section" class="flex min-h-0 min-w-0 flex-1 flex-col gap-3">
  <Group class="w-full min-w-0 justify-between gap-3">
    <Block class="rounded-md border border-border bg-card px-3 py-1.5 text-sm font-medium">{board.slice.title}</Block>
    <Group class="min-w-0 gap-2">
      {#if board.status}
        <Text role="status" class="max-w-xs min-w-0 truncate text-sm text-muted-foreground">{board.status}</Text>
      {:else if !board.canRun}
        <Text class="max-w-xs min-w-0 truncate text-sm text-muted-foreground">{board.reasons()[0] ?? locked}</Text>
      {/if}
      <Button type="button" size="sm" disabled={!board.canRun} onclick={() => board.run()}>
        {runLabel}
      </Button>
      <Button
        type="button"
        size="sm"
        variant="outline"
        aria-pressed={!jsonOpen}
        onclick={() => (jsonOpen = !jsonOpen)}
      >
        {jsonOpen ? hideJson : showJson}
      </Button>
    </Group>
  </Group>
  <Box class="flex min-h-0 min-w-0 flex-1 overflow-hidden">
    <Box class="min-h-0 min-w-0 flex-1">
      {#key board.sliceId}
        <DecisionFlow
          nodes={graph.nodes}
          edges={graph.edges}
          colorMode={themeMode}
          isValidConnection={acceptConnection}
          deleteLabel={deleteEdge}
          onconnect={(connection) => board.connect(connection.source, connection.target)}
          onplace={(id, x, y) => board.place(id, x, y)}
          ondelete={(deleted) =>
            board.removeLinks(deleted.edges.map((edge) => ({ source: edge.source, target: edge.target })))}
        />
      {/key}
    </Box>
    {#if jsonOpen}
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
      <div
        role="separator"
        aria-orientation="vertical"
        aria-label={splitLabel}
        aria-valuemin={220}
        aria-valuemax={640}
        aria-valuenow={jsonWidth}
        tabindex="0"
        class="cursor-col-resize touch-none outline-none"
        onpointerdown={startDrag}
        onkeydown={resizeKey}
      ></div>
      <Stack class="h-full min-h-0 shrink-0 gap-2" style="width: {jsonWidth}px">
        <Text class="text-sm font-medium">{jsonLabel}</Text>
        <JsonEditor value={board.requestText} revision={board.revision} onchange={(text) => board.editRequest(text)} />
      </Stack>
    {/if}
  </Box>
  <input
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
</Block>
