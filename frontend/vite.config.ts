import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import pkg from './package.json' with { type: 'json' }

const backendTarget = process.env.VITE_BACKEND_URL ?? 'http://localhost:8080'

export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  plugins: [react()],
  server: {
    proxy: {
      '/api': backendTarget,
      '/auth': backendTarget,
      '/reports': backendTarget,
    },
  },
})
