#!/bin/bash
set -euo pipefail

# 此脚本用于生成 Windows 资源文件 (.syso)
# GoReleaser 会在构建 Windows 版本前调用此脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "==> Generating Windows resource file (.syso)..."

log() {
    echo "==> $*"
}

get_version() {
    local version

    if [[ -n "${CCS_VERSION:-}" ]]; then
        version="${CCS_VERSION#v}"
        echo "$version"
        return
    fi

    version="$(git -C "$PROJECT_ROOT" describe --tags --abbrev=0 2>/dev/null || true)"
    if [[ -z "$version" && -f "$PROJECT_ROOT/internal/version/version.go" ]]; then
        version="$(grep -E '^const Version' "$PROJECT_ROOT/internal/version/version.go" | head -n1 | sed -E 's/.*\"([^\"]+)\".*/\1/' || true)"
    fi

    version="${version#v}"
    if [[ -z "$version" ]]; then
        version="0.0.0"
    fi

    echo "$version"
}

ensure_goversioninfo() {
    if command -v goversioninfo &> /dev/null; then
        return
    fi

    log "goversioninfo not found. Installing as fallback..."
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
}

create_versioninfo_json() {
    local target="$1"

    cat > "$target" <<EOF
{
    "FixedFileInfo": {
        "FileVersion": {
            "Major": ${VERSION_MAJOR},
            "Minor": ${VERSION_MINOR},
            "Patch": ${VERSION_PATCH},
            "Build": ${VERSION_BUILD}
        },
        "ProductVersion": {
            "Major": ${VERSION_MAJOR},
            "Minor": ${VERSION_MINOR},
            "Patch": ${VERSION_PATCH},
            "Build": ${VERSION_BUILD}
        },
        "FileFlagsMask": "3f",
        "FileFlags ": "00",
        "FileOS": "040004",
        "FileType": "01",
        "FileSubType": "00"
    },
    "StringFileInfo": {
        "Comments": "Cross-platform ClaudeCode/Codex configuration management tool",
        "CompanyName": "CC Switch CLI",
        "FileDescription": "CC Switch CLI - Configuration Management Tool",
        "FileVersion": "${VERSION_STRING}",
        "InternalName": "ccs",
        "LegalCopyright": "Copyright (c) 2025 YangQing-Lin",
        "LegalTrademarks": "",
        "OriginalFilename": "ccs.exe",
        "PrivateBuild": "",
        "ProductName": "CC Switch CLI",
        "ProductVersion": "${VERSION_STRING}",
        "SpecialBuild": ""
    },
    "VarFileInfo": {
        "Translation": {
            "LangID": "0409",
            "CharsetID": "04B0"
        }
    },
    "IconPath": "build/windows/icon.ico",
    "ManifestPath": "build/windows/ccs.exe.manifest"
}
EOF
}

generate_rc() {
    cat > "$RC_FILE" <<EOF
#include <windows.h>

IDI_ICON1 ICON "${ICON_PATH}"
CREATEPROCESS_MANIFEST_RESOURCE_ID RT_MANIFEST "${MANIFEST_PATH}"

VS_VERSION_INFO VERSIONINFO
 FILEVERSION    ${VERSION_MAJOR},${VERSION_MINOR},${VERSION_PATCH},${VERSION_BUILD}
 PRODUCTVERSION ${VERSION_MAJOR},${VERSION_MINOR},${VERSION_PATCH},${VERSION_BUILD}
 FILEFLAGSMASK  0x3fL
#ifdef _DEBUG
 FILEFLAGS      0x1L
#else
 FILEFLAGS      0x0L
#endif
 FILEOS         VOS_NT_WINDOWS32
 FILETYPE       VFT_APP
 FILESUBTYPE    VFT2_UNKNOWN
BEGIN
    BLOCK "StringFileInfo"
    BEGIN
        BLOCK "040904b0"
        BEGIN
            VALUE "CompanyName",      "CC Switch CLI\\0"
            VALUE "FileDescription",  "Cross-platform ClaudeCode/Codex configuration management tool\\0"
            VALUE "FileVersion",      "${VERSION_STRING}\\0"
            VALUE "InternalName",     "ccs\\0"
            VALUE "LegalCopyright",   "Copyright (c) 2025 YangQing-Lin\\0"
            VALUE "OriginalFilename", "ccs.exe\\0"
            VALUE "ProductName",      "CC Switch CLI\\0"
            VALUE "ProductVersion",   "${VERSION_STRING}\\0"
        END
    END
    BLOCK "VarFileInfo"
    BEGIN
        VALUE "Translation", 0x409, 1200
    END
END
EOF
}

