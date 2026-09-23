<script lang="ts">
  import { Braces, Combine } from "lucide-svelte/icons";
  import { Alert, Block, Box, Button, Group, Text } from "$ui8kit/ui";
  import DecisionFlow from "$components/widgets/DecisionFlow.svelte";
  import JsonEditor from "$components/widgets/JsonEditor.svelte";
  import { legalLink } from "$lib/layout";
  import type { Playground } from "$lib/playground.svelte";
  import type { Connection, Edge } from "@xyflow/svelte";

  let {
    requestLabel,
    stateLabel,
    questionsLabel,
    stateJsonError,
    stateEmptyError,
    questionsJsonError,
    questionsShapeError,
    combineLabel,
    hideCombine,
    runLabel,
    hideJson,
    showJson,
    splitLabel,
    deleteEdge,
    trapLabel,
    themeMode,
    board,
  }: {
    requestLabel: string;
    stateLabel: string;
    questionsLabel: string;
    stateJsonError: string;
    stateEmptyError: string;
    questionsJsonError: string;
    questionsShapeError: string;
    combineLabel: string;
    hideCombine: string;
    runLabel: string;
    hideJson: string;
    showJson: string;
    splitLabel: string;
    deleteEdge: string;
    trapLabel: string;
    themeMode: "light" | "dark";
    board: Playground;
  } = $props();

  type Side = "closed" | "json" | "combine";

  let side = $state<Side>(window.matchMedia("(max-width: 767px)").matches ? "closed" : "json");
  let jsonWidth = $state(360);
  const graph = $derived(board.graph);
  const note = $derived(board.notice());

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

  function openJson() {
    side = side === "json" ? "closed" : "json";
  }

  function openCombine() {
    side = side === "combine" ? "closed" : "combine";
  }

  function acceptConnection(connection: Connection | Edge) {
    if (!legalLink(board.slice, connection.source, connection.target)) return false;
    return !board.links.some((link) => link.source === connection.source && link.target === connection.target);
  }
</script>

<Block tag="section" class="flex min-h-0 min-w-0 flex-1 flex-col gap-3">
  <Group class="w-full min-w-0 items-start justify-between gap-3">
    <Alert
      variant={note.variant}
      role={note.role}
      data-ui8kit="alert"
      class="flex w-fit max-w-[calc(100%-7rem)] flex-col gap-1 px-3 py-2 shadow-none"
    >
      <Text class="text-xs font-bold">{note.title}</Text>
      <Text class="text-[calc(0.75rem*0.9)] text-muted-foreground">{note.text}</Text>
    </Alert>
    <Group class="shrink-0 gap-2">
      <Button type="button" size="sm" disabled={!board.canRun} onclick={() => board.run()}>
        {runLabel}
      </Button>
      <Button
        type="button"
        size="icon"
        variant="ghost"
        aria-pressed={side === "combine"}
        aria-label={side === "combine" ? hideCombine : combineLabel}
        onclick={openCombine}
      >
        <Combine size={18} strokeWidth={2} />
      </Button>
      <Button
        type="button"
        size="icon"
        variant="ghost"
        aria-pressed={side === "json"}
        aria-label={side === "json" ? hideJson : showJson}
        onclick={openJson}
      >
        <Braces size={18} strokeWidth={2} />
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
    {#if side !== "closed"}
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
      <Box class="flex h-full min-h-0 shrink-0 flex-col gap-2" style="width: {jsonWidth}px">
        {#if side === "combine"}
          <Box class="flex min-h-0 flex-1 flex-col gap-1">
            <Text class="px-1 text-xs text-muted-foreground">{stateLabel}</Text>
            {#if board.stateIssue}
              <Text class="px-1 text-xs text-destructive">
                {board.stateIssue === "state-empty" ? stateEmptyError : stateJsonError}
              </Text>
            {/if}
            <JsonEditor
              label={stateLabel}
              value={board.stateText}
              revision={board.stateRevision}
              onchange={(text) => board.editState(text)}
            />
          </Box>
          <Box class="flex min-h-0 flex-1 flex-col gap-1">
            <Text class="px-1 text-xs text-muted-foreground">{questionsLabel}</Text>
            {#if board.questionsIssue}
              <Text class="px-1 text-xs text-destructive">
                {board.questionsIssue === "questions-json" ? questionsJsonError : questionsShapeError}
              </Text>
            {/if}
            <JsonEditor
              label={questionsLabel}
              value={board.questionsText}
              revision={board.questionsRevision}
              onchange={(text) => board.editQuestions(text)}
            />
          </Box>
        {:else}
          <JsonEditor label={requestLabel} value={board.requestText} revision={board.revision} readonly />
        {/if}
      </Box>
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
