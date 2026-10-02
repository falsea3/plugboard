/// <reference types="vitest/config" />
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    svelte(),
    {
      name: 'keep-wails-dist-placeholder',
      closeBundle() {
        writeFileSync(resolve('dist/.gitkeep'), '');
      }
    }
  ],
  clearScreen: false,
  test: {
    include: ['src/**/*.test.ts'],
  },
  server: {
    host: '127.0.0.1',
    strictPort: false
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/node_modules/@codemirror/') || id.includes('/node_modules/codemirror/') || id.includes('/node_modules/@lezer/')) return 'editor';
        }
      }
    }
  }
});
