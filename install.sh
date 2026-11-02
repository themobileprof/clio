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
    x86_64)          ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    armv7l|armv8l|arm) ARCH="arm" ;;   # 32-bit ARM Termux (older phones)
    i686|i386)       ARCH="386" ;;      # 32-bit x86 Termux (rare)
    *)
        echo "Unsupported Arch: $ARCH"
        echo "Build from source: ./build-termux.sh"
        exit 1
        ;;
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
mv "$BIN_NAME" "$INSTALL_DIR/clio.bin"

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

if [ -z "$MODULE_ID" ]; then
    echo "Usage: clio <module_id>"
    echo "Example: clio termux_setup"
    exit 1
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BIN="$SCRIPT_DIR/clio.bin"
if [ ! -x "$BIN" ]; then
    echo "❌ Clio program is missing at $BIN"
    echo "Reinstall Clio, then run 'clio' and type 'sync'."
    exit 1
fi

# The Go program only reads the saved script. It does not start the wizard,
# because Termux blocks programs that Clio itself launches.
echo "📦 Loading module: $MODULE_ID"
if ! BASH_SCRIPT=$("$BIN" --print-module "$MODULE_ID"); then
    exit 1
fi

if [ -z "$BASH_SCRIPT" ]; then
    echo "❌ Module '$MODULE_ID' not found in database or not preprocessed"
    echo "Run 'clio' and type 'sync' to download and process modules."
    exit 1
fi

# Write script to temp file. Older syncs stored ---FLOW_END--- as a real
# line. The shell would try to run it.
TEMP_SCRIPT=$(mktemp)
CLEAN_SCRIPT=$(mktemp)
printf '%s\n' "$BASH_SCRIPT" > "$TEMP_SCRIPT"
grep -v '^---FLOW_END---$' "$TEMP_SCRIPT" > "$CLEAN_SCRIPT" || true
trap "rm -f $TEMP_SCRIPT $CLEAN_SCRIPT" EXIT

# Source the variables
source "$CLEAN_SCRIPT"

# Display module info
echo "📋 $MODULE_NAME"
[ -n "$MODULE_DESC" ] && echo "   $MODULE_DESC"
[ -n "$ESTIMATED_TIME" ] && echo "   ⏱️  Estimated time: $ESTIMATED_TIME"
echo ""

# val NAME prints the variable or nothing. Missing flags are not errors.
val() {
    local name=$1
    if [ "${!name+x}" = "x" ]; then
        printf '%s' "${!name}"
    fi
}

