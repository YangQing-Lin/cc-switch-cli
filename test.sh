#!/bin/bash

# CC-Switch CLI 测试脚本
# 用于快速运行所有测试并生成报告

set -e

echo "======================================"
echo "  CC-Switch CLI 测试套件"
echo "======================================"
echo ""

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. 运行所有测试
echo -e "${YELLOW}[1/3] 运行所有测试...${NC}"
go test ./internal/... -v

# 2. 生成覆盖率报告
echo ""
echo -e "${YELLOW}[2/3] 生成覆盖率报告...${NC}"
go test ./internal/... -cover

# 3. 生成详细覆盖率文件（可选）
if [ "$1" == "--coverage" ]; then
    echo ""
    echo -e "${YELLOW}[3/3] 生成详细覆盖率文件...${NC}"
    go test -coverprofile=coverage.out ./internal/...
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}✓ 覆盖率报告已生成: coverage.html${NC}"
fi

echo ""
echo -e "${GREEN}======================================"
echo -e "  ✓ 所有测试通过！"
echo -e "======================================${NC}"
