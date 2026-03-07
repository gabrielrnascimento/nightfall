# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Install dependencies
pnpm install

# Dev mode (TypeScript watch + live-server on port 3000)
pnpm dev

# Build once
pnpm build
```

## Architecture

Single-file TypeScript app (`app.ts`) compiled to `dist/`. No framework, no bundler — `tsc` outputs JS directly, and `index.html` loads it.

Connects to the backend WebSocket at `ws://127.0.0.1:3001` (default). The backend only accepts connections from `http://127.0.0.1:3000` or `http://localhost:3000`, so the live-server must run on port 3000.

See `docs/WEBSOCKET_PROTOCOL.md` (in repo root) for the full message spec.
