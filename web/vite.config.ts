import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

const backend = 'http://localhost:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': backend, '/healthz': backend },
  },
  test: { environment: 'node', include: ['src/**/*.test.ts'] },
});
