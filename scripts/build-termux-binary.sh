#!/bin/bash
# arm64 binary that starts under Termux on Android 10 and Android 16.
#
# Android 10 execs a static ET_EXEC itself, so the ordinary linux binary works
# there. Android 16 sends every program through /system/bin/linker64, which
# rejects ET_EXEC ("has unexpected e_type: 2").
#
# GOOS=android with CGO disabled loads and then hangs before main, because the
# Android runtime needs cgo to set up the g register. This is a linux PIE
# instead: linker64 accepts the ELF type, and the Linux runtime starts on its own.
set -euo pipefail

OUT="${1:-clio-android-arm64}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -buildmode=pie -ldflags="-s -w -I /system/bin/linker64" -o "$OUT" ./cmd/clio
go run ./scripts/patch_termux_elf.go "$OUT"

if ! readelf -l "$OUT" | grep -q 'Requesting program interpreter: /system/bin/linker64'; then
  echo "missing linker64 interpreter in $OUT" >&2
  exit 1
fi
if ! readelf -h "$OUT" | grep -q 'DYN'; then
  echo "$OUT is not a PIE (Android 16 rejects e_type 2)" >&2
  exit 1
fi
if readelf -d "$OUT" | grep -q NEEDED; then
  echo "$OUT links a shared library; Termux cannot provide glibc" >&2
  exit 1
fi
echo "Built $OUT"
