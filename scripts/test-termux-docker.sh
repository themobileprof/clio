#!/usr/bin/env bash
# Test Clio binaries inside Termux Docker (32-bit arm and 64-bit aarch64).
set -uo pipefail

CLIO_REPO="$(cd "$(dirname "$0")/.." && pwd)"
PLATFORM="${1:-arm}"  # arm (32-bit) or aarch64 (64-bit)
BINARY="/tmp/clio-${PLATFORM}"

case "$PLATFORM" in
  arm)
    IMAGE="termux/termux-docker:arm"
    DOCKER_PLATFORM="linux/arm/v7"
    GOARCH="arm"
    GOARM="7"
    ;;
  aarch64|arm64)
    IMAGE="termux/termux-docker:aarch64"
    DOCKER_PLATFORM="linux/arm64"
    GOARCH="arm64"
    GOARM=""
    ;;
  *)
    echo "Usage: $0 [arm|aarch64]"
    exit 1
    ;;
esac

echo "=== Building clio for ${GOARCH} ==="
cd "$CLIO_REPO"
if [ -n "$GOARM" ]; then
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" GOARM="$GOARM" \
    go build -ldflags="-s -w" -o "$BINARY" ./cmd/clio
else
  CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
    go build -ldflags="-s -w" -o "$BINARY" ./cmd/clio
fi
file "$BINARY"
echo ""

echo "=== Testing in ${IMAGE} (${DOCKER_PLATFORM}) ==="
export TERMUX_VERSION=1

queries=(
  "list files"
  "check disk space"
  "wetin dey inside folder"
  "my phone storage don full"
  "data no dey work"
  "how do I copy files"
  "chekc disk space"
  "where am I"
)

PASS=0
FAIL=0

for q in "${queries[@]}"; do
  out=$(docker run --rm --platform "$DOCKER_PLATFORM" --privileged \
    -e TERMUX_VERSION=1 \
    -v "$BINARY:/clio:ro" \
    "$IMAGE" \
    sh -c "echo '$q' | /clio" 2>/dev/null) || out="EXEC_FAIL"

  if [[ -z "$out" || "$out" == "EXEC_FAIL" || "$out" == *"no match"* ]]; then
    echo "FAIL  | $q => ${out:-empty}"
    ((FAIL++)) || true
  else
    echo "PASS  | $q => $out"
    ((PASS++)) || true
  fi
done

echo ""
echo "Results (${PLATFORM}): $PASS passed, $FAIL failed"
exit $([[ $FAIL -eq 0 ]] && echo 0 || echo 1)
