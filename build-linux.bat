@echo off
chcp 65001 >nul
title Gbootcms Linux Build ^& Package

set APP_DIR=%~dp0
set DIST_DIR=%APP_DIR%dist
set PKG_NAME=gbootcms-linux
set PKG_DIR=%DIST_DIR%\%PKG_NAME%
set LINUX_PORT=8080

echo ========================================
echo  Gbootcms Linux Builder ^& Packager (Ultra-Lite Production)
echo  Target: linux/amd64  Port: %LINUX_PORT%
echo ========================================
echo.

:: ---- Step 1: Cross-compile ----
echo [Step 1/3] Cross-compiling for linux/amd64 (stripped)...
cd /d "%APP_DIR%"
set GOPROXY=https://goproxy.cn,direct
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_COMMIT=%%i
if "%GIT_COMMIT%"=="" set GIT_COMMIT=release
for /f "tokens=*" %%i in ('powershell -Command "Get-Date -Format 'yyyy-MM-dd HH:mm'"') do set BUILD_TIME=%%i
go build -ldflags="-s -w -buildid= -X 'gbootcms/core/version.GitCommit=%GIT_COMMIT%' -X 'gbootcms/core/version.BuildTime=%BUILD_TIME%' -X 'gbootcms/core/version.Version=v1.0.0'" -trimpath -o "%APP_DIR%bin\gbootcms" .
if %ERRORLEVEL% neq 0 (
    echo [FAILED] Build error.
    pause
    exit /b 1
)
set GOOS=
set GOARCH=
echo [OK] Binary built: bin\gbootcms

:: UPX executable compression
if exist "%APP_DIR%tools\upx.exe" (
    echo [INFO] Running UPX executable compression...
    "%APP_DIR%tools\upx.exe" --best --lzma "%APP_DIR%bin\gbootcms" >nul 2>&1
    echo [OK] Binary compressed with UPX.
)
echo.

:: ---- Step 2: Clean old package ----
echo [Step 2/3] Preparing package directory...
if exist "%PKG_DIR%" rmdir /s /q "%PKG_DIR%"
mkdir "%PKG_DIR%"
mkdir "%PKG_DIR%\bin"
mkdir "%PKG_DIR%\config"
mkdir "%PKG_DIR%\data"
mkdir "%PKG_DIR%\template"
mkdir "%PKG_DIR%\static"
mkdir "%PKG_DIR%\apps\admin\view"
echo [OK] Directory structure created.
echo.

:: ---- Step 3: Copy files ----
echo [Step 3/3] Copying files...

:: Binary
copy "%APP_DIR%bin\gbootcms" "%PKG_DIR%\bin\" >nul

:: Config: Generate safe default config template without overwriting existing production config.json
if exist "%APP_DIR%config\config.example.json" (
    powershell -Command "(Get-Content '%APP_DIR%config\config.example.json') -replace '\"port\": 8080', '\"port\": %LINUX_PORT%' | Set-Content '%PKG_DIR%\config\config.json.default' -Encoding UTF8"
)

:: Database: Package default complete database template so new installs have full content and no captcha,
:: without overwriting an existing production database
if exist "%APP_DIR%data\0f13e78cfb83ebff93064bc3b9b7157d.db" (
    copy "%APP_DIR%data\0f13e78cfb83ebff93064bc3b9b7157d.db" "%PKG_DIR%\data\data.default.db" >nul
)
echo. > "%PKG_DIR%\data\.gitkeep"

:: Templates
xcopy "%APP_DIR%template" "%PKG_DIR%\template" /E /I /Q /Y >nul

:: Static (Including demo uploads, excluding backups)
xcopy "%APP_DIR%static" "%PKG_DIR%\static" /E /I /Q /Y >nul
if not exist "%PKG_DIR%\static\upload" mkdir "%PKG_DIR%\static\upload"
if exist "%PKG_DIR%\static\backup" rmdir /s /q "%PKG_DIR%\static\backup"
mkdir "%PKG_DIR%\static\backup"
echo. > "%PKG_DIR%\static\backup\.gitkeep"

