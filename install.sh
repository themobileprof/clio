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

# Determine install path (Termux support)
USE_SUDO=""
if [ -n "$TERMUX_VERSION" ]; then
    INSTALL_DIR="$PREFIX/bin"
else
    INSTALL_DIR="/usr/local/bin"
    if [ ! -w "$INSTALL_DIR" ]; then
        echo "Note: $INSTALL_DIR is not writable."
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        echo "Installing to $INSTALL_DIR instead."
    fi
fi

echo "Detected Platform: $OS/$ARCH"

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
        echo "Detected apt-get. Please install 'man-db' manually (or run as root)."
        # check if we are root, if so, install
        if [ "$(id -u)" -eq 0 ]; then
             apt-get update -qq && apt-get install -y man-db
        fi
    elif command -v apk >/dev/null 2>&1; then
        echo "Detected apk. Please install 'man-db' manually (or run as root)."
        if [ "$(id -u)" -eq 0 ]; then
             apk add man-db man-pages
        fi
    elif command -v dnf >/dev/null 2>&1; then
        echo "Detected dnf. Please install 'man-db' manually (or run as root)."
        if [ "$(id -u)" -eq 0 ]; then
             dnf install -y man-db
        fi
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
if command -v curl >/dev/null 2>&1; then
    curl -fsL -o "$BIN_NAME" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$BIN_NAME" "$DOWNLOAD_URL"
else
    echo "Error: Neither curl nor wget found."
    exit 1
fi

if [ ! -f "$BIN_NAME" ] || [ ! -s "$BIN_NAME" ]; then
    echo "Error: Failed to download binary. Check if the release/asset exists."
    exit 1
fi

# Verify it's not a text file (simple heuristic: check size)
# Go binaries are usually > 1MB. Let's be conservative and say > 100KB (102400 bytes).
# "Not Found" is 9 bytes.
FILE_SIZE=$(wc -c < "$BIN_NAME" | tr -d '[:space:]')
if [ "$FILE_SIZE" -lt 102400 ]; then
    echo "Error: Downloaded file is too small ($FILE_SIZE bytes). Likely an error message or invalid binary."
    cat "$BIN_NAME" # Print content to help debug
    rm "$BIN_NAME"
    exit 1
fi

chmod +x "$BIN_NAME"
mv "$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"

# Create .clio directory structure
CLIO_DIR="$HOME/.clio"
mkdir -p "$CLIO_DIR/modules"

echo "✅ Successfully installed $BIN_NAME to $INSTALL_DIR"

# Check PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "⚠️  Warning: $INSTALL_DIR is not in your PATH. You may need to add it:"
       echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
       ;;
esac

echo ""
echo "To uninstall later, run:"
echo "  curl -sfL https://raw.githubusercontent.com/$OWNER/$REPO/main/uninstall.sh | bash"
echo ""
