#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

CACHE_ROOT="${ROOT_DIR}/.cache"
export GOPATH="${CACHE_ROOT}/gopath"
export GOCACHE="${CACHE_ROOT}/gocache"
export GOMODCACHE="${CACHE_ROOT}/gomodcache"

mkdir -p "${GOPATH}" "${GOCACHE}" "${GOMODCACHE}"

echo "[cc-switch-cli] GOPATH=${GOPATH}"
echo "[cc-switch-cli] GOCACHE=${GOCACHE}"
echo "[cc-switch-cli] GOMODCACHE=${GOMODCACHE}"

go test -coverprofile=coverage.out -covermode=count ./...
