#!/bin/bash
set -e

echo "Installing Always at Morg..."

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    linux)   OS="linux" ;;
    darwin)  OS="darwin" ;;
    *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    x86_64|amd64)    ARCH="amd64" ;;
    arm64|aarch64)   ARCH="arm64" ;;
    armv7l)          ARCH="arm" ;;
    *)               echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

BINARY="always-at-morg-${OS}_${ARCH}"
URL="https://web.always-at-morg.bid/releases/${BINARY}"
INSTALL_DIR="$HOME/.local/bin"

echo "Downloading $BINARY..."
mkdir -p "$INSTALL_DIR"

if command -v curl &> /dev/null; then
    curl -fsSL "$URL" -o "$INSTALL_DIR/morg"
elif command -v wget &> /dev/null; then
    wget -q "$URL" -O "$INSTALL_DIR/morg"
else
    echo "Error: curl or wget required"
    exit 1
fi

chmod +x "$INSTALL_DIR/morg"

echo "✓ Installed to $INSTALL_DIR/morg"
echo ""

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Add to PATH:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

echo "Run with: morg"
