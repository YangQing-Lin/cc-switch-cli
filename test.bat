@echo off
setlocal

set "ROOT_DIR=%~dp0"
cd /d "%ROOT_DIR%"

set "CACHE_ROOT=%ROOT_DIR%.cache"
set "GOPATH=%CACHE_ROOT%\\gopath"
set "GOCACHE=%CACHE_ROOT%\\gocache"
set "GOMODCACHE=%CACHE_ROOT%\\gomodcache"

if not exist "%GOPATH%" mkdir "%GOPATH%"
if not exist "%GOCACHE%" mkdir "%GOCACHE%"
if not exist "%GOMODCACHE%" mkdir "%GOMODCACHE%"

echo [cc-switch-cli] GOPATH=%GOPATH%
echo [cc-switch-cli] GOCACHE=%GOCACHE%
echo [cc-switch-cli] GOMODCACHE=%GOMODCACHE%

go test -coverprofile=coverage.out -covermode=count ./...
exit /b %errorlevel%
