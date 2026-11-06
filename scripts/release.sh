#!/bin/bash
# Publish Clio binaries with gh. GitHub Actions cannot start jobs while the
# account is locked for billing. The registry picks up install.sh from git on
# its next restart, so this script does not upload the installer.

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TAG="${1:-}"
NOTES="${2:-}"
OUT="${RELEASE_DIR:-$(mktemp -d)}"

if [ -z "$TAG" ]; then
  echo "Usage: scripts/release.sh v0.0.28 \"what changed\""
  exit 1
fi

cd "$REPO_DIR"
mkdir -p "$OUT"

build() {
  local goos="$1" goarch="$2" goarm="${3:-}"
  local name="clio-${goos}-${goarch}"
  echo "Building $name..."
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" \
    go build -ldflags="-s -w" -o "$OUT/$name" ./cmd/clio
}

build linux amd64
build linux arm64
build linux arm 7
build linux 386
build darwin amd64
build darwin arm64

gh release create "$TAG" --target main --title "$TAG" --notes "${NOTES:-$TAG}" \
  "$OUT/clio-linux-amd64" \
  "$OUT/clio-linux-arm64" \
  "$OUT/clio-linux-arm" \
  "$OUT/clio-linux-386" \
  "$OUT/clio-darwin-amd64" \
  "$OUT/clio-darwin-arm64"

echo "https://github.com/themobileprof/clio/releases/tag/$TAG"
