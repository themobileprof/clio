# Building Clio for Termux/Android

## The SIGSYS Problem Explained

When running Go 1.24+ programs on Android/Termux that spawn subprocesses, you may encounter:

```
SIGSYS: bad system call
PC=0x1c280 m=2 sigcode=1
internal/syscall/unix.PidFDOpen
```

### What's Actually Happening

**Important clarification**: This crash happens during **module execution** (when spawning subprocesses), NOT during runtime initialization or basic queries.

✅ **Works fine**: Basic Clio queries like "extract tar file", REPL, search functionality
❌ **Crashes**: Running modules like `setup` that execute shell commands (if done in Go binary)

### The Call Chain (Old Approach)

```
User runs: clio >> setup
  → ExecuteModule()
    → executeStep() with type="command"
      → os/exec.Command("sh", "-c", cmd)
        → os/exec.Cmd.Start()
          → Go 1.24+ runtime probes pidfd_open(434)
            → Android seccomp returns SIGSYS (not ENOSYS)
              → CRASH ❌
```

### The Key Difference

- **ENOSYS** ("not implemented") → Go falls back gracefully ✅
- **SIGSYS** ("blocked by security policy") → Immediate crash ❌

Android uses seccomp-bpf to block newer syscalls for security. Go 1.24+ runtime doesn't expect SIGSYS during process spawning.

### Why Userspace Fixes Don't Work

Initial attempts to fix this via `SysProcAttr` modifications proved ineffective:

```go
// ❌ DOESN'T WORK - PidFD is just storage, not a control flag
cmd.SysProcAttr.PidFD = nil

// ✅ WORKS for clone3, but doesn't help with pidfd_open
cmd.SysProcAttr.Cloneflags = 0
```

**Why?** Go 1.24+'s runtime still **probes** for `pidfd_open` availability during subprocess setup, regardless of `PidFD` pointer value. The Android kernel responds with SIGSYS (fatal), not ENOSYS (graceful fallback).

### The Solution: External Script Executor

Since userspace fixes are insufficient, Clio now separates concerns:

1. **Go binary (`clio`)**: Handles all query/search functionality (no subprocess spawning)
2. **Bash script (`clio-run-module`)**: Executes YAML module workflows (pure bash)

**New execution flow:**

```
User runs: clio-run-module termux_setup setup
  → Bash script queries SQLite DB (using sqlite3 CLI)
    → Bash parses YAML and executes shell commands
      → Bash doesn't use Go runtime at all
        → SUCCESS ✅
```

**Advantages:**
- ✅ Go binary works perfectly for all search/REPL functionality
- ✅ Module execution works without syscall restrictions
- ✅ No Go version constraints (can use latest Go 1.24+)
- ✅ Clean separation of concerns
- ✅ Install script provides both tools automatically

## Building for Termux

### Option 1: Use Pre-Built Binaries (Recommended)

```bash
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh | bash
```

### Option 2: Build on Termux

**Prerequisites:**
```bash
pkg update
pkg install golang git
```

**Build:**
```bash
git clone https://github.com/themobileprof/clio.git
cd clio
chmod +x build-termux.sh
./build-termux.sh
```

**Install:**
```bash
cp clio $PREFIX/bin/
```

### Option 3: Cross-Compile from Linux/Mac

**For ARM64 (most Termux devices):**
```bash
GOTOOLCHAIN=go1.23.8 CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -ldflags="-s -w" -o clio-arm64 ./cmd/clio
```

**For ARM (32-bit devices):**
```bash
GOTOOLCHAIN=go1.23.8 CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 \
  go build -ldflags="-s -w" -o clio-arm ./cmd/clio
```

**Transfer to device:**
```bash
# Via adb
adb push clio-arm64 /data/data/com.termux/files/usr/bin/clio
adb shell chmod +x /data/data/com.termux/files/usr/bin/clio

# Or via Termux over SSH
scp clio-arm64 termux@device:/data/data/com.termux/files/usr/bin/clio
```

### Option 4: Temporary Fix (Not Recommended)

If you control the container/device seccomp policy:

```bash
# Disable seccomp entirely (DANGEROUS)
docker run --security-opt seccomp=unconfined ...

# Or add pidfd_open (434) to your seccomp allowlist
```

**Warning:** This weakens Android's security model. Only do this if you understand the implications.

## Technical Details

### Why External Script Instead of Go Version Downgrade?

