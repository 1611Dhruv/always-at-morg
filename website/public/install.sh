#!/bin/bash
set -e

# Always at Morg - Installation Script
# Usage: curl -fsSL https://always-at-morg.bid/install.sh | bash

VERSION="${VERSION:-latest}"
BASE_URL="https://always-at-morg.bid/releases"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}"
echo "╔═══════════════════════════════════════╗"
echo "║    Always at Morg - Installer         ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        CYGWIN*|MINGW*|MSYS*) os="windows";;
        *)
            echo -e "${RED}Error: Unsupported operating system$(uname -s)${NC}"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64";;
        arm64|aarch64)  arch="arm64";;
        armv7l)         arch="arm";;
        *)
            echo -e "${RED}Error: Unsupported architecture $(uname -m)${NC}"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Download and install
install_binary() {
    local platform=$(detect_platform)
    local binary_name="always-at-morg-${platform}"

    if [ "$platform" = "windows_amd64" ]; then
        binary_name="${binary_name}.exe"
    fi

    echo -e "${GREEN}Detected platform: ${platform}${NC}"
    echo -e "${GREEN}Install directory: ${INSTALL_DIR}${NC}"
    echo ""

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Download URL
    local download_url="${BASE_URL}/${binary_name}"

    echo -e "${YELLOW}Downloading from: ${download_url}${NC}"

    # Download binary
    if command -v curl &> /dev/null; then
        curl -fsSL "$download_url" -o "$INSTALL_DIR/morg"
    elif command -v wget &> /dev/null; then
        wget -q "$download_url" -O "$INSTALL_DIR/morg"
    else
        echo -e "${RED}Error: Neither curl nor wget found. Please install one of them.${NC}"
        exit 1
    fi

    # Make executable
    chmod +x "$INSTALL_DIR/morg"

    echo ""
    echo -e "${GREEN}✓ Successfully installed Always at Morg!${NC}"
    echo ""

    # Check if install dir is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}Note: $INSTALL_DIR is not in your PATH${NC}"
        echo "Add this line to your ~/.bashrc, ~/.zshrc, or ~/.profile:"
        echo ""
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
        echo "Then run: source ~/.bashrc (or ~/.zshrc)"
        echo ""
        echo "Or run directly: $INSTALL_DIR/morg"
    else
        echo "Run the game with: morg"
    fi

    echo ""
    echo "To connect to a server, use: morg <server-url>"
    echo "Example: morg ws://localhost:8080/ws"
}

# Main installation
install_binary
