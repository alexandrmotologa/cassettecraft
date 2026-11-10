import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  base: './',
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8585',
      '/ca.crt': 'http://localhost:8585'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
