# Remote Development Setup

This guide explains how to test the app remotely (e.g., on your phone) using ngrok or similar tunneling services.

## The Problem

When running `npm run dev`, Vite serves the frontend on `localhost:5173` and the app connects to Supabase at `http://127.0.0.1:54321`. If you tunnel only the frontend through ngrok, remote devices can't reach the Supabase API (because `127.0.0.1` is localhost on the Mac, not on the remote device).

## Solution: Development Proxy

The `dev-proxy.cjs` script creates a single proxy server that forwards:
- Frontend requests → Vite dev server (port 5173)
- `/api/*` requests → Go API server (port 8080)
- `/supabase/*` requests → Supabase API (port 54321)
- `/mailpit/*` requests → Mailpit (port 54324)

This allows you to use **one ngrok tunnel** for all services. In dev mode, the Supabase URL is auto-resolved from `window.location.origin/supabase`, so no env override is needed.

## Setup Steps

### 1. Start Supabase locally
```bash
supabase start
```

### 2. Update your `.env` file

Get the publishable key:
```bash
supabase status -o env | grep ANON_KEY
```

Then set in your `.env` file:
```bash
VITE_SUPABASE_PUBLISHABLE_KEY=<your-publishable-key>
```

> **Note:** `VITE_SUPABASE_URL` is not needed for local dev. The app auto-resolves Supabase via `window.location.origin/supabase` in dev mode.

### 3. Start the development servers

In one terminal, start Vite:
```bash
npm run dev
```

In another terminal, start the proxy:
```bash
npm run dev:proxy
```

The proxy runs on port 3000 by default.

### 4. Create an ngrok tunnel

Tunnel the **proxy** (not Vite directly):
```bash
ngrok http 3000
```

Copy the ngrok URL (e.g., `https://abc123.ngrok-free.dev`).

### 5. Test remotely

Open the ngrok URL in your phone's browser. The app auto-resolves Supabase via `window.location.origin/supabase`, so no `VITE_SUPABASE_URL` override is needed — requests go through the proxy automatically.

## How It Works

```
Remote Device
    ↓
https://abc123.ngrok-free.dev
    ↓
ngrok tunnel
    ↓
localhost:3000 (dev-proxy.cjs)
    ├─ / → localhost:5173 (Vite)
    └─ /supabase → localhost:54321 (Supabase)
```

## Troubleshooting

**"Host not allowed" error from Vite:**
- Check that `vite.config.ts` includes `allowedHosts: [".ngrok-free.dev"]`

**Auth errors / CORS errors:**
- Verify the proxy is running on port 3000
- Check that ngrok is tunneling port 3000 (the proxy), not 5173 (Vite directly)

**Changes not reflecting:**
- Vite hot reload works through the proxy via WebSocket proxying
- If HMR breaks, restart both the Vite dev server and the proxy
