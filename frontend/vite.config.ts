import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// When running locally (outside Docker), the API is at localhost:8080.
// When running inside Docker Compose, the API container is named 'api'.
const apiTarget = process.env.API_HOST
  ? `http://${process.env.API_HOST}:8080`
  : 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    watch: {
      usePolling: true,   // needed for file-change detection inside Docker on Windows
      interval: 300,
    },
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
})
