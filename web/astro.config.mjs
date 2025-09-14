// @ts-check
import { defineConfig } from "astro/config";
import vercel from "@astrojs/vercel";

// https://astro.build/config
export default defineConfig({
  site: "https://wowsimstats.com",
  output: "server", // Server mode for dynamic SPA routes
  adapter: vercel(),

  vite: {
    server: {
      // Headers for development server
      headers: {
        "Accept-Ranges": "bytes",
      },
    },
  },

  // Pure SPA with static JSON files served alongside SSR
  integrations: [],
});
