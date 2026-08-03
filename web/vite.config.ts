import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const BACKEND = process.env.VITE_BACKEND_URL ?? 'http://localhost:8282'

// Both the REST API and the WebSocket are proxied to the Go backend so the
// browser sees a single origin. That keeps development same-origin; the backend
// separately sets CORS headers and accepts a ?token= query parameter, which is
// what lets the app also run deployed apart from the API.
//
// changeOrigin stays false deliberately. The WebSocket library authorises a
// handshake by comparing the browser's Origin header against the request's Host
// header, and rewriting Host to the backend breaks that match — the handshake is
// then rejected as cross-origin unless the backend happens to allowlist
// localhost:5173. Forwarding the original Host keeps the request genuinely
// same-origin, so this works against any backend port with no server config.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: BACKEND, changeOrigin: false, ws: true },
    },
  },
})
