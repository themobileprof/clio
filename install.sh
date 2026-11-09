#!/bin/bash
set -e

VERSION=0.0.29

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

# phone-loader:start
# Android 10 execs the ordinary linux binary. Android 16's linker rejects
# that file ("has unexpected e_type: 2") and needs the position-independent one.
# Both are release downloads. The phone does not compile anything.
loader_rejected() {
    local bin="$1"
    local out
    if command -v timeout >/dev/null 2>&1; then
        out=$(timeout 20 "$bin" --print-module __clio_install_probe__ </dev/null 2>&1 || true)
    else
        out=$("$bin" --print-module __clio_install_probe__ </dev/null 2>&1 || true)
    fi
    printf '%s\n' "$out" | grep -q -e 'unexpected e_type' -e 'CANNOT LINK EXECUTABLE' -e 'Exec format error'
}

download_asset() {
    local asset="$1"
    local dest="$2"
    local url="https://github.com/$OWNER/$REPO/releases/latest/download/$asset"
    echo "Downloading $url..."
    if command -v curl >/dev/null 2>&1; then
        curl -fsL -o "$dest" "$url" || { rm -f "$dest"; return 1; }
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$dest" "$url" || { rm -f "$dest"; return 1; }
    else
        echo "Error: Neither curl nor wget found."
        exit 1
    fi
    if [ ! -s "$dest" ]; then
        echo "Error: Failed to download $asset."
        rm -f "$dest"
        return 1
    fi
    # Go binaries are usually > 1MB. "Not Found" is a few bytes.
    local file_size
    file_size=$(wc -c < "$dest" | tr -d '[:space:]')
    if [ "$file_size" -lt 102400 ]; then
        echo "Error: $asset is too small ($file_size bytes). The release may not include it yet."
        cat "$dest"
        rm -f "$dest"
        return 1
    fi
    chmod +x "$dest"
}

try_asset() {
    local asset="$1"
    local dest="$2"
    local check_loader="$3"
    rm -f "$dest"
    if ! download_asset "$asset" "$dest"; then
        return 1
    fi
    if [ "$check_loader" = "yes" ] && loader_rejected "$dest"; then
        echo "This phone refused to start $asset."
        rm -f "$dest"
        return 1
    fi
    echo "Using $asset"
    return 0
}

install_downloaded_binary() {
    local dest="$1"
    # Piped to sh on purpose (curl ... | sh). Do not use bash arrays here.
    if [ "$ARCH" = "arm64" ] && [ "$(uname -o 2>/dev/null || true)" = "Android" ]; then
        if try_asset clio-linux-arm64 "$dest" yes; then
            return 0
        fi
        if try_asset clio-android-arm64 "$dest" yes; then
            return 0
        fi
        echo "Error: No Clio download would start on this phone."
        exit 1
    fi
    if try_asset "${BIN_NAME}-${OS}-${ARCH}" "$dest" no; then
        return 0
    fi
    echo "Error: Failed to download binary. Check if the release/asset exists."
    exit 1
}
# phone-loader:end

install_downloaded_binary "$BIN_NAME"
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
# clio                         opens the >> prompt
# clio termux_setup            runs that wizard in this shell
# clio ask 'question'          DeepSeek or Gemini when online, otherwise the offline list
# clio key deepseek            saves the DeepSeek key
# clio key gemini              saves the Gemini key
# clio model deepseek|gemini   chooses which one clio ask uses
set -euo pipefail
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
BIN="$SCRIPT_DIR/clio.bin"
MODEL_FILE="$HOME/.config/clio/model"
if [ ! -x "$BIN" ]; then
    echo "Clio program is missing at $BIN. Reinstall Clio."
    exit 1
fi

usage() {
    echo "Usage:"
    echo "  clio                      open the >> prompt"
    echo "  clio termux_setup         run that wizard"
    echo "  clio ask 'your question'  DeepSeek or Gemini, or the offline list"
    echo "  clio key deepseek         save the DeepSeek key"
    echo "  clio key gemini           save the Gemini key"
    echo "  clio model deepseek       ask DeepSeek from now on"
    echo "  clio model gemini         ask Gemini from now on"
    exit 1
}

need_llm() {
    if ! command -v llm >/dev/null 2>&1; then
        echo "The model program is missing. Run: clio termux_setup"
        exit 1
    fi
}

key_file() {
    printf '%s\n' "${LLM_USER_PATH:-$HOME/.config/io.datasette.llm}/keys.json"
}

has_key() {
    local file names
    file=$(key_file)
    # The key file is enough. Starting llm just to list names costs a
    # Python startup on the phone before the question is even sent.
    if [ -f "$file" ] && grep -q "\"$1\"" "$file"; then
        return 0
    fi
    names=$(llm keys 2>/dev/null || true)
    case "
$names
" in
        *"
$1
"*) return 0 ;;
    esac
    return 1
}

