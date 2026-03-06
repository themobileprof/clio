#!/bin/bash
# Build script for Termux/Android to avoid SIGSYS errors
# This ensures proper ARM architecture and disables newer syscalls

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

# Build with specific flags to avoid newer syscalls
# - Disable CGO to avoid libc dependencies
# - Set GOOS=linux explicitly
#- Disable newer syscalls via build tags
export CGO_ENABLED=0
export GOOS=linux
export GOARCH="$GOARCH"

# Build with tags to disable problematic features
# The syscall compatibility is handled by safeexec package
go build -v \
    -ldflags="-s -w" \
    -o clio \
    ./cmd/clio

# Verify it built
if [ ! -f "clio" ]; then
    echo "Error: Build failed"
    exit 1
fi

echo "✅ Build successful: $(pwd)/clio"
echo "   Architecture: $GOARCH"
echo "   Size: $(du -h clio | cut -f1)"
echo ""
echo "To install:"
echo "  cp clio \$PREFIX/bin/ && chmod +x \$PREFIX/bin/clio"
