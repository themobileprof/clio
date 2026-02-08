# Clio - Offline-First Smart CLI Assistant

Clio is a lightweight, offline-first command-line assistant designed for Termux on Android (and other Linux systems). It acts as a natural language interface for your shell, helping you find the right command without needing to search online.

## Features

-   **Offline-First**: Uses a compiled, static catalog of ~100 common operations for instant results.
-   **Smart Detection**:
    -   **Verb-Noun Parsing**: Understands "create directory", "list files", "check ip".
    -   **Stemming**: Handles variations like "copying", "copied", "files".
    -   **Fuzzy Matching**: Forgives typos.
-   **System Integration**: Searches standard `man` pages if the static catalog fails.
-   **Safe Execution**: Optimized for Android (Termux) to avoid syscall crashes (`safeexec`).
-   **Interactive REPL**: A simple menu-driven interface to view useage, run commands, or search again.

## Installation

### Automatic Install (Recommended)
You can install the latest binary automatically using curl. This script detects your OS/Arch (including Termux) and downloads the correct binary.

```bash
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh | bash
```

### Manual Download
Download the binary for your platform from the [Releases Page](https://github.com/themobileprof/clio/releases/latest) and place it in your path.

### Build from Source (Optional)
If you prefer to build it yourself:

```bash
go build -ldflags="-s -w" -o clio ./cmd/clio
```

### Uninstall
To remove Clio from your system:

```bash
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/uninstall.sh | bash
```

Or if you have the repository:

```bash
./uninstall.sh
```

The uninstall script will:
- Remove the Clio binary
- Optionally remove configuration directory (`~/.clio`) and modules
- Optionally remove database (`~/.clio.db`) with cached data

## Usage

Run the binary directly:

```bash
./clio
```

**Special Commands:**
- `setup` - Run Termux setup wizard (Termux only, first-time setup)
- `sync` - Download latest automation modules from GitHub
- `clear` - Clear the screen
- `exit` or `quit` - Exit Clio

### Interactive Mode
Type your query at the prompt:

```text
>> how do I extract a tar file?
✓ Use: tar -xzvf
────────────────────────
Purpose : Extract tar.gz archive

What would you like to do?
  1) Show examples and usage
  2) Run the command
  3) Show command only
  ...
```

### Termux Setup (First-Time Users)
If you're on Termux and haven't set up your development environment yet:

```text
>> setup
```

This interactive wizard will configure:
- **System Updates**: Package updates and mirror optimization
- **Storage Access**: Android storage integration
- **Zsh Shell**: Oh-My-Zsh with Powerlevel10k theme
- **Dev Tools**: Git, GitHub CLI, Vim (configured as lightweight IDE)
- **AI Assistant**: LLM tool with DeepSeek integration
- **Languages**: Optional installation of PHP, Node.js, Golang, Python

The setup takes 10-20 minutes and only needs to be run once.

### Module Sync
To fetch the latest automation modules from the central repository:

```text
>> sync
🔄 Syncing modules from remote...
  Processing archive_directory.yaml...
✅ Sync complete. Updated 66 modules.
```

### Pipe Mode
You can also pipe queries directly:

```bash
echo "check disk space" | ./clio
```

## Architecture

1.  **Layer 1 (Static)**: Instant lookup for common patterns using Verb-Noun mapping.
2.  **Layer 2 (Man Pages)**: Searches system manual pages.
3.  **Layer 3 (Modules)**: Executes sophisticated automation flows (YAML) synced from [GitHub](https://github.com/themobileprof/clipilot/tree/main/modules).
4.  **Layer 4 (Remote)**: Fallback to remote API for complex queries.
