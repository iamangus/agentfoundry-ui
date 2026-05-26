import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/chat': 'http://localhost:8080',
      '/agents': 'http://localhost:8080',
      '/tools': 'http://localhost:8080',
      '/api/keys': 'http://localhost:8080',
      '/api/v1': 'http://localhost:8080',
      '/auth': 'http://localhost:8080',
    },
  },
})
