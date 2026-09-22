<script lang="ts">
  import { Badge, Block, Button, Grid, Stack, Text, Title } from "$ui8kit/ui";
  import type { SliceDef } from "$lib/slices";

  let {
    title,
    lead,
    openLabel,
    slices,
    onOpen,
  }: {
    title: string;
    lead: string;
    openLabel: string;
    slices: SliceDef[];
    onOpen: (id: string) => void;
  } = $props();
</script>

<Block tag="section" class="mx-auto flex w-full max-w-6xl flex-col gap-6 py-4">
  <Stack class="gap-2">
    <Title as={2}>{title}</Title>
    <Text class="text-muted-foreground">{lead}</Text>
  </Stack>
  <Grid cols="1-3" class="gap-4">
    {#each slices as slice (slice.id)}
      <Button
        type="button"
        variant="outline"
        class="h-auto w-full flex-col items-start gap-3 whitespace-normal p-4 text-left"
        onclick={() => onOpen(slice.id)}
      >
        <Badge variant="secondary">{slice.domain}</Badge>
        <Title as={3} class="text-base">{slice.title}</Title>
        <Text class="text-sm text-muted-foreground">{slice.summary}</Text>
        <Text class="text-sm">{openLabel}</Text>
      </Button>
    {/each}
  </Grid>
</Block>
