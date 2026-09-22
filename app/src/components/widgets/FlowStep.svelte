<script lang="ts">
  import { Handle, Position, type NodeProps } from "@xyflow/svelte";
  import { Block, Button, Stack, Text, Title } from "$ui8kit/ui";
  import type { StepData } from "$lib/layout";

  let { data }: NodeProps = $props();
  const step = $derived(data as StepData);
</script>

<Block
  class="w-60 rounded-md border bg-card px-3 py-3 shadow-sm {step.included
    ? 'border-primary'
    : 'border-dashed border-border'}"
>
  {#if step.lane !== "input"}
    <Handle type="target" position={Position.Top} />
  {/if}
  <Stack class="gap-2">
    <Text class="text-xs tracking-wide text-muted-foreground">{step.lane}</Text>
    <Title as={3} class="text-sm">{step.title}</Title>
    <Text class="text-xs text-muted-foreground">{step.detail}</Text>
    {#if step.questionId}
      <Button
        type="button"
        size="sm"
        variant={step.included ? "secondary" : "outline"}
        onclick={(event: MouseEvent) => {
          event.stopPropagation();
          if (step.questionId) step.ontoggle?.(step.questionId);
        }}
      >
        {step.included ? step.includedLabel : step.includeLabel}
      </Button>
    {/if}
  </Stack>
  {#if step.lane !== "output"}
    <Handle type="source" position={Position.Bottom} />
  {/if}
</Block>
