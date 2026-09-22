<script lang="ts">
  import { Block, Link } from "$ui8kit/ui";
  import AppHeader from "$components/AppHeader.svelte";
  import AppMain from "$components/AppMain.svelte";
  import HomeView from "$views/HomeView.svelte";
  import { copy } from "$lib/copy";
</script>

<Block class="flex min-h-screen flex-col bg-background font-sans text-foreground">
  <Link href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-background focus:p-2">
    {copy.skip}
  </Link>
  <AppHeader title={copy.title} />
  <AppMain>
    <HomeView title={copy.homeTitle} body={copy.homeBody} />
  </AppMain>
</Block>