# Replace {{ .NAME }} with the shell variable NAME.
expand_vars() {
    local text=$1 name token
    while [[ $text =~ \{\{[[:space:]]*\.?[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*\}\} ]]; do
        token="${BASH_REMATCH[0]}"
        name="${BASH_REMATCH[1]}"
        text="${text//"$token"/$(val "$name")}"
    done
    printf '%s' "$text"
}

configure_zshrc() {
    local f="$HOME/.zshrc"
    if [ ! -f "$f" ]; then
        cat > "$f" << 'EOF'
# Zsh configuration
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="powerlevel10k/powerlevel10k"
plugins=(git z sudo colored-man-pages)
source $ZSH/oh-my-zsh.sh
EOF
    fi
    if ! grep -q powerlevel10k "$f" 2>/dev/null; then
        sed -i 's/ZSH_THEME="robbyrussell"/ZSH_THEME="powerlevel10k\/powerlevel10k"/' "$f" || true
    fi
    if ! grep -q 'alias ll=' "$f" 2>/dev/null; then
        cat >> "$f" << 'EOF'

# Useful aliases
alias ll="ls -lah"
alias la="ls -A"
alias l="ls -CF"
alias ..="cd .."
alias ...="cd ../.."
EOF
    fi
}

mark_setup_complete() {
    mkdir -p "$HOME/.clio"
    printf '1\n' > "$HOME/.clio/termux_setup_complete"
}

run_command() {
    local id=$1
    local desc cmd show interactive cont cond
    desc=$(val "${id}_DESCRIPTION")
    cmd=$(val "${id}_COMMAND")
    show=$(val "${id}_SHOW_OUTPUT")
    interactive=$(val "${id}_INTERACTIVE")
    cont=$(val "${id}_CONTINUE_ON_ERROR")
    cond=$(val "${id}_CONDITION")
    if [ -n "$cond" ]; then
        cond=$(expand_vars "$cond")
        if [ -z "$cond" ] || [ "$cond" = "false" ]; then
            return 0
        fi
    fi
    [ -n "$desc" ] && echo "${desc}..."
    if [ -z "$cmd" ]; then
        echo "⚠️  Skipping an empty command"
        return 0
    fi
    cmd=$(expand_vars "$cmd")
    # Wizard commands were written for a normal shell. nounset turns a missing
    # optional flag into "unbound variable", and pipefail turns "cmd | head"
    # into a failure when the writer gets SIGPIPE. Keep errexit so a real
    # failing command still stops the step.
    local status=0
    set +u
    set +o pipefail
    if [ "$interactive" = "true" ] || [ "$show" = "true" ]; then
        eval "$cmd" || status=$?
    else
        eval "$cmd" >/dev/null 2>&1 || status=$?
    fi
    set -u
    set -o pipefail
    if [ "$status" -ne 0 ]; then
        if [ "$cont" != "true" ]; then
            echo "❌ Command failed"
            return 1
        fi
        echo "⚠️  Warning: Command failed"
    fi
}

# run_one returns 10 when the caller should jump to GOTO_LABEL.
run_one() {
    local prefix=$1 i=$2
    local id="${prefix}${i}"
    local step_type
    step_type=$(val "${id}_TYPE")
    case "$step_type" in
        message)
            printf '%b\n' "$(val "${id}_CONTENT")"
            ;;
        confirm)
            local prompt default on_no on_yes hint response
            prompt=$(val "${id}_PROMPT")
            default=$(val "${id}_DEFAULT")
            on_no=$(val "${id}_ON_NO")
            on_yes=$(val "${id}_ON_YES")
            [ -z "$prompt" ] && prompt="Continue?"
            [ -z "$default" ] && default="yes"
            hint="[Y/n]"
            [ "$default" != "yes" ] && [ "$default" != "y" ] && hint="[y/N]"
            if ! read -r -p "$prompt $hint: " response; then
                echo "❌ No answer, stopping"
                exit 0
            fi
            response=$(printf '%s' "$response" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')
            [ -z "$response" ] && response="$default"
            if [[ "$response" =~ ^y(es)?$ ]]; then
                if [ -n "$on_yes" ]; then
                    GOTO_LABEL=$on_yes
                    return 10
                fi
            else
                if [ "$on_no" = "abort" ]; then
                    echo "❌ Aborted by user"
                    exit 0
                fi
                if [ -n "$on_no" ]; then
                    GOTO_LABEL=$on_no
                    return 10
                fi
            fi
            ;;
        input)
            local prompt var required value
            prompt=$(val "${id}_PROMPT")
            var=$(val "${id}_VARIABLE")
            required=$(val "${id}_REQUIRED")
            while true; do
                if ! read -r -p "$prompt: " value; then
                    echo "❌ No answer, stopping"
                    exit 0
                fi
                value="${value#"${value%%[![:space:]]*}"}"
                value="${value%"${value##*[![:space:]]}"}"
                if [ -n "$value" ] || [ "$required" != "true" ]; then
                    break
                fi
                echo "This field is required."
            done
            if [ -n "$var" ]; then
                printf -v "$var" '%s' "$value"
            fi
            ;;
        command)
            run_command "$id" || return $?
            ;;
        section)
            local title count
            SECTION_INDEX=$((SECTION_INDEX + 1))
            title=$(val "${id}_TITLE")
            count=$(val "${id}_SUB_COUNT")
            [ -z "$count" ] && count=0
            echo ""
            echo "[$SECTION_INDEX/$(val SECTION_COUNT)] $title"
            echo "────────────────────────────────────────────────────────────"
            run_list "${id}_SUB_" "$count" || return $?
            echo "✅ $title complete"
            ;;
        check_command)
            local cmd on_missing on_exists
            cmd=$(val "${id}_COMMAND")
            on_missing=$(val "${id}_ON_MISSING")
            on_exists=$(val "${id}_ON_EXISTS")
            if command -v "$cmd" >/dev/null 2>&1; then
                if [ -n "$on_exists" ] && [ "$on_exists" != "skip" ]; then
                    GOTO_LABEL=$on_exists
                    return 10
                fi
            else
                if [ -n "$on_missing" ] && [ "$on_missing" != "skip" ]; then
                    GOTO_LABEL=$on_missing
                    return 10
                fi
            fi
            ;;
        check_path)
            local path on_missing on_exists
            path=$(val "${id}_PATH")
            path="${path/#\~/$HOME}"
            on_missing=$(val "${id}_ON_MISSING")
            on_exists=$(val "${id}_ON_EXISTS")
            if [ -e "$path" ]; then
                if [ -n "$on_exists" ] && [ "$on_exists" != "skip" ]; then
                    GOTO_LABEL=$on_exists
                    return 10
                fi
            else
                if [ -n "$on_missing" ] && [ "$on_missing" != "skip" ]; then
                    GOTO_LABEL=$on_missing
                    return 10
                fi
            fi
            ;;
        label)
            ;;
        goto)
            GOTO_LABEL=$(val "${id}_LABEL")
            return 10
            ;;
        file_operation)
            local op desc
            op=$(val "${id}_OPERATION")
            desc=$(val "${id}_DESCRIPTION")
            [ -n "$desc" ] && echo "${desc}..."
            case "$op" in
                configure_zshrc) configure_zshrc ;;
                mark_complete) mark_setup_complete ;;
                *) echo "⚠️  Skipping unknown file operation: $op" ;;
            esac
            ;;
        "")
            ;;
        *)
            echo "⚠️  Skipping unknown step: $step_type"
            ;;
    esac
    return 0
}

