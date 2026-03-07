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

# Install the module runner script (works around Android/Termux syscall restrictions)
echo "Installing clio-run-module helper script..."
cat > "$INSTALL_DIR/clio-run-module" <<'EOMODRUNNER'
#!/bin/bash
# clio-run-module - Execute Clio YAML modules without triggering Android seccomp syscall blocks
# Pure bash implementation - no Python required

set -euo pipefail

MODULE_ID="${1:-}"
FLOW_NAME="${2:-setup}"
DB_PATH="${HOME}/.clio/modules.db"

if [ -z "$MODULE_ID" ]; then
    echo "Usage: clio-run-module <module_id> [flow_name]"
    echo "Example: clio-run-module termux_setup setup"
    exit 1
fi

if [ ! -f "$DB_PATH" ]; then
    echo "❌ Module database not found at: $DB_PATH"
    echo "Run 'clio' and type 'sync' to download modules first."
    exit 1
fi

# Check for sqlite3
if ! command -v sqlite3 >/dev/null 2>&1; then
    echo "❌ sqlite3 is required but not installed."
    echo "On Termux: pkg install sqlite"
    exit 1
fi

# Load pre-processed bash script from database
echo "📦 Loading module: $MODULE_ID"
BASH_SCRIPT=$(sqlite3 "$DB_PATH" "SELECT bash_script FROM modules WHERE module_id = '$MODULE_ID';" 2>/dev/null)

if [ -z "$BASH_SCRIPT" ]; then
    echo "❌ Module '$MODULE_ID' not found in database or not preprocessed"
    echo "Run 'clio' and type 'sync' to download and process modules."
    exit 1
fi

# Write script to temp file
TEMP_SCRIPT=$(mktemp)
echo "$BASH_SCRIPT" > "$TEMP_SCRIPT"
trap "rm -f $TEMP_SCRIPT" EXIT

# Source the variables
source "$TEMP_SCRIPT"

# Display module info
echo "📋 $MODULE_NAME"
[ -n "$MODULE_DESC" ] && echo "   $MODULE_DESC"
[ -n "$ESTIMATED_TIME" ] && echo "   ⏱️  Estimated time: $ESTIMATED_TIME"
echo ""

# Execute steps
SECTION_INDEX=0
for ((i=0; i<STEP_COUNT; i++)); do
    type_var="STEP_${i}_TYPE"
    step_type="${!type_var}"
    
    case "$step_type" in
        message)
            content_var="STEP_${i}_CONTENT"
            echo -e "${!content_var}"
            ;;
        confirm)
            prompt_var="STEP_${i}_PROMPT"
            default_var="STEP_${i}_DEFAULT"
            on_no_var="STEP_${i}_ON_NO"
            
            prompt_text="${!prompt_var:-Continue?}"
            default_val="${!default_var:-yes}"
            on_no="${!on_no_var}"
            
            hint="[Y/n]"
            [ "$default_val" != "yes" ] && hint="[y/N]"
            
            read -p "$prompt_text $hint: " response
            response=$(echo "$response" | tr '[:upper:]' '[:lower:]')
            [ -z "$response" ] && response="$default_val"
            
            if [[ "$response" =~ ^n(o)?$ ]]; then
                if [ "$on_no" = "abort" ]; then
                    echo "❌ Aborted by user"
                    exit 0
                fi
            fi
            ;;
        command)
            desc_var="STEP_${i}_DESCRIPTION"
            cmd_var="STEP_${i}_COMMAND"
            show_var="STEP_${i}_SHOW_OUTPUT"
            interactive_var="STEP_${i}_INTERACTIVE"
            continue_var="STEP_${i}_CONTINUE_ON_ERROR"
            
            [ -n "${!desc_var}" ] && echo "${!desc_var}..."
            
            cmd="${!cmd_var}"
            if [ "${!interactive_var}" = "true" ]; then
                eval "$cmd" || {
                    if [ "${!continue_var}" != "true" ]; then
                        echo "❌ Command failed"
                        exit 1
                    fi
                    echo "⚠️  Warning: Command failed"
                }
            elif [ "${!show_var}" = "true" ]; then
                eval "$cmd" || {
                    if [ "${!continue_var}" != "true" ]; then
                        echo "❌ Command failed"
                        exit 1
                    fi
                    echo "⚠️  Warning: Command failed"
                }
            else
                eval "$cmd" >/dev/null 2>&1 || {
                    if [ "${!continue_var}" != "true" ]; then
                        echo "❌ Command failed"
                        exit 1
                    fi
                    echo "⚠️  Warning: Command failed"
                }
            fi
            ;;
        section)
            title_var="STEP_${i}_TITLE"
            sub_count_var="STEP_${i}_SUB_COUNT"
            
            ((SECTION_INDEX++))
            echo ""
            echo "[$SECTION_INDEX/$SECTION_COUNT] ${!title_var}"
            echo "────────────────────────────────────────────────────────────"
            
            # Execute substeps
            sub_count="${!sub_count_var:-0}"
            for ((j=0; j<sub_count; j++)); do
                sub_type_var="STEP_${i}_SUB_${j}_TYPE"
                sub_type="${!sub_type_var}"
                
                if [ "$sub_type" = "command" ]; then
                    sub_desc_var="STEP_${i}_SUB_${j}_DESCRIPTION"
                    sub_cmd_var="STEP_${i}_SUB_${j}_COMMAND"
                    sub_show_var="STEP_${i}_SUB_${j}_SHOW_OUTPUT"
                    sub_interactive_var="STEP_${i}_SUB_${j}_INTERACTIVE"
                    sub_continue_var="STEP_${i}_SUB_${j}_CONTINUE_ON_ERROR"
                    
                    [ -n "${!sub_desc_var}" ] && echo "${!sub_desc_var}..."
                    
                    sub_cmd="${!sub_cmd_var}"
                    if [ "${!sub_interactive_var}" = "true" ]; then
                        eval "$sub_cmd" || {
                            if [ "${!sub_continue_var}" != "true" ]; then
                                echo "❌ Command failed"
                                exit 1
                            fi
                            echo "⚠️  Warning: Command failed"
                        }
                    elif [ "${!sub_show_var}" = "true" ]; then
                        eval "$sub_cmd" || {
                            if [ "${!sub_continue_var}" != "true" ]; then
                                echo "❌ Command failed"
                                exit 1
                            fi
                            echo "⚠️  Warning: Command failed"
                        }
                    else
                        eval "$sub_cmd" >/dev/null 2>&1 || {
                            if [ "${!sub_continue_var}" != "true" ]; then
                                echo "❌ Command failed"
                                exit 1
                            fi
                            echo "⚠️  Warning: Command failed"
                        }
                    fi
                fi
            done
            
            echo "✅ ${!title_var} complete"
            ;;
        check_command)
            cmd_var="STEP_${i}_COMMAND"
            command -v "${!cmd_var}" >/dev/null 2>&1 || {
                echo "⚠️  Command '${!cmd_var}' not found, skipping..."
            }
            ;;
    esac
done

echo ""
echo "✅ Module execution completed!"
EOMODRUNNER

chmod +x "$INSTALL_DIR/clio-run-module"

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
