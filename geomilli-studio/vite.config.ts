import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [
    react(),
    {
      name: 'docs-redirect',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/docs') {
            res.writeHead(301, { Location: '/docs/' });
            res.end();
            return;
          }
          next();
        });
      },
    },
  ],
  server: {
    port: Number(process.env.STUDIO_PORT || process.env.PORT || 3000),
    proxy: {
      '/v1': {
        target: process.env.VITE_API_PROXY_TARGET || `http://localhost:${process.env.API_PORT || process.env.PORT || 8080}`,
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          maplibre: ['maplibre-gl'],
          react: ['react', 'react-dom'],
          lucide: ['lucide-react'],
        },
      },
    },
    chunkSizeWarningLimit: 1200,
  },
});
