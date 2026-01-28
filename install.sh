#!/bin/bash
set -e

# Repo information
OWNER="themobileprof"
REPO="clio"
BIN_NAME="clio"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)      echo "Unsupported Arch: $ARCH"; exit 1 ;;
esac

echo "Detected Platform: $OS/$ARCH"

# Determine install path (Termux support)
if [ -n "$TERMUX_VERSION" ]; then
    INSTALL_DIR="$PREFIX/bin"
else
    INSTALL_DIR="/usr/local/bin"
    # Fallback to local user bin if no root access?
    if [ ! -w "$INSTALL_DIR" ]; then
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    fi
fi

# Fetch latest release URL (using GitHub API or assume 'latest' redirect)
# For simplicity, using 'latest' API to find asset name
LATEST_URL="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
echo "Fetching latest release version..."

# Helper to install man
install_man() {
    echo "'man' command not found. Attempting to install..."
    if [ -n "$TERMUX_VERSION" ]; then
        pkg install -y man
    elif command -v apt-get >/dev/null 2>&1; then
        SUDO=""
        [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1 && SUDO="sudo"
        echo "Detected apt-get. Installing man-db..."
        $SUDO apt-get update -qq && $SUDO apt-get install -y man-db
    elif command -v apk >/dev/null 2>&1; then
        SUDO=""
        [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1 && SUDO="sudo"
        echo "Detected apk. Installing man-db..."
        $SUDO apk add man-db man-pages
    elif command -v dnf >/dev/null 2>&1; then
        SUDO=""
        [ "$(id -u)" -ne 0 ] && command -v sudo >/dev/null 2>&1 && SUDO="sudo"
        echo "Detected dnf. Installing man-db..."
        $SUDO dnf install -y man-db
    else
        echo "Warning: Could not detect a supported package manager (pkg, apt, apk, dnf). Please install 'man' manually."
    fi
}

# Check for man and install if missing
if ! command -v man >/dev/null 2>&1; then
    install_man
fi

# Simple grep to find the regular download url for the right binary
# Asset name format: clio-{os}-{arch} (from release.yml)
ASSET_NAME="${BIN_NAME}-${OS}-${ARCH}"

# Construct download URL (assuming public repo)
DOWNLOAD_URL="https://github.com/$OWNER/$REPO/releases/latest/download/$ASSET_NAME"

echo "Downloading $DOWNLOAD_URL..."
curl -L -o "$BIN_NAME" "$DOWNLOAD_URL"

chmod +x "$BIN_NAME"
mv "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"

echo "Successfully installed $BIN_NAME to $INSTALL_DIR"
