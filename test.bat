@echo off
REM CC-Switch CLI 测试脚本 (Windows)
REM 用于快速运行所有测试并生成报告

echo ======================================
echo   CC-Switch CLI 测试套件
echo ======================================
echo.

REM 1. 运行所有测试
echo [1/3] 运行所有测试...
go test ./internal/... -v
if %ERRORLEVEL% neq 0 (
    echo 测试失败！
    exit /b 1
)

echo.

REM 2. 生成覆盖率报告
echo [2/3] 生成覆盖率报告...
go test ./internal/... -cover

echo.

REM 3. 生成详细覆盖率文件（可选）
if "%1"=="--coverage" (
    echo [3/3] 生成详细覆盖率文件...
    go test -coverprofile=coverage.out ./internal/...
    go tool cover -html=coverage.out -o coverage.html
    echo √ 覆盖率报告已生成: coverage.html
)

echo.
echo ======================================
echo   √ 所有测试通过！
echo ======================================
