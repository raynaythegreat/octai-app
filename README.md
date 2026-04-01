# OctAi App

OctAi is an all-in-one AI-powered business operations desktop application. Manage AI agents, 20+ LLM providers, 16 messaging channels, and 25+ built-in tools from a beautiful cross-platform desktop app.

## Features

- **Multi-LLM Support**: 20+ AI providers with 60+ models, automatic failover, load balancing
- **16 Messaging Channels**: Telegram, Discord, Slack, WhatsApp, WeChat, and more
- **25+ Built-in Tools**: Shell, filesystem, web search, messaging, and more
- **Multi-Agent System**: Orchestrator, Sales, Support, Research, and custom agents
- **Embedded Terminal**: Full shell access via xterm.js
- **Mobile Access**: Access OctAi from your phone via Tailscale VPN
- **System Tray**: Background operation with quick controls
- **Cross-Platform**: Linux, macOS, Windows

## Prerequisites

- [Rust](https://rustup.rs/) (latest stable)
- [Go](https://go.dev/) 1.24+
- [Node.js](https://nodejs.org/) 24+
- [pnpm](https://pnpm.io/) 10+

### Platform-specific

**Linux (Debian/Ubuntu):**
```bash
sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0
```

**Linux (Fedora):**
```bash
sudo dnf install webkit2gtk4.1 gtk3
```

**macOS:** No additional dependencies

**Windows:** No additional dependencies

## Development

```bash
git clone https://github.com/raynaythegreat/octai-app.git
cd octai-app

make build          # Build Go backend + React frontend
make dev            # Start Tauri in dev mode
```

## Production Build

```bash
make build-app      # Full production build for current platform
make build-all      # Cross-compile Go backend for all platforms
```

Build outputs:
- **Linux**: `.deb`, `.rpm`, `.AppImage`
- **macOS**: `.dmg`, `.app`
- **Windows**: `.msi`, `.exe` (NSIS installer)

## Architecture

```
octai-app/
├── src-tauri/          # Tauri 2.0 Rust shell (window, tray, sidecar management)
├── frontend/           # React 19 + TypeScript + Tailwind v4 + shadcn/ui
├── go-backend/         # Go backend (HTTP API, WebSocket, PTY, Tailscale)
│   ├── cmd/octai-app/  # Go entrypoint
│   ├── pkg/            # Core library packages
│   └── web/backend/    # API route handlers
└── branding/           # OctAi brand assets
```

## License

MIT