# run_list walks steps. A return of 10 jumps to GOTO_LABEL in this list.
run_list() {
    local prefix=$1 count=$2
    local -A labels=()
    local i=0 step_type name status
    while [ "$i" -lt "$count" ]; do
        step_type=$(val "${prefix}${i}_TYPE")
        if [ "$step_type" = "label" ]; then
            name=$(val "${prefix}${i}_NAME")
            [ -n "$name" ] && labels[$name]=$i
        fi
        i=$((i + 1))
    done
    i=0
    while [ "$i" -lt "$count" ]; do
        status=0
        run_one "$prefix" "$i" || status=$?
        if [ "$status" -eq 10 ]; then
            name=$GOTO_LABEL
            if [ -z "${labels[$name]+x}" ]; then
                echo "❌ Unknown step: $name"
                return 1
            fi
            i=${labels[$name]}
            continue
        fi
        if [ "$status" -ne 0 ]; then
            return "$status"
        fi
        i=$((i + 1))
    done
}

SECTION_INDEX=0
GOTO_LABEL=
run_list "STEP_" "$STEP_COUNT"

echo ""
echo "✅ Module execution completed!"
EOMODRUNNER

chmod +x "$INSTALL_DIR/clio-run-module"

cat > "$INSTALL_DIR/clio" <<'EOWRAPPER'
#!/bin/bash
# clio with no arguments opens the prompt.
# clio termux_setup runs that wizard in this shell, not inside >>.
set -euo pipefail
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BIN="$SCRIPT_DIR/clio.bin"
if [ ! -x "$BIN" ]; then
    echo "Clio program is missing at $BIN. Reinstall Clio."
    exit 1
fi
if [ $# -eq 0 ]; then
    exec "$BIN"
fi
case "$1" in
    --*) exec "$BIN" "$@" ;;
esac
exec "$SCRIPT_DIR/clio-run-module" "$@"
EOWRAPPER
chmod +x "$INSTALL_DIR/clio"

echo "✅ Successfully installed clio to $INSTALL_DIR"

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
