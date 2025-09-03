// @ts-check
import { defineConfig } from "astro/config";

// https://astro.build/config
export default defineConfig({
  site: "https://wowsimstats.com",
  vite: {
    server: {
      // Add headers for SQLite database support
      headers: {
        'Accept-Ranges': 'bytes'
      }
    }
  }
});
