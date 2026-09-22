<script lang="ts">
  import { Block, Link } from "$ui8kit/ui";
  import AppHeader from "$components/AppHeader.svelte";
  import AppMain from "$components/AppMain.svelte";
  import HomeView from "$views/HomeView.svelte";
  import SlicesView from "$views/SlicesView.svelte";
  import PlaygroundView from "$views/PlaygroundView.svelte";
  import { copy } from "$lib/copy";
  import { createPlayground } from "$lib/playground.svelte";
  import { parseRoute, createRouter } from "$lib/route.svelte";
  import { slices } from "$lib/slices";
  import { createTheme } from "$lib/theme.svelte";

  const theme = createTheme();
  const router = createRouter();
  const initial = parseRoute(window.location.pathname);
  const board = createPlayground(initial.name === "playground" ? initial.sliceId : slices[0].id);

  $effect(() => {
    const id = window.setInterval(() => board.tick(), 250);
    return () => window.clearInterval(id);
  });

  $effect(() => {
    const route = router.route;
    if (route.name === "playground") board.load(route.sliceId);
  });

  const active = $derived(router.route.name);

  function openSlice(id: string) {
    board.forceLoad(id);
    router.navigate(`/playground/${id}`);
  }

  async function importFile(file: File) {
    const text = await file.text();
    try {
      const value = JSON.parse(text) as unknown;
      if (!board.importProject(value)) return;
      router.navigate(`/playground/${board.sliceId}`);
    } catch {
      return;
    }
  }

  function exportFile() {
    const payload = JSON.stringify(board.exportProject(), null, 2);
    const blob = new Blob([payload], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "lex-flow-project.json";
    link.click();
    URL.revokeObjectURL(url);
  }
</script>

<Block class="flex min-h-screen flex-col bg-background font-sans text-foreground">
  <Link href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-background focus:p-2">
    {copy.skip}
  </Link>
  <AppHeader
    logo={copy.logo}
    active={active}
    items={[
      { name: "home", href: "/", label: copy.nav.home },
      { name: "slices", href: "/slices", label: copy.nav.slices },
      { name: "playground", href: "/playground", label: copy.nav.playground },
    ]}
    themeMode={theme.mode}
    themeLight={copy.themeLight}
    themeDark={copy.themeDark}
    projectLabel={copy.project}
    importLabel={copy.importProject}
    exportLabel={copy.exportProject}
    fileLabel={copy.fileInput}
    onNavigate={router.navigate}
    onTheme={theme.toggle}
    onImport={importFile}
    onExport={exportFile}
  />
  <AppMain class="p-4">
    {#if router.route.name === "slices"}
      <SlicesView
        title={copy.slices.title}
        lead={copy.slices.lead}
        openLabel={copy.slices.open}
        {slices}
        onOpen={openSlice}
      />
    {:else if router.route.name === "playground"}
      <PlaygroundView
        heading={copy.playground.title}
        jsonLabel={copy.playground.json}
        runLabel={copy.playground.run}
        locked={copy.playground.locked}
        trapLabel={copy.honeypot}
        themeMode={theme.mode}
        {board}
      />
    {:else}
      <HomeView
        title={copy.home.title}
        lead={copy.home.lead}
        lines={copy.home.lines}
        body={copy.home.body}
        github={copy.home.github}
        playground={copy.home.playground}
        onPlayground={() => openSlice(slices[0].id)}
      />
    {/if}
  </AppMain>
</Block>
