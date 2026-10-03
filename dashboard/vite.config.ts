import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

// In dev, /api is proxied to the gate's admin listener with the admin token
// attached, mirroring what nginx does in the container.
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  return {
    plugins: [react(), tailwindcss()],
    server: {
      proxy: {
        '/api': {
          target: env.GATE_ADMIN_URL ?? 'http://localhost:8081',
          headers: { Authorization: `Bearer ${env.ADMIN_TOKEN ?? 'dev-admin-token'}` },
        },
      },
    },
  }
})
