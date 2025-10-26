#!/bin/bash
set -euo pipefail

# 此脚本用于生成 Windows 资源文件 (.syso)
# GoReleaser 会在构建 Windows 版本前调用此脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "==> Generating Windows resource file (.syso)..."

# 检查是否在 Windows 或有 Wine 环境
if command -v x86_64-w64-mingw32-windres &> /dev/null; then
    # Linux with mingw-w64
    RC_COMPILER="x86_64-w64-mingw32-windres"
elif command -v windres &> /dev/null; then
    # Windows with MinGW
    RC_COMPILER="windres"
else
    echo "Warning: windres not found. Installing goversioninfo as fallback..."
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

    # 使用 goversioninfo（纯 Go 实现）
    cd "$PROJECT_ROOT"

    # 创建 versioninfo.json
    cat > versioninfo.json <<EOF
{
    "FixedFileInfo": {
        "FileVersion": {
            "Major": 1,
            "Minor": 0,
            "Patch": 0,
            "Build": 0
        },
        "ProductVersion": {
            "Major": 1,
            "Minor": 0,
            "Patch": 0,
            "Build": 0
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
        "FileVersion": "1.0.0.0",
        "InternalName": "ccs",
        "LegalCopyright": "Copyright (c) 2025 YangQing-Lin",
        "LegalTrademarks": "",
        "OriginalFilename": "ccs.exe",
        "PrivateBuild": "",
        "ProductName": "CC Switch CLI",
        "ProductVersion": "1.0.0.0",
        "SpecialBuild": ""
    },
    "VarFileInfo": {
        "Translation": {
            "LangID": "0409",
            "CharsetID": "04B0"
        }
    },
    "IconPath": "build/windows/icon.ico",
    "ManifestPath": ""
}
EOF

    goversioninfo -o resource_windows_amd64.syso
    echo "✓ Generated resource_windows_amd64.syso using goversioninfo"
    exit 0
fi

# 使用 windres 编译
cd "$SCRIPT_DIR"
$RC_COMPILER -i resource.rc -o "$PROJECT_ROOT/resource_windows_amd64.syso" -O coff

echo "✓ Generated resource_windows_amd64.syso using $RC_COMPILER"
