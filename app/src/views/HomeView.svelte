<script lang="ts">
  import { Block, Button, Group, List, ListItem, Stack, Text, Title } from "$ui8kit/ui";

  let {
    title,
    lead,
    lines,
    body,
    github,
    playground,
    onPlayground,
  }: {
    title: string;
    lead: string;
    lines: readonly string[];
    body: string;
    github: string;
    playground: string;
    onPlayground: () => void;
  } = $props();
</script>

<Block tag="section" class="mx-auto flex w-full max-w-3xl flex-col gap-8 py-8">
  <Stack class="gap-4">
    <Title as={2}>{title}</Title>
    <Text class="text-muted-foreground">{lead}</Text>
    <List class="gap-1 font-mono text-sm">
      {#each lines as line (line)}
        <ListItem>{line}</ListItem>
      {/each}
    </List>
    <Text>{body}</Text>
  </Stack>
  <Group class="gap-3">
    <Button href="https://github.com/fastygo/lex" external target="_blank" rel="noreferrer">
      {github}
    </Button>
    <Button type="button" variant="outline" onclick={onPlayground}>
      {playground}
    </Button>
  </Group>
</Block>
