# Clio - Offline-First Smart CLI Assistant

Clio is a lightweight, offline-first command-line assistant that helps you find and run shell commands using natural language. Perfect for Termux on Android, but works on any Linux system or macOS.

**Stop Googling. Start asking.**

```bash
>> extract tar file
✓ Use: tar -xzvf

>> find large files
✓ Use: find . -type f -size +100M

>> check memory usage
✓ Use: free -h
```

## Features

-   **🚀 Offline-First**: Works without internet. Uses a compiled catalog of ~100 common operations for instant results.
-   **🧠 Natural Language**: Type queries like "copy file", "list processes", "check disk space" - no syntax required.
-   **✨ Smart Detection**:
    -   **Verb-Noun Parsing**: Understands "create directory", "list files", "check ip"
    -   **Stemming**: Handles variations like "copying", "copied", "files"
    -   **Fuzzy Matching**: Forgives typos ("chek disk space" still works)
-   **🔒 Safe by Default**: Shows commands before running. You remain in control.
-   **📚 Man Page Integration**: Searches system manuals when static catalog doesn't match.
-   **🤖 Automation Modules**: YAML-based workflows for complex tasks (setup wizards, backups, deployments).
-   **📱 Termux Optimized**: Special handling for Android syscall restrictions - no SIGSYS crashes.
-   **⚡ Fast & Lightweight**: Single ~18MB binary. No dependencies. Instant startup.

## Installation

### Automatic Install (Recommended)
You can install the latest binary automatically using curl. This script detects your OS/Arch (including Termux) and downloads the correct binary.

**Primary method (recommended):**
```bash
curl -fsSL https://clipilot.themobileprof.com/clio | sh
```

**Fallback method (if registry is unavailable):**
```bash
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh | bash
```

