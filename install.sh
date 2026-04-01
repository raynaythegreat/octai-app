#!/bin/bash
set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_BIN_DIR="$HOME/.local/bin"
INSTALL_APP_DIR="$HOME/.local/share/applications"
INSTALL_ICON_DIR="$HOME/.local/share/icons"
TAURI_DIR="$ROOT_DIR/src-tauri"
RELEASE_DIR="$TAURI_DIR/target/release"
APP_BINARY="octai-app"
BACKEND_BINARY="octai-backend-x86_64-unknown-linux-gnu"

echo "==> Building production app..."
cd "$ROOT_DIR"
make build-app
echo "    Done"

echo "==> Installing binary and sidecar to $INSTALL_BIN_DIR/..."
mkdir -p "$INSTALL_BIN_DIR"
cp "$RELEASE_DIR/$APP_BINARY" "$INSTALL_BIN_DIR/$APP_BINARY"
chmod +x "$INSTALL_BIN_DIR/$APP_BINARY"
if [ -f "$RELEASE_DIR/octai-backend" ]; then
    cp "$RELEASE_DIR/octai-backend" "$INSTALL_BIN_DIR/octai-backend"
    chmod +x "$INSTALL_BIN_DIR/octai-backend"
    echo "    Sidecar: $INSTALL_BIN_DIR/octai-backend"
fi
echo "    Done"

echo "==> Installing .desktop file..."
mkdir -p "$INSTALL_APP_DIR"
cat > "$INSTALL_APP_DIR/octai-app.desktop" <<DESKTOP
[Desktop Entry]
Name=OctAi
Comment=AI Agent Desktop App
Exec=$INSTALL_BIN_DIR/$APP_BINARY
Icon=$INSTALL_ICON_DIR/octai-app.png
Terminal=false
Type=Application
Categories=Development;Utility;
StartupNotify=true
DESKTOP
echo "    Done"

echo "==> Installing icon..."
mkdir -p "$INSTALL_ICON_DIR"
cp "$TAURI_DIR/icons/128x128.png" "$INSTALL_ICON_DIR/octai-app.png"
echo "    Done"

echo "==> Updating desktop database..."
update-desktop-database "$INSTALL_APP_DIR" 2>/dev/null || true
echo "    Done"

echo ""
echo "==> Installation complete!"
echo "    Binary:  $INSTALL_BIN_DIR/$APP_BINARY"
echo "    Desktop: $INSTALL_APP_DIR/octai-app.desktop"
echo "    Icon:    $INSTALL_ICON_DIR/octai-app.png"
