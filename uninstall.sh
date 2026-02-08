#!/bin/bash
set -e

BIN_NAME="clio"

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
        read -p "Remove configuration directory? (y/N): " -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
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

# Check if database exists
if [ -f "$HOME/.clio.db" ]; then
    DB_SIZE=$(du -h "$HOME/.clio.db" | cut -f1)
    echo "  • Database: $HOME/.clio.db ($DB_SIZE)"
fi

echo ""
read -p "Remove Clio binary? (y/N): " -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Uninstall cancelled"
    exit 0
fi

# Remove binary
echo "Removing binary..."
rm -f "$BINARY_PATH"
echo "✅ Removed $BINARY_PATH"

# Ask about config directory
if [ -d "$CLIO_DIR" ]; then
    echo ""
    read -p "Remove configuration directory ~/.clio? This includes downloaded modules. (y/N): " -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$CLIO_DIR"
        echo "✅ Removed $CLIO_DIR"
    else
        echo "⏭️  Kept $CLIO_DIR (modules preserved)"
    fi
fi

# Ask about database
if [ -f "$HOME/.clio.db" ]; then
    echo ""
    read -p "Remove database ~/.clio.db? This includes command history and cached data. (y/N): " -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
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
