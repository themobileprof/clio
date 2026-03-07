#!/bin/bash
# Build script for Termux/Android
# Ensures proper ARM architecture and compatibility

set -e

echo "Building Clio for Termux/Android..."

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    aarch64|arm64) GOARCH="arm64" ;;
    armv7l|armv8l) GOARCH="arm" ;;
    x86_64) GOARCH="amd64" ;;
    i686) GOARCH="386" ;;
    *)
        echo "Warning: Unknown architecture $ARCH, defaulting to arm64"
        GOARCH="arm64"
        ;;
esac

echo "Target architecture: $GOARCH"
echo ""
echo "Note: For module execution on Termux, use the 'clio-run-module' script"
echo "      (Installed automatically by install.sh alongside the binary)"
echo "      This avoids Android seccomp syscall restrictions (pidfd_open)"
echo ""

# Build with Android compatibility
export CGO_ENABLED=0
export GOOS=linux
export GOARCH="$GOARCH"

# Build
go build -v \
    -ldflags="-s -w" \
    -o clio \
    ./cmd/clio

# Verify it built
if [ ! -f "clio" ]; then
    echo "Error: Build failed"
    exit 1
fi

echo ""
echo "✅ Build successful: $(pwd)/clio"
echo "   Architecture: $GOARCH"
echo "   Size: $(du -h clio | cut -f1)"
echo ""
echo "To install:"
echo "  cp clio \$PREFIX/bin/ && chmod +x \$PREFIX/bin/clio"
