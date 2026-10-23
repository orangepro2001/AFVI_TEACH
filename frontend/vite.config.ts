import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath } from 'node:url'

// Dev server proxies the host endpoints to the Go dev host (cmd/*-wails -web 127.0.0.1:47700).
const backend = process.env.AFVI_BACKEND || 'http://127.0.0.1:47700'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: [
      // win32-vue is vendored (frontend/vendor/win32-vue, built from ../win32-theme-for-wails)
      { find: '@win32-vue/win32/styles', replacement: fileURLToPath(new URL('./vendor/win32-vue/src/styles', import.meta.url)) },
      { find: '@win32-vue/win32/components.css', replacement: fileURLToPath(new URL('./vendor/win32-vue/dist/win32.css', import.meta.url)) },
      { find: /^@win32-vue\/win32$/, replacement: fileURLToPath(new URL('./vendor/win32-vue/dist/win32-vue.js', import.meta.url)) },
    ],
  },
  base: './',
  build: { outDir: 'dist', emptyOutDir: true, target: 'es2020' },
  server: {
    port: 5173,
    proxy: { '/cmd': backend, '/poll': backend, '/img': backend },
  },
  test: { include: ['src/**/*.spec.ts'], environment: 'node' },
})
