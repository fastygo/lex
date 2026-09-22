<script lang="ts">
  import { Cog, Moon, Sun } from "lucide-svelte/icons";
  import { Block, Box, Button, Disclosure, Group, Input, Stack, Summary, Title } from "$ui8kit/ui";

  let {
    logo,
    items,
    active,
    themeMode,
    themeLight,
    themeDark,
    projectLabel,
    importLabel,
    exportLabel,
    fileLabel,
    onNavigate,
    onTheme,
    onImport,
    onExport,
  }: {
    logo: string;
    items: { href: string; label: string; name: string }[];
    active: string;
    themeMode: "light" | "dark";
    themeLight: string;
    themeDark: string;
    projectLabel: string;
    importLabel: string;
    exportLabel: string;
    fileLabel: string;
    onNavigate: (href: string) => void;
    onTheme: () => void;
    onImport: (file: File) => void;
    onExport: () => void;
  } = $props();
</script>

<Block tag="header" class="border-b border-border px-4 py-3">
  <Group class="w-full justify-between gap-4">
    <Button type="button" variant="ghost" class="px-2 text-base font-semibold" onclick={() => onNavigate("/")}>
      <Title as={1} class="text-base">{logo}</Title>
    </Button>
    <Group class="gap-2">
      {#each items as item (item.href)}
        <Button
          type="button"
          variant={active === item.name ? "secondary" : "ghost"}
          size="sm"
          onclick={() => onNavigate(item.href)}
        >
          {item.label}
        </Button>
      {/each}
      <Button
        type="button"
        variant="outline"
        size="icon"
        aria-label={themeMode === "dark" ? themeLight : themeDark}
        onclick={onTheme}
      >
        {#if themeMode === "dark"}
          <Sun size={18} strokeWidth={2} />
        {:else}
          <Moon size={18} strokeWidth={2} />
        {/if}
      </Button>
      <Disclosure class="relative">
        <Summary variant="ghost" size="sm" class="rounded-md border border-input px-2 py-2" aria-label={projectLabel}>
          <Cog size={18} strokeWidth={2} />
        </Summary>
        <Box class="absolute right-0 z-20 mt-2 w-44 rounded-md border border-border bg-popover p-2 shadow-md">
          <Stack class="gap-2">
            <Button type="button" variant="ghost" class="justify-start" onclick={onExport}>
              {exportLabel}
            </Button>
            <Button type="button" variant="ghost" class="justify-start" onclick={() => document.getElementById("lex-project-file")?.click()}>
              {importLabel}
            </Button>
          </Stack>
        </Box>
      </Disclosure>
      <Input
        id="lex-project-file"
        type="file"
        accept="application/json"
        class="sr-only"
        aria-label={fileLabel}
        onchange={(event: Event) => {
          const input = event.currentTarget as HTMLInputElement;
          const file = input.files?.[0];
          if (file) onImport(file);
          input.value = "";
        }}
      />
    </Group>
  </Group>
</Block>
