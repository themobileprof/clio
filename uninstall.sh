#!/bin/bash
set -e

BIN_NAME="clio"

# curl | bash leaves stdin on the script. A normal read then consumes the
# next line of this file, which is not "y", and uninstall always cancels.
# Read the answer from the terminal instead.
confirm() {
    local prompt=$1 answer
    if [ ! -r /dev/tty ]; then
        echo "❌ No terminal available. Save this file and run: bash uninstall.sh"
        exit 1
    fi
    if ! read -r -p "$prompt" answer </dev/tty; then
        echo
        echo "❌ Uninstall cancelled"
        exit 0
    fi
    answer=$(printf '%s' "$answer" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')
    [ "$answer" = "y" ] || [ "$answer" = "yes" ]
}

echo "╔══════════════════════════════════════════════════════════╗"
echo "║       Clio Uninstaller                                   ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# Determine install path (same logic as install.sh)
if [ -n "$TERMUX_VERSION" ]; then
    INSTALL_DIR="$PREFIX/bin"
else
    INSTALL_DIR="/usr/local/bin"
    if [ ! -f "$INSTALL_DIR/$BIN_NAME" ]; then
        # Check alternative location
        INSTALL_DIR="$HOME/.local/bin"
    fi
fi

BINARY_PATH="$INSTALL_DIR/$BIN_NAME"
CLIO_DIR="$HOME/.clio"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ $BIN_NAME not found at $BINARY_PATH"
    echo ""
    echo "Checked locations:"
    echo "  • $PREFIX/bin/$BIN_NAME" 2>/dev/null || true
    echo "  • /usr/local/bin/$BIN_NAME"
    echo "  • $HOME/.local/bin/$BIN_NAME"
    echo ""
    
    # Still offer to remove config directory
    if [ -d "$CLIO_DIR" ]; then
        echo "Configuration directory exists: $CLIO_DIR"
        echo "Removing it also removes the database, Pico CSS, and downloaded modules."
        if confirm "Remove configuration directory? (y/N): "; then
            rm -rf "$CLIO_DIR"
            echo "✅ Removed $CLIO_DIR"
        else
            echo "⏭️  Kept configuration directory"
        fi
    fi
    exit 0
fi

echo "Found Clio installation:"
echo "  • Binary: $BINARY_PATH"
echo "  • Size: $(du -h "$BINARY_PATH" | cut -f1)"

if [ -d "$CLIO_DIR" ]; then
    CLIO_SIZE=$(du -sh "$CLIO_DIR" 2>/dev/null | cut -f1 || echo "unknown")
    echo "  • Config: $CLIO_DIR ($CLIO_SIZE)"
    
    # Count modules
    MODULE_COUNT=$(find "$CLIO_DIR/modules" -name "*.yaml" 2>/dev/null | wc -l || echo "0")
    if [ "$MODULE_COUNT" -gt 0 ]; then
        echo "  • Modules: $MODULE_COUNT YAML module(s)"
    fi
fi

# The database lives inside ~/.clio. ~/.clio.db is an old unused path.
if [ -f "$CLIO_DIR/clio.db" ]; then
    DB_SIZE=$(du -h "$CLIO_DIR/clio.db" | cut -f1)
    echo "  • Database: $CLIO_DIR/clio.db ($DB_SIZE)"
fi
if [ -f "$HOME/.clio.db" ]; then
    DB_SIZE=$(du -h "$HOME/.clio.db" | cut -f1)
    echo "  • Old database file: $HOME/.clio.db ($DB_SIZE)"
fi

echo ""
if ! confirm "Remove Clio binary? (y/N): "; then
    echo "❌ Uninstall cancelled"
    exit 0
fi

# Remove binary
echo "Removing binary..."
rm -f "$BINARY_PATH" "$INSTALL_DIR/clio.bin" "$INSTALL_DIR/clio-run-module"
echo "✅ Removed $BINARY_PATH"

# Ask about config directory
if [ -d "$CLIO_DIR" ]; then
    echo ""
    echo "Removing ~/.clio also removes the database (~/.clio/clio.db), Pico CSS, and downloaded modules."
    if confirm "Remove configuration directory ~/.clio? (y/N): "; then
        rm -rf "$CLIO_DIR"
        echo "✅ Removed $CLIO_DIR"
    else
        echo "⏭️  Kept $CLIO_DIR (modules preserved)"
    fi
fi

# Ask about database
if [ -f "$HOME/.clio.db" ]; then
    echo ""
    if confirm "Remove old database file ~/.clio.db? (y/N): "; then
        rm -f "$HOME/.clio.db"
        echo "✅ Removed $HOME/.clio.db"
    else
        echo "⏭️  Kept $HOME/.clio.db"
    fi
fi

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  ✅ Clio uninstalled successfully                        ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "Thanks for using Clio! 👋"
echo ""
