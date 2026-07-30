import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', '')
  const target = env.VITE_DEV_API_ORIGIN || 'http://localhost:8080'

  return {
    plugins: [svelte()],
    server: {
      host: '0.0.0.0',
      port: 3000,
      proxy: {
        '/api': { target, changeOrigin: true },
        // Keep this trailing slash: a bare '/s' prefix would proxy Vite's
        // own '/src/*' modules and leave the development page blank.
        '^/s/': { target, changeOrigin: true },
      },
    },
  }
})
