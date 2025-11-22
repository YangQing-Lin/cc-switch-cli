#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

BUILD_DIR="$ROOT_DIR/build"
WINDOWS_RESOURCE_SCRIPT="$BUILD_DIR/windows/generate_syso.sh"

OS_LIST=(linux darwin windows)
ARCH_LIST=(amd64 arm64)

total_targets=$(( ${#OS_LIST[@]} * ${#ARCH_LIST[@]} ))

log() {
  echo "==> $*"
}

# ---------------------------------------------------------------------------
# Pre-build checks
# ---------------------------------------------------------------------------
log "Formatting code (go fmt ./...)"
go fmt ./...

log "Running tests (go test ./...)"
go test ./...

# Version metadata (aligned with GoReleaser ldflags)
VERSION="$(git describe --tags --abbrev=0 2>/dev/null || true)"
if [[ -z "$VERSION" ]]; then
  VERSION="$(grep -E '^const Version' internal/version/version.go | head -n1 | sed -E 's/.*\"([^\"]+)\".*/\1/')"
fi
if [[ -z "$VERSION" ]]; then
  VERSION="dev"
fi

BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
LDFLAGS="-X github.com/YangQing-Lin/cc-switch-cli/internal/version.Version=${VERSION} -X github.com/YangQing-Lin/cc-switch-cli/internal/version.BuildDate=${BUILD_DATE} -X github.com/YangQing-Lin/cc-switch-cli/internal/version.GitCommit=${GIT_COMMIT}"

log "Using version metadata: version=${VERSION}, date=${BUILD_DATE}, commit=${GIT_COMMIT}"

mkdir -p "$BUILD_DIR"

# Windows resource file (run once before Windows builds)
if printf '%s\0' "${OS_LIST[@]}" | grep -qz "windows"; then
  log "Generating Windows resource file (.syso)"
  bash "$WINDOWS_RESOURCE_SCRIPT"
fi

# ---------------------------------------------------------------------------
# Build targets in parallel
# ---------------------------------------------------------------------------
build_target() {
  local os="$1"
  local arch="$2"
  local index="$3"
  local output="$BUILD_DIR/ccs-${os}-${arch}"

  if [[ "$os" == "windows" ]]; then
    output="${output}.exe"
  fi

  log "[${index}/${total_targets}] Building ${os}/${arch} -> ${output}"

  if env CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "$LDFLAGS" -o "$output" ./main.go; then
    echo "✓ Built ${output}"
  else
    echo "✗ Build failed for ${os}/${arch}"
    return 1
  fi
}

pids=()
labels=()
index=0

for os in "${OS_LIST[@]}"; do
  for arch in "${ARCH_LIST[@]}"; do
    index=$((index + 1))
    build_target "$os" "$arch" "$index" &
    pids+=("$!")
    labels+=("${os}/${arch}")
  done
done

for i in "${!pids[@]}"; do
  if ! wait "${pids[$i]}"; then
    echo "Build failed for ${labels[$i]}"
    exit 1
  fi
done

log "All builds completed successfully. Artifacts are in $BUILD_DIR"
