import { fileURLToPath, URL } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  resolve: {
    alias: {
      "$ui8kit/ui": fileURLToPath(new URL("./src/kit/ui/index.svelte.ts", import.meta.url)),
      "$ui8kit/utils": fileURLToPath(new URL("./src/kit/utils/index.ts", import.meta.url)),
      $components: fileURLToPath(new URL("./src/components", import.meta.url)),
      $views: fileURLToPath(new URL("./src/views", import.meta.url)),
      $lib: fileURLToPath(new URL("./src/lib", import.meta.url)),
    },
  },
});
