// Development proxy server for ngrok tunneling (npm run dev:proxy).
// Mirrors the proxy routes in vite.config.ts — keep both in sync.
const express = require('express');
const httpProxy = require('http-proxy');

const app = express();
const PORT = process.env.PROXY_PORT || 3000;

// Create proxy instances
const viteProxy = httpProxy.createProxyServer({ target: 'http://localhost:5173', ws: true, changeOrigin: true });
const supabaseProxy = httpProxy.createProxyServer({ target: 'http://localhost:54321', changeOrigin: true });
const mailpitProxy = httpProxy.createProxyServer({ target: 'http://localhost:54324', changeOrigin: true });
const goProxy = httpProxy.createProxyServer({ target: 'http://localhost:8080', changeOrigin: true });

// Handle proxy errors safely. During a WebSocket upgrade the third argument
// is a net.Socket (no .writeHead), not an http.ServerResponse — calling
// Express helpers on it would crash the process.
function handleProxyError(label) {
  return (err, _req, resOrSocket) => {
    console.error(`${label} proxy error:`, err.message);
    if (typeof resOrSocket.writeHead === 'function') {
      if (!resOrSocket.headersSent) {
        resOrSocket.writeHead(502, { 'Content-Type': 'application/json' });
        resOrSocket.end(JSON.stringify({ error: `${label} proxy error` }));
      }
    } else {
      // WebSocket socket — just tear it down.
      resOrSocket.destroy();
    }
  };
}

viteProxy.on('error', handleProxyError('Vite'));
supabaseProxy.on('error', handleProxyError('Supabase'));
mailpitProxy.on('error', handleProxyError('Mailpit'));
goProxy.on('error', handleProxyError('Go API'));

// Go API routes — must come before the Vite catch-all.
// Express strips the matched prefix from req.url, so restore it before proxying.
app.use('/api', (req, res) => {
  req.url = '/api' + req.url;
  console.log(`→ Proxying ${req.method} ${req.url} to Go server`);
  goProxy.web(req, res);
});

// Invite endpoint lives outside /api — it's a user-facing onboarding URL.
app.use('/invite', (req, res) => {
  req.url = '/invite' + req.url;
  console.log(`→ Proxying ${req.method} ${req.url} to Go server`);
  goProxy.web(req, res);
});

// Mailpit API routes (for dev auto-fill OTP codes)
app.use('/mailpit', (req, res) => {
  // Remove /mailpit prefix
  req.url = req.url.replace(/^\/mailpit/, '');
  console.log(`→ Proxying ${req.method} /mailpit${req.url} to Mailpit`);
  mailpitProxy.web(req, res);
});

// Supabase API routes
app.use('/supabase', (req, res) => {
  // Remove /supabase prefix
  req.url = req.url.replace(/^\/supabase/, '');
  console.log(`→ Proxying ${req.method} /supabase${req.url} to Supabase`);
  supabaseProxy.web(req, res);
});

// Everything else goes to Vite
app.use('/', (req, res) => {
  console.log(`→ Proxying ${req.method} ${req.url} to Vite`);
  viteProxy.web(req, res);
});

const server = app.listen(PORT, () => {
  console.log(`\n🚀 Development proxy running on http://localhost:${PORT}`);
  console.log(`   Frontend: proxied from http://127.0.0.1:5173`);
  console.log(`   Supabase API: /supabase/* → http://127.0.0.1:54321`);
  console.log(`   Mailpit API: /mailpit/* → http://127.0.0.1:54324`);
  console.log(`   Go API: /api/* → http://127.0.0.1:8080`);
  console.log(`   Invite: /invite/* → http://127.0.0.1:8080`);
  console.log(`\n   Supabase URL is auto-resolved via window.location.origin/supabase\n`);
});

// Handle WebSocket upgrades for Vite HMR
server.on('upgrade', (req, socket, head) => {
  console.log(`→ Proxying WebSocket ${req.url} to Vite`);
  viteProxy.ws(req, socket, head);
});