build_with_windres() {
    local arch="$1"
    local target="$2"
    local output="$PROJECT_ROOT/resource_windows_${arch}.syso"

    if "$RC_COMPILER" --target="$target" -i "$RC_FILE" -o "$output" -O coff; then
        log "Generated ${output} using ${RC_COMPILER} (target=${target})"
        return 0
    fi

    return 1
}

VERSION="$(get_version)"
VERSION_BASE="${VERSION%%+*}"
VERSION_BASE="${VERSION_BASE%%-*}"
IFS='.' read -r VERSION_MAJOR VERSION_MINOR VERSION_PATCH VERSION_BUILD <<<"$VERSION_BASE"
VERSION_MAJOR=${VERSION_MAJOR:-0}
VERSION_MINOR=${VERSION_MINOR:-0}
VERSION_PATCH=${VERSION_PATCH:-0}
VERSION_BUILD=${VERSION_BUILD:-0}
VERSION_STRING="${VERSION_MAJOR}.${VERSION_MINOR}.${VERSION_PATCH}.${VERSION_BUILD}"

ICON_PATH="$SCRIPT_DIR/icon.ico"
MANIFEST_PATH="$SCRIPT_DIR/ccs.exe.manifest"
RC_FILE="$(mktemp "${SCRIPT_DIR}/resource.XXXXXX.rc")"
VERSIONINFO_JSON="$(mktemp)"
trap 'rm -f "$RC_FILE" "$VERSIONINFO_JSON"' EXIT

log "Detected version: ${VERSION_STRING}"

if command -v x86_64-w64-mingw32-windres &> /dev/null; then
    RC_COMPILER="x86_64-w64-mingw32-windres"
elif command -v windres &> /dev/null; then
    RC_COMPILER="windres"
else
    RC_COMPILER=""
fi

generate_rc

if [[ -n "$RC_COMPILER" ]]; then
    log "Using ${RC_COMPILER} to generate resource files"

    if ! build_with_windres "amd64" "pe-x86-64"; then
        log "windres failed for amd64. Falling back to goversioninfo..."
        ensure_goversioninfo
        create_versioninfo_json "$VERSIONINFO_JSON"
        (cd "$PROJECT_ROOT" && GOOS=windows GOARCH=amd64 goversioninfo -o resource_windows_amd64.syso "$VERSIONINFO_JSON")
    fi

    if ! build_with_windres "arm64" "pe-aarch64"; then
        log "windres failed for arm64 target pe-aarch64, retrying pe-arm64..."
        if ! build_with_windres "arm64" "pe-arm64"; then
            log "windres cannot build arm64 resource. Falling back to goversioninfo for arm64..."
            ensure_goversioninfo
            create_versioninfo_json "$VERSIONINFO_JSON"
            (cd "$PROJECT_ROOT" && GOOS=windows GOARCH=arm64 goversioninfo -o resource_windows_arm64.syso "$VERSIONINFO_JSON")
        fi
    fi
else
    log "windres not found. Using goversioninfo fallback..."
    ensure_goversioninfo
    create_versioninfo_json "$VERSIONINFO_JSON"
    (cd "$PROJECT_ROOT" && GOOS=windows GOARCH=amd64 goversioninfo -o resource_windows_amd64.syso "$VERSIONINFO_JSON")
    (cd "$PROJECT_ROOT" && GOOS=windows GOARCH=arm64 goversioninfo -o resource_windows_arm64.syso "$VERSIONINFO_JSON")
fi

log "✓ Windows resource generation completed"
