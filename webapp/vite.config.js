import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  base: './',
  plugins: [
    vue(),
    tailwindcss(),
  ],
  test: {
    globals: true,
    environment: 'jsdom',
    exclude: ['e2e/**', 'node_modules/**'],
  },
  server: {
    proxy: {
      '/config': 'http://localhost:8080',
      '/version': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/upload': 'http://localhost:8080',
      '/file': 'http://localhost:8080',
      '/resumable': 'http://localhost:8080',
      '/stream': 'http://localhost:8080',
      '/archive': 'http://localhost:8080',
      '/qrcode': 'http://localhost:8080',
      '/auth': 'http://localhost:8080',
      '/me': 'http://localhost:8080',
      '/user': 'http://localhost:8080',
      '/users': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
      '/stats': 'http://localhost:8080',
    }
  }
})
