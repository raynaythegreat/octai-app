#!/bin/bash
set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
TAURI_DIR="$ROOT_DIR/src-tauri"
GO_BACKEND_DIR="$ROOT_DIR/go-backend"
GO_BINARY="$TAURI_DIR/binaries/octai-backend-x86_64-unknown-linux-gnu"

cleanup() {
    echo ""
    echo "==> Cleaning up..."
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    lsof -ti:18800 2>/dev/null | xargs kill -9 2>/dev/null || true
    lsof -ti:18790 2>/dev/null | xargs kill -9 2>/dev/null || true
    echo "    Done"
}
trap cleanup EXIT INT TERM

echo "==> [1/5] Building Go backend..."
cd "$GO_BACKEND_DIR"
CGO_ENABLED=0 go build -o "$GO_BINARY" ./cmd/octai-app/
echo "    Done"

echo "==> [2/5] Starting Go backend on :18790..."
"$GO_BINARY" --port 18790 --console &
BACKEND_PID=$!
echo "    Backend PID: $BACKEND_PID"

echo "==> [3/5] Starting frontend dev server on :18800..."
cd "$FRONTEND_DIR"
pnpm --silent install 2>/dev/null
pnpm dev &
FRONTEND_PID=$!

echo "==> [4/5] Waiting for services..."
for i in $(seq 1 30); do
    if curl -s -o /dev/null http://localhost:18800/ 2>/dev/null; then
        echo "    Frontend ready"
        break
    fi
    sleep 1
done

echo "==> [5/5] Starting Tauri..."
cd "$TAURI_DIR"
source "$HOME/.cargo/env" 2>/dev/null || true
export PATH="$HOME/.cargo/bin:$PATH"
cargo tauri dev

cleanup