Initial attempts considered using Go 1.22 (which doesn't use `pidfd_open`), but this approach had problems:

**Go 1.22 approach:**
- ❌ Dependencies require Go 1.24+ (modernc.org/sqlite)
- ❌ Go 1.22 reached EOL in March 2024
- ❌ Security vulnerabilities accumulate over time
- ❌ Limits development toolchain options

**External script approach (current solution):**
- ✅ Main binary uses latest Go 1.24+ (best performance, security)
- ✅ Module execution via Python (no syscall restrictions)
- ✅ Clean separation: search/query vs automation
- ✅ No compromise on dependencies or toolchain
- ✅ Python 3 is standard on Termux

### How the External Script Works

The `clio-run-module` script installed alongside `clio`:

1. **Reads** YAML module definitions from `~/.clio/modules.db` (using `sqlite3` CLI)
2. **Parses** YAML using bash regex and text processing
3. **Executes** shell commands via bash `eval`
4. **Handles** all step types: message, confirm, input, command, section, etc.

**Key advantage:** Bash doesn't involve Go runtime at all, completely avoiding `pidfd_open` and other restricted syscalls.

### Script Dependencies

The `clio-run-module` bash script requires:
- ✅ **sqlite3 CLI** - Pre-installed on Termux, reads from `~/.clio/modules.db`
- ✅ **Standard bash** - Available everywhere (Termux, Linux, macOS)
- ❌ **No Python** - Removed to avoid tooling/pip dependencies
- ❌ **No yq or external YAML parsers** - Simple bash regex parsing sufficient

**Why this matters:** Users can `curl install.sh | bash` and everything just works. No package managers, no pip installs, no setup required.

### Installation Flow

When users run the `install.sh` script:

1. Downloads the `clio` binary from GitHub releases
2. Installs to `$PREFIX/bin/clio` (Termux) or `/usr/local/bin/clio`
3. **Creates** `clio-run-module` script in the same directory
4. Sets executable permissions on both files

### CGO Disabled

`CGO_ENABLED=0` ensures:
- No libc dependencies
- Pure Go binary that works on all Android versions
- Smaller binary size
- Faster startup time

The `modernc.org/sqlite` dependency is CGO-free, so database functionality works without CGO.

## Verification

After installing, verify it works:

```bash
# Test the main binary
clio
>> list files
>> sync

# Test module execution
clio-run-module termux_setup setup
```

If you encounter issues:

1. **Check Python 3**: `python3 --version` (required for module runner)
2. **Check scripts installed**: `which clio clio-run-module`
3. **Check database**: `ls -lh ~/.clio/modules.db` (created after first `sync`)
4. **Report issue**: https://github.com/themobileprof/clio/issues

## Known Issues

### Issue: "clio-run-module: command not found"

**Symptom:** Script not found after installation

**Cause:** Install script didn't complete or PATH issue

**Fix:** 
```bash
# Check if it exists
ls -l $PREFIX/bin/clio-run-module  # Termux
ls -l ~/.local/bin/clio-run-module # Linux

# If missing, reinstall
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh | bash
```

### Issue: "Module database not found"

**Symptom:** clio-run-module can't find modules

**Cause:** Haven't run 'sync' yet

**Fix:** Run `clio`, then type `sync` to download modules

### Issue: "sqlite3 not found"

**Symptom:** clio-run-module fails immediately

**Cause:** sqlite3 CLI not available (rare - should be pre-installed)

**Fix:** `pkg install sqlite` (on Termux)

### Issue: "cannot execute binary file"

**Symptom:** `clio` binary won't run on device

**Cause:** Architecture mismatch (amd64 binary on ARM device)

**Fix:** Install script should detect this automatically. If building manually:
- Most Termux: `GOARCH=arm64`
- Older devices: `GOARCH=arm GOARM=7`

### Issue: "permission denied"

**Symptom:** Can't execute after copying to $PREFIX/bin

**Fix:** `chmod +x $PREFIX/bin/clio $PREFIX/bin/clio-run-module`

## References

- [Go on Android](https://github.com/golang/go/wiki/Mobile)
- [Android Seccomp Filters](https://source.android.com/docs/security/features/seccomp)
- [Go 1.24 Syscall Changes](https://go.dev/doc/go1.24#runtime)
- [Termux Package Management](https://wiki.termux.com/wiki/Package_Management)