### Manual Download
Download the binary for your platform from the [Releases Page](https://github.com/themobileprof/clio/releases/latest) and place it in your path.

### Build from Source

**Standard build:**
```bash
git clone https://github.com/themobileprof/clio.git
cd clio
go build -ldflags="-s -w" -o clio ./cmd/clio
```

**For Termux (on device):**
```bash
./build-termux.sh
cp clio $PREFIX/bin/
```

**Cross-compile for Termux (from Linux/Mac):**
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o clio-arm64 ./cmd/clio
# Transfer to device via adb or other method
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

### Starting Clio

Launch the interactive assistant:

```bash
clio
```

Or if in the source directory:

```bash
./clio
```

### Basic Examples

**File Operations:**
```
>> list files
✓ Use: ls -la

>> find pdf files
✓ Use: find . -name "*.pdf"

>> copy file
✓ Use: cp
[You can then edit to: cp source.txt dest.txt]
```

**System Information:**
```
>> check disk space
✓ Use: df -h

>> memory usage
✓ Use: free -h

>> show processes
✓ Use: ps aux
```

**Archives & Compression:**
```
>> extract tar file
✓ Use: tar -xzvf

>> unzip file
✓ Use: unzip

>> create tar archive
✓ Use: tar -czvf
```

### Special Commands

- **`setup`** - Show instructions for running module workflows (displays `clio-run-module` command)
- **`sync`** - Download latest automation modules from GitHub
- **`clear`** - Clear the screen
- **`exit`** or **`quit`** - Exit Clio

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

### Module Execution

Clio includes automation modules (YAML-based workflows) for complex tasks. To execute modules:

```bash
# In the Clio REPL, sync modules first
>> sync

# Exit and run modules with the external script
$ clio-run-module termux_setup setup
$ clio-run-module <module_id> <flow_name>
```

**Example - Termux Setup:**
If you're on Termux and want to configure your development environment:

```bash
$ clio-run-module termux_setup setup
```

**Why external script?** On Termux/Android, Go 1.24+'s subprocess handling triggers blocked syscalls (pidfd_open). Using a bash script avoids this. For simplicity and consistency, all platforms use the same execution method.

**The termux_setup module** includes:
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
🔄 Syncing modules from registry...
  Downloading org.themobileprof.archive_directory...
✅ Sync complete. Updated 66 modules.
```

Clio uses delta sync - only changed modules are downloaded, making subsequent syncs much faster. If the registry is unavailable, Clio automatically falls back to GitHub.

### Configuration
Clio can be configured via `~/.clio/config.yaml`:

```yaml
# Registry URL for module sync
registry_url: https://clipilot.themobileprof.com

# How long to cache module list (default: 24h)
cache_ttl: 24h

# Auto-sync interval (default: 168h / 7 days)
sync_interval: 168h
```

If the config file doesn't exist, Clio uses sensible defaults.

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

## Troubleshooting

### Termux: Module Execution

**Why use an external script?**

On Termux/Android, module execution (which spawns shell commands) must use the `clio-run-module` helper script instead of running directly in the Go binary.

**Technical Background:**

Go 1.24+ uses the `pidfd_open` syscall (syscall #434) for process management when spawning subprocesses. Android's seccomp filter blocks this syscall with `SIGSYS`, which would crash the Go binary if it tried to execute commands.

**The Solution:**

Clio separates concerns:
- ✅ **Query/search functionality** runs in the Go binary (no subprocess spawning)
- ✅ **Module execution** uses the `clio-run-module` bash script (reads pre-processed format from DB)

**Usage:**

```bash
# In the Clio REPL, sync modules first
>> sync

# Exit and run modules with the external script
$ clio-run-module termux_setup setup
$ clio-run-module <module_id> <flow_name>
```

The `clio-run-module` script is automatically installed alongside Clio and requires only `sqlite3` (pre-installed on Termux).

**What changed:**

Previously, userspace fixes (setting `PidFD = nil` in Go) were attempted but ineffective - Go's runtime still probes for `pidfd_open` during subprocess spawning. The only reliable solution is to avoid subprocess spawning in the Go binary entirely.

**Implementation:**

The `clio` binary handles all query/search functionality without issues. For module execution:
1. During `sync`, Go parses YAML and converts to bash-friendly key-value format
2. Both YAML and bash format are stored in the database
3. `clio-run-module` reads the pre-processed bash format (no YAML parsing in bash)
4. Bash script sources variables and executes workflow steps

**Benefits:**
- ✅ Simple: No fragile YAML parsing in bash
- ✅ Robust: Go's proper YAML parser handles everything during sync
- ✅ Universal: Same approach works on all platforms (Termux, Linux, macOS)

**Installation:**

The `install.sh` script automatically creates `clio-run-module` in the same directory as the `clio` binary on all platforms (e.g., `$PREFIX/bin/clio-run-module` on Termux, `/usr/local/bin/clio-run-module` on Linux/macOS).

## Development
- ✅ `faccessat2` - Custom `LookPath` avoids this syscall

**Why this works:**
Unlike runtime initialization issues, module execution happens in userspace where we control the `exec.Cmd` configuration. By setting `PidFD = nil`, we tell Go not to use pidfd functionality, avoiding the blocked syscall entirely.

**For detailed technical info**, see [docs/TERMUX_BUILD_GUIDE.md](docs/TERMUX_BUILD_GUIDE.md)


### Command Not Found

If `clio: command not found` after installation:

```bash
# Add to PATH (already done by installer, but verify)
export PATH="$HOME/.local/bin:$PATH"

# Or for Termux:
export PATH="$PREFIX/bin:$PATH"

# Make permanent (add to ~/.bashrc or ~/.zshrc)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Module Sync Fails

If `sync` command fails:
1. **Check internet connection**: `ping -c 1 8.8.8.8`
2. **Verify GitHub access**: `curl -I https://github.com`
3. **Check firewall/proxy**: Ensure outbound HTTPS is allowed
4. **Fallback behavior**: Clio automatically falls back to direct GitHub download if the registry is unavailable

### Permission Denied

If you get permission errors after copying the binary:
```bash
# Make it executable
chmod +x /path/to/clio

# For Termux:
chmod +x $PREFIX/bin/clio
```

### Database Errors

If you see SQLite errors:
```bash
# Reset the database
rm ~/.clio.db
clio  # Will recreate on next run
```

## Contributing

Contributions welcome! Areas of interest:
- **Layer 1 Commands**: Add more verb-noun mappings in `internal/layer1/static.go`
- **YAML Modules**: Create automation workflows in `modules/` directory
- **Documentation**: Improve guides, add translations
- **Testing**: Especially on different Android versions/devices

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) file.

## Acknowledgments

- Built with [Go](https://go.dev/)
- SQLite via [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go, no CGO)
- Inspired by natural language interfaces like GitHub Copilot CLI

---

**Made with ❤️ for the Termux community**

For more issues, see [GitHub Issues](https://github.com/themobileprof/clio/issues).