pick_provider() {
    local choice=""
    PROVIDER=
    if [ -f "$MODEL_FILE" ]; then
        choice=$(tr -d '[:space:]' < "$MODEL_FILE")
    fi
    case "$choice" in
        deepseek|gemini)
            if has_key "$choice"; then
                PROVIDER=$choice
                return 0
            fi
            ;;
    esac
    if has_key deepseek; then
        PROVIDER=deepseek
        return 0
    fi
    if has_key gemini; then
        PROVIDER=gemini
        return 0
    fi
    return 1
}

validate_key() {
    local name="$1" url header py
    case "$name" in
        deepseek)
            url=https://api.deepseek.com/models
            header=Authorization
            ;;
        gemini)
            url=https://generativelanguage.googleapis.com/v1beta/models
            header=x-goog-api-key
            ;;
        *)
            return 2
            ;;
    esac
    py=$(command -v python || command -v python3 || true)
    if [ -z "$py" ]; then
        return 3
    fi
    printf '%s' "$KEY" | "$py" -c '
import sys
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError
url, header = sys.argv[1], sys.argv[2]
key = sys.stdin.read().strip()
if not key:
    sys.exit(4)
value = ("Bearer " + key) if header == "Authorization" else key
req = Request(url, headers={header: value})
try:
    with urlopen(req, timeout=10) as resp:
        sys.exit(0 if resp.status == 200 else 2)
except HTTPError:
    sys.exit(2)
except URLError:
    sys.exit(3)
' "$url" "$header"
}

offline_answer() {
    local why="$1"
    shift
    local answer
    if answer=$(printf '%s\n' "$*" | "$BIN" 2>/dev/null) && [ -n "$answer" ]; then
        if [ -n "$why" ]; then
            printf '%s\n' "$why"
        fi
        printf 'Use: %s\n' "$answer"
        return 0
    fi
    if [ -n "$why" ]; then
        printf '%s\n' "$why"
    fi
    return 1
}

ensure_deepseek_model() {
    local path dir file
    path=$(llm keys path 2>/dev/null || true)
    if [ -z "$path" ]; then
        return 0
    fi
    dir=$(dirname "$path")
    file="$dir/extra-openai-models.yaml"
    if [ -f "$file" ] && grep -q 'model_id: deepseek-flash' "$file"; then
        return 0
    fi
    mkdir -p "$dir"
    printf '\n- model_id: deepseek-flash\n  model_name: deepseek-flash\n  api_base: https://api.deepseek.com\n  api_key_name: deepseek\n' >> "$file"
}

# One short Python process, streaming. DeepSeek's thinking mode is off,
# so the first word is not held back. llm is only the fallback.
ask_fast() {
    local py system
    system="$1"
    shift
    py=$(command -v python || command -v python3 || true)
    [ -n "$py" ] || return 1
    printf '%s' "$*" | "$py" -c '
import json, os, sys
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError
provider, system = sys.argv[1], sys.argv[2]
question = sys.stdin.read()
base = os.environ.get("LLM_USER_PATH") or os.path.expanduser("~/.config/io.datasette.llm")
try:
    keys = json.load(open(os.path.join(base, "keys.json")))
except Exception:
    sys.exit(1)
key = str(keys.get(provider) or "").strip()
if not key:
    sys.exit(1)
if provider == "deepseek":
    url = "https://api.deepseek.com/chat/completions"
    payload = {
        "model": "deepseek-flash",
        "messages": [
            {"role": "system", "content": system},
            {"role": "user", "content": question},
        ],
        "max_tokens": 220,
        "stream": True,
        "thinking": {"type": "disabled"},
    }
    headers = {"Authorization": "Bearer " + key, "Content-Type": "application/json"}
else:
    url = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.8-flash:streamGenerateContent?alt=sse"
    payload = {
        "systemInstruction": {"parts": [{"text": system}]},
        "contents": [{"role": "user", "parts": [{"text": question}]}],
        "generationConfig": {
            "maxOutputTokens": 220,
            "thinkingConfig": {"thinkingLevel": "low", "includeThoughts": False},
        },
    }
    headers = {"x-goog-api-key": key, "Content-Type": "application/json"}
req = Request(url, data=json.dumps(payload).encode(), headers=headers)
try:
    resp = urlopen(req, timeout=45)
except (HTTPError, URLError):
    sys.exit(1)
wrote = False
for raw in resp:
    line = raw.decode("utf-8", "replace").strip()
    if not line.startswith("data:"):
        continue
    data = line[5:].strip()
    if not data or data == "[DONE]":
        continue
    try:
        obj = json.loads(data)
    except Exception:
        continue
    text = ""
    choices = obj.get("choices") or []
    if choices:
        delta = (choices[0] or {}).get("delta") or {}
        text = delta.get("content") or ""
    if not text:
        cands = obj.get("candidates") or []
        if cands:
            parts = ((cands[0] or {}).get("content") or {}).get("parts") or []
            for part in parts:
                if part.get("thought"):
                    continue
                text += part.get("text") or ""
    if text:
        sys.stdout.write(text)
        sys.stdout.flush()
        wrote = True
sys.exit(0 if wrote else 1)
' "$PROVIDER" "$system"
}

