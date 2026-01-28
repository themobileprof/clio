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

### Prerequisites
-   Go 1.22+
-   `man` (optional, for Layer 2 search)

### Build
To build a small, static binary optimized for size:

```bash
go build -ldflags="-s -w" -o clio ./cmd/clio
```

## Usage

Run the binary directly:

```bash
./clio
```

### Interactive Mode
Type your query at the prompt:

```text
>> how do I extract a tar file?
✓ Command found: tar -xzvf
────────────────────────
Purpose : Extract tar.gz archive

What would you like to do?
  1) Show examples and usage
  2) Run the command
  3) Show command only
  ...
```

### Pipe Mode
You can also pipe queries directly:

```bash
echo "check disk space" | ./clio
```

## Architecture

1.  **Layer 1 (Static)**: Instant lookup for common patterns using Verb-Noun mapping.
2.  **Layer 2 (Man Pages)**: Searches system manual pages.
3.  **Layer 3 (Modules)**: (Coming Soon) Custom SQLite-based workflows.
4.  **Layer 4 (Remote)**: Fallback to remote API for complex queries.
