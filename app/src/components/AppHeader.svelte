<script lang="ts">
  import { Cog, Menu, Moon, Sun, X } from "lucide-svelte/icons";
  import { Block, Box, Button, Dialog, Disclosure, Group, Inline, List, ListItem, Separator, Stack, Summary, Text, Title } from "$ui8kit/ui";

  let {
    logo,
    items,
    active,
    themeMode,
    themeLight,
    themeDark,
    menuLabel,
    menuTitle,
    closeLabel,
    project,
    onNavigate,
    onTheme,
  }: {
    logo: string;
    items: { href: string; label: string; name: string }[];
    active: string;
    themeMode: "light" | "dark";
    themeLight: string;
    themeDark: string;
    menuLabel: string;
    menuTitle: string;
    closeLabel: string;
    project?: {
      label: string;
      importLabel: string;
      exportLabel: string;
      fileLabel: string;
      onImport: (file: File) => void;
      onExport: () => void;
    };
    onNavigate: (href: string) => void;
    onTheme: () => void;
  } = $props();

  let menuOpen = $state(false);

  function go(href: string) {
    menuOpen = false;
    onNavigate(href);
  }

  function pickProjectFile() {
    menuOpen = false;
    document.getElementById("lex-project-file")?.click();
  }

  function onKey(event: KeyboardEvent) {
    if (menuOpen && event.key === "Escape") menuOpen = false;
  }

  function onScrim(event: MouseEvent) {
    if (event.target === event.currentTarget) menuOpen = false;
  }

  function watchDesktop() {
    const query = window.matchMedia("(min-width: 768px)");
    const onChange = () => {
      if (query.matches) menuOpen = false;
    };
    query.addEventListener("change", onChange);
    return () => query.removeEventListener("change", onChange);
  }
</script>

<svelte:window onkeydown={onKey} />

<Block tag="header" class="relative min-w-0 border-b border-border px-4 py-3" {@attach watchDesktop}>
  <Group class="w-full min-w-0 justify-between gap-4">
    <Button type="button" variant="ghost" class="px-2 text-base font-bold" onclick={() => onNavigate("/")}>
      <Title as={1} class="text-base font-bold">
        <Inline class="text-logo-primary">{logo.slice(0, 2)}</Inline><Inline class="text-logo-accent">{logo.slice(2)}</Inline>
      </Title>
    </Button>
    <Group class="gap-2">
      <Group class="hidden gap-2 md:flex">
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
      </Group>
      <Button
        type="button"
        variant="ghost"
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
      {#if project}
        <Disclosure variant="ghost" class="relative hidden md:block">
          <Summary variant="ghost" class="h-9 w-9 justify-center rounded-md p-0" aria-label={project.label}>
            <Cog size={18} strokeWidth={2} />
          </Summary>
          <Box class="absolute right-0 z-20 mt-2 w-44 rounded-md border border-border bg-popover p-2 shadow-md">
            <Stack class="gap-2">
              <Button type="button" variant="ghost" class="justify-start" onclick={project.onExport}>
                {project.exportLabel}
              </Button>
              <Button type="button" variant="ghost" class="justify-start" onclick={pickProjectFile}>
                {project.importLabel}
              </Button>
            </Stack>
          </Box>
        </Disclosure>
        <input
          id="lex-project-file"
          type="file"
          accept="application/json"
          class="sr-only"
          aria-label={project.fileLabel}
          onchange={(event: Event) => {
            const input = event.currentTarget as HTMLInputElement;
            const file = input.files?.[0];
            if (file) project.onImport(file);
            input.value = "";
          }}
        />
      {/if}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="md:hidden"
        aria-label={menuLabel}
        aria-expanded={menuOpen}
        aria-controls="lex-nav-sheet"
        onclick={() => (menuOpen = true)}
      >
        <Menu size={18} strokeWidth={2} />
      </Button>
    </Group>
  </Group>
  <Dialog
    id="lex-nav-sheet"
    variant="sheet"
    size="sm"
    open={menuOpen}
    data-ui8kit="dialog"
    data-side="start"
    data-state={menuOpen ? "open" : "closed"}
    aria-label={menuTitle}
    onclick={onScrim}
  >
    <Stack class="h-full w-72 max-w-[85vw] gap-4 border-r border-border bg-background p-4 shadow-lg">
      <Group class="w-full justify-between">
        <Text class="text-sm font-medium">{menuTitle}</Text>
        <Button type="button" variant="ghost" size="icon" aria-label={closeLabel} autofocus={menuOpen} onclick={() => (menuOpen = false)}>
          <X size={18} strokeWidth={2} />
        </Button>
      </Group>
      <List class="flex flex-col gap-1">
        {#each items as item (item.href)}
          <ListItem>
            <Button
              type="button"
              variant={active === item.name ? "secondary" : "ghost"}
              class="w-full justify-start"
              onclick={() => go(item.href)}
            >
              {item.label}
            </Button>
          </ListItem>
        {/each}
      </List>
      {#if project}
        <Separator />
        <List class="flex flex-col gap-1">
          <ListItem>
            <Button type="button" variant="ghost" class="w-full justify-start" onclick={() => { menuOpen = false; project.onExport(); }}>
              {project.exportLabel}
            </Button>
          </ListItem>
          <ListItem>
            <Button type="button" variant="ghost" class="w-full justify-start" onclick={pickProjectFile}>
              {project.importLabel}
            </Button>
          </ListItem>
        </List>
      {/if}
    </Stack>
  </Dialog>
</Block>
