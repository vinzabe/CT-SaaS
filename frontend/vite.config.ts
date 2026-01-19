import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 7843,
    proxy: {
      '/api': {
        target: 'http://localhost:7842',
        changeOrigin: true,
      },
    },
  },
})
