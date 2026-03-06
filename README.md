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

- **`setup`** - Run Termux setup wizard (Termux only, first-time use)
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

### Termux: "SIGSYS: bad system call" Error

**If building from source:**

1. **On Termux (recommended):**
   ```bash
   ./build-termux.sh
   cp clio $PREFIX/bin/
   ```

2. **Cross-compile from Linux/Mac:**
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o clio-arm64 ./cmd/clio
   ```

3. **For detailed instructions**, see [docs/TERMUX_BUILD_GUIDE.md](docs/TERMUX_BUILD_GUIDE.md)

**How it works:**
- Custom `safeexec` wrapper intercepts all command execution
- Sets `SysProcAttr.Cloneflags = 0` to force legacy `clone()` instead of `clone3()`
- Manual `LookPath` implementation avoids `faccessat2` syscall
- Platform-specific code via Go build tags (Linux vs others)
- CGO disabled for pure Go binary with no libc dependencies
- Compatible with Go 1.24+ while maintaining Android 7+ support

**Why not just downgrade Go?**
The `modernc.org/sqlite` dependency requires Go 1.24+, so we use runtime mitigation instead of version downgrade.

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