ask_model() {
    local model why="" short
    if [ "$#" -eq 0 ]; then
        echo "Usage: clio ask 'your question'"
        exit 1
    fi
    # A phone screen. Neovim's ,ga and ,gr are separate and stay long.
    short="Reply briefly for a phone. At most four short sentences, or one command and one line of explanation."
    if pick_provider && ask_fast "$short" "$@"; then
        exit 0
    fi
    if [ -n "${PROVIDER:-}" ] && command -v llm >/dev/null 2>&1; then
        case "$PROVIDER" in
            deepseek)
                ensure_deepseek_model
                model=deepseek-flash
                ;;
            gemini) model=gemini-3.8-flash ;;
        esac
        case "$PROVIDER" in
            deepseek)
                llm -m "$model" -s "$short" -o max_tokens 220 -R -- "$@"
                ;;
            gemini)
                llm -m "$model" -s "$short" -o max_output_tokens 220 -o thinking_level low -R -- "$@"
                ;;
        esac && exit 0
        why="The model did not answer. Offline answer:"
    fi
    if offline_answer "$why" "$@"; then
        exit 0
    fi
    if ! command -v llm >/dev/null 2>&1; then
        echo "The model program is missing. Run: clio termux_setup"
    else
        echo "No match in the offline list."
        echo "  clio key deepseek"
        echo "  clio key gemini"
    fi
    exit 1
}

on_termux() {
    [ -n "${TERMUX_VERSION:-}" ] || [ -d /data/data/com.termux ]
}

# Termux cannot compile jiter or pydantic-core. Use Android wheels and an
# llm release that accepts those wheels. A newer llm asks pip to build them.
termux_pip() {
    python -m pip install --upgrade-strategy only-if-needed \
        --only-binary=jiter,pydantic-core \
        --extra-index-url https://pypi.flet.dev/ \
        --extra-index-url https://eutalix.github.io/android-pydantic-core/ \
        'jiter==0.15.0' 'pydantic-core==2.50.0' 'pydantic==2.14.0' \
        'llm==0.32.1' "$@"
}

install_plugin() {
    local package="$1" needle="$2"
    if llm models 2>/dev/null | grep -q "$needle"; then
        return 0
    fi
    if on_termux; then
        termux_pip "$package"
    else
        llm install "$package"
    fi
}

save_key() {
    local name="$1" package="$2" needle="$3" KEY status=0
    need_llm
    # Install before the prompt. Pip output used to appear after the key,
    # and llm then asked for the key a second time.
    if ! install_plugin "$package" "$needle"; then
        echo "The $name model program did not install. Nothing was saved."
        exit 1
    fi
    if [ -t 0 ]; then
        printf 'Enter your %s API key: ' "$name"
        IFS= read -rs KEY || true
        printf '\n'
    else
        IFS= read -r KEY || true
    fi
    if [ -z "${KEY//[[:space:]]/}" ]; then
        echo "No key entered. Nothing was saved."
        exit 1
    fi
    validate_key "$name" || status=$?
    if [ "$status" -ne 0 ]; then
        if [ "$status" -eq 3 ]; then
            echo "Could not reach $name. The key was not saved."
        else
            echo "That $name key was rejected. It was not saved."
        fi
        unset KEY
        exit 1
    fi
    # --value skips llm's own "Enter key:" prompt. A pipe does not, because
    # that prompt reads the terminal, not the pipe.
    llm keys set "$name" --value "$KEY"
    # Neovim reads this file. It is a different store from the one clio ask uses.
    if [ "$name" = "deepseek" ]; then
        mkdir -p "$HOME/.config/gp"
        umask 077
        printf '%s' "$KEY" > "$HOME/.config/gp/deepseek.key"
        chmod 600 "$HOME/.config/gp/deepseek.key"
    fi
    unset KEY
    echo "Saved the $name key."
    exit 0
}

if [ $# -eq 0 ]; then
    exec "$BIN"
fi
case "$1" in
    --*) exec "$BIN" "$@" ;;
    key|keys)
        if [ $# -ne 2 ]; then
            usage
        fi
        case "$2" in
            deepseek) save_key deepseek llm-deepseek deepseek-chat ;;
            gemini) save_key gemini llm-gemini gemini-3.8-flash ;;
            *) usage ;;
        esac
        ;;
    model)
        if [ $# -eq 1 ]; then
            if [ -f "$MODEL_FILE" ]; then
                echo "Saved model: $(tr -d '[:space:]' < "$MODEL_FILE")"
            else
                echo "No saved model. Clio asks DeepSeek when that key exists, otherwise Gemini."
            fi
            exit 0
        fi
        if [ $# -ne 2 ]; then
            usage
        fi
        case "$2" in
            deepseek|gemini)
                mkdir -p "$HOME/.config/clio"
                printf '%s\n' "$2" > "$MODEL_FILE"
                echo "Clio will ask $2."
                exit 0
                ;;
            *) usage ;;
        esac
        ;;
    ask)
        shift
        ask_model "$@"
        ;;
esac
# One word is a wizard name. A question has to use: clio ask
if [ $# -eq 1 ]; then
    exec "$SCRIPT_DIR/clio-run-module" "$@"
fi
usage
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
