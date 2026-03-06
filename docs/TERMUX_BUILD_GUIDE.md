# Building Clio for Termux/Android

## The SIGSYS Problem Explained

When running Go programs on Android/Termux with Go 1.24+, you may encounter:

```
SIGSYS: bad system call
PC=0x1c280 m=2 sigcode=1
internal/syscall/unix.PidFDOpen
```

### What's Actually Happening

**Important clarification**: This crash happens during **module execution** (when spawning subprocesses), NOT during runtime initialization.

✅ **Works fine**: Basic Clio queries like "extract tar file"
❌ **Crashes**: Running modules like `setup` that execute shell commands

### The Call Chain

```
User runs: clio >> setup
  → ExecuteModule()
    → executeStep() with type="command"
      → safeexec.Command("sh", "-c", cmd)
        → os/exec.Cmd.Start()
          → Go 1.24 tries pidfd_open(434)
            → Android seccomp returns SIGSYS
              → CRASH ❌
```

### The Key Difference

- **ENOSYS** ("not implemented") → Go falls back gracefully ✅
- **SIGSYS** ("blocked by security policy") → Crash ❌

Android uses seccomp to block newer syscalls for security. Go 1.24 doesn't expect SIGSYS during process spawning.

### Why This Happens

1. **Go 1.24 uses newer syscalls**: `pidfd_open`, `clone3`, `faccessat2`
2. **Android's seccomp filters block these** on many devices for security
3. **Crash during subprocess spawning**: When modules execute commands
4. **Architecture mismatches**: Cross-compilation issues compound the problem

### The Solution: Runtime Mitigation

Since the crash happens during **subprocess spawning** (not runtime init), we CAN fix it in application code:

```go
// internal/safeexec/safeexec_linux.go
func Command(name string, arg ...string) *exec.Cmd {
    cmd := exec.Command(name, arg...)
    
    if cmd.SysProcAttr == nil {
        cmd.SysProcAttr = &syscall.SysProcAttr{}
    }
    
    // Disable pidfd_open - Go 1.24 won't try to use it
    cmd.SysProcAttr.PidFD = nil
    
    // Force legacy clone() instead of clone3()
    cmd.SysProcAttr.Cloneflags = 0
    
    return cmd
}
```

**What this fixes:**
- ✅ `pidfd_open` - Disabled by setting `PidFD = nil`
- ✅ `clone3` - Forced to legacy via `Cloneflags = 0`  
- ✅ `faccessat2` - Custom `LookPath` avoids it

**Why this works:**
Module commands go through our `safeexec.Command()` wrapper, which configures `SysProcAttr` before Go tries any syscalls. This is different from runtime initialization where we have no control.

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

### Why Go 1.23 Instead of 1.24?

Go 1.23 doesn't use `pidfd_open`, avoiding the blocked syscall entirely:

- ✅ No `pidfd_open` calls
- ✅ Still maintained (1.23.8 has security updates)
- ✅ Compatible with Android 7+ (Termux minimum)
- ✅ Works with all Clio dependencies

### Why Not Downgrade the Project?

Clio's `go.mod` specifies Go 1.24:

```go
go 1.24

toolchain go1.24.3
```

**Reasons:**
1. **Development on Linux/Mac works fine** with Go 1.24
2. **Dependencies benefit** from Go 1.24 improvements
3. **Only Termux has the issue** - project-wide downgrade is overkill
4. **GOTOOLCHAIN handles it cleanly** for Termux-specific builds

### The GOTOOLCHAIN Solution

Go 1.21+ supports toolchain selection via `GOTOOLCHAIN` environment variable:

```bash
GOTOOLCHAIN=go1.23.8 go build ./cmd/clio
```

This overrides the `go.mod` toolchain **only for that build**, allowing:
- Development with Go 1.24 on Linux/Mac
- Production builds with Go 1.23 for Termux
- No project-wide compromises


### CGO Disabled

`CGO_ENABLED=0` ensures:
- No libc dependencies
- Pure Go binary that works on all Android versions
- Smaller binary size
- Faster startup time

The `modernc.org/sqlite` dependency is CGO-free, so database functionality works without CGO.

## Verification

After building/installing, verify it works:

```bash
clio
>> list files
```

If you still get SIGSYS errors:

1. **Check Go version**: `go version` (should be 1.23.x)
2. **Check architecture**: `file clio` (should match your device)
3. **Check Android version**: `getprop ro.build.version.release` (should be 7+)
4. **Report issue**: https://github.com/themobileprof/clio/issues

## Known Issues

### Issue: "bad system call" on Go 1.24+

**Symptom:** SIGSYS when running setup or interactive commands

**Cause:** Go 1.24 uses `pidfd_open` which Android blocks

**Fix:** Downgraded to Go 1.23.5 in `go.mod`

### Issue: "cannot execute binary file"

**Symptom:** Binary won't run on device

**Cause:** Architecture mismatch (amd64 binary on ARM device)

**Fix:** Use correct GOARCH when building:
- Most Termux: `GOARCH=arm64`
- Older devices: `GOARCH=arm GOARM=7`

### Issue: "permission denied"

**Symptom:** Can't execute after copying to $PREFIX/bin

**Fix:** `chmod +x $PREFIX/bin/clio`

## References

- [Go on Android](https://github.com/golang/go/wiki/Mobile)
- [Android Seccomp Filters](https://source.android.com/docs/security/features/seccomp)
- [Go 1.24 Syscall Changes](https://go.dev/doc/go1.24#runtime)
- [Termux Package Management](https://wiki.termux.com/wiki/Package_Management)
