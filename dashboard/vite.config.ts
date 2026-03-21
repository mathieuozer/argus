import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { mockApiPlugin } from './mock-api';

const useMockApi = !process.env.ARGUS_API_URL;

export default defineConfig({
  plugins: [
    react(),
    ...(useMockApi ? [mockApiPlugin()] : []),
  ],
  server: {
    port: 5173,
    ...(useMockApi
      ? {}
      : {
          proxy: {
            '/api': {
              target: process.env.ARGUS_API_URL || 'http://localhost:8080',
              changeOrigin: true,
            },
            '/ws': {
              target: process.env.ARGUS_WS_URL || 'ws://localhost:8080',
              ws: true,
            },
          },
        }),
  },
});