:: Clean source maps and redundant legacy files from package
del /s /q "%PKG_DIR%\*.map" >nul 2>&1
if exist "%PKG_DIR%\static\admin\font-awesome" rmdir /s /q "%PKG_DIR%\static\admin\font-awesome"
if exist "%PKG_DIR%\static\admin\layui\images\face" rmdir /s /q "%PKG_DIR%\static\admin\layui\images\face"
if exist "%PKG_DIR%\static\admin\layui\lay\modules" rmdir /s /q "%PKG_DIR%\static\admin\layui\lay\modules"
if exist "%PKG_DIR%\wiki" rmdir /s /q "%PKG_DIR%\wiki"
if exist "%PKG_DIR%\docs" rmdir /s /q "%PKG_DIR%\docs"
if exist "%PKG_DIR%\tools" rmdir /s /q "%PKG_DIR%\tools"

:: Admin views
xcopy "%APP_DIR%apps\admin\view" "%PKG_DIR%\apps\admin\view" /E /I /Q /Y >nul

:: Start script (Safely auto-initializes config.json and database if not present)
(
echo #!/bin/bash
echo cd "$(dirname "$0")"
echo if [ ! -f "data/0f13e78cfb83ebff93064bc3b9b7157d.db" ] ^&^& [ -f "data/data.default.db" ]; then
echo     cp data/data.default.db data/0f13e78cfb83ebff93064bc3b9b7157d.db
echo     chmod 666 data/0f13e78cfb83ebff93064bc3b9b7157d.db
echo fi
echo if [ ! -f "config/config.json" ] ^&^& [ -f "config/config.json.default" ]; then
echo     cp config/config.json.default config/config.json
echo fi
echo chmod +x ./bin/gbootcms
echo ./bin/gbootcms
) > "%PKG_DIR%\start.sh"

:: systemd service file
(
echo [Unit]
echo Description=Gbootcms CMS Service
echo After=network.target
echo.
echo [Service]
echo Type=simple
echo WorkingDirectory=/opt/gbootcms
echo ExecStart=/opt/gbootcms/bin/gbootcms
echo Restart=on-failure
echo RestartSec=5
echo.
echo [Install]
echo WantedBy=multi-user.target
) > "%PKG_DIR%\gbootcms.service"

echo [OK] Files copied.
echo.

:: ---- Step 4: Create tarball ----
echo [Step 3/3] Creating tarball...
if exist "%DIST_DIR%\%PKG_NAME%.tar.gz" del "%DIST_DIR%\%PKG_NAME%.tar.gz"
cd /d "%DIST_DIR%"
tar -czf "%PKG_NAME%.tar.gz" "%PKG_NAME%"
if %ERRORLEVEL% neq 0 (
    echo [WARN] tar failed, trying 7z...
    where 7z >nul 2>&1
    if %ERRORLEVEL% equ 0 (
        7z a -ttar "%PKG_NAME%.tar" "%PKG_NAME%" >nul
        7z a -tgzip "%PKG_NAME%.tar.gz" "%PKG_NAME%.tar" >nul
        del "%PKG_NAME%.tar" 2>nul
    ) else (
        echo [FAILED] Neither tar nor 7z available.
        pause
        exit /b 1
    )
)

:: Cleanup temp dir
rmdir /s /q "%PKG_DIR%"

powershell -Command "$file = '%DIST_DIR%\%PKG_NAME%.tar.gz'; if (Test-Path $file) { $s = [math]::Round((Get-Item $file).Length / 1MB, 2); Write-Host \"[OK] Ultra-Lite Production Package: $file ($s MB)\" -ForegroundColor Green }"
echo.
echo ========================================
echo  Package: dist\%PKG_NAME%.tar.gz
echo  Port:    %LINUX_PORT%
echo ========================================
echo.