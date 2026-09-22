<script lang="ts">
  import { Block, Stack, Text, Title } from "$ui8kit/ui";

  let { title, body }: { title: string; body: string } = $props();
</script>

<Block tag="section">
  <Stack class="gap-4">
    <Title as={2}>{title}</Title>
    <Text>{body}</Text>
  </Stack>
</Block>
