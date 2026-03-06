# Building Clio for Termux/Android

## The SIGSYS Problem

When building Go programs for Android/Termux, you may encounter:

```
SIGSYS: bad system call
PC=0x1c280 m=2 sigcode=1
internal/syscall/unix.PidFDOpen
```

This happens because:

1. **Go 1.24+ uses newer syscalls** like `pidfd_open`, `clone3`, `faccessat2`
2. **Android's seccomp filters block these syscalls** on many devices
3. **Architecture mismatches** (amd64 toolchains on ARM devices) cause additional issues

## Solution

### Option 1: Use Pre-Built Binaries (Recommended)

```bash
curl -sfL https://raw.githubusercontent.com/themobileprof/clio/main/install.sh | bash
```

The install script automatically detects Termux and downloads the correct ARM binary.

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
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o clio-arm64 ./cmd/clio
```

**For ARM (32-bit devices):**
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -o clio-arm ./cmd/clio
```

**Transfer to device:**
```bash
adb push clio-arm64 /data/data/com.termux/files/usr/bin/clio
adb shell chmod +x /data/data/com.termux/files/usr/bin/clio
```

## Technical Details

### Syscall Compatibility

Clio implements several mitigations in `internal/safeexec/`:

1. **Custom LookPath**: Avoids `faccessat2` by manually checking file permissions
2. **Cloneflags = 0**: Forces use of legacy `clone()` instead of `clone3()`
3. **PidFD = false**: Disables `pidfd_open` usage (Go 1.24+)
4. **Go 1.23**: Uses Go version before `pidfd_open` was introduced

### Why Go 1.24 with Safeexec?

Go 1.24 introduced `pidfd_open` for better process management, but Android's seccomp filters block this syscall. The solution:

- ✅ **Keep Go 1.24** - Required by modernc.org/sqlite dependency
- ✅ **Runtime mitigation** - `safeexec` wrapper sets `Cloneflags = 0` to force older syscalls
- ✅ **Custom LookPath** - Avoids `faccessat2` by manually checking permissions
- ✅ **Build tags** - Platform-specific code for Linux vs others

This approach:
- Works with current dependencies
- Doesn't require Go version downgrade
- Handles syscall blocking at runtime
- Compatible with Android 7+ (Termux minimum)

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
