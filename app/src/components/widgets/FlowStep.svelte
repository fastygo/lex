<script lang="ts">
  import { Handle, Position, type NodeProps } from "@xyflow/svelte";
  import { Block, Stack, Text, Title } from "$ui8kit/ui";
  import type { StepData } from "$lib/layout";

  let { data }: NodeProps = $props();
  const step = $derived(data as StepData);
  const handleStyle = "width: 12px; height: 12px;";
</script>

<Block
  class="w-60 cursor-grab rounded-md border bg-card px-3 py-3 shadow-sm active:cursor-grabbing {step.included
    ? 'border-primary'
    : 'border-dashed border-border'}"
>
  {#if !step.grouped && step.lane !== "input"}
    <Handle type="target" position={Position.Top} style={handleStyle} />
  {/if}
  <Stack class="gap-1">
    <Text class="text-xs tracking-wide text-muted-foreground">{step.lane}</Text>
    <Title as={3} class="text-sm">{step.title}</Title>
    <Text class="text-xs text-muted-foreground">{step.detail}</Text>
  </Stack>
  {#if !step.grouped && step.lane !== "output"}
    <Handle type="source" position={Position.Bottom} style={handleStyle} />
  {/if}
</Block>
