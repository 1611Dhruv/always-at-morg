#!/bin/bash
set -e

# Build script for creating multi-platform releases
# Usage: ./build-releases.sh [version]

VERSION="${1:-latest}"
OUTPUT_DIR="website/public/releases"

echo "Building Always at Morg - Version: ${VERSION}"
echo "=========================================="

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Platforms to build for
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "windows/amd64"
)

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"

    OUTPUT_NAME="always-at-morg-${GOOS}_${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo "Building for ${GOOS}/${GOARCH}..."

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.Version=${VERSION}" \
        -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
        cmd/client/main.go

    echo "✓ Created ${OUTPUT_NAME}"
done

echo ""
echo "✓ Build complete! Binaries created in ${OUTPUT_DIR}/"
echo ""
echo "Binaries will be served from: https://always-at-morg.bid/releases/"
echo ""
echo "Next steps:"
echo "  1. Build the website: cd website && npm run build"
echo "  2. Deploy the website/build/ folder to always-at-morg.bid"
echo "  3. Install script will download from: https://always-at-morg.bid/releases/"
