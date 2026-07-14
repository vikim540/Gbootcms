@echo off
chcp 65001 >nul
title Gbootcms Linux Build & Package

set APP_DIR=%~dp0
set DIST_DIR=%APP_DIR%dist
set PKG_NAME=gbootcms-linux
set PKG_DIR=%DIST_DIR%\%PKG_NAME%
set LINUX_PORT=8080

echo ========================================
echo  Gbootcms Linux Builder & Packager
echo  Target: linux/amd64  Port: %LINUX_PORT%
echo ========================================
echo.

:: ---- Step 1: Cross-compile ----
echo [Step 1/4] Cross-compiling for linux/amd64...
cd /d "%APP_DIR%"
set GOPROXY=https://goproxy.cn,direct
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -o "%APP_DIR%bin\gbootcms" .
if %ERRORLEVEL% neq 0 (
    echo [FAILED] Build error.
    pause
    exit /b 1
)
set GOOS=
set GOARCH=
echo [OK] Binary built: bin\gbootcms
echo.

:: ---- Step 2: Clean old package ----
echo [Step 2/4] Preparing package directory...
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
echo [Step 3/4] Copying files...

:: Binary
copy "%APP_DIR%bin\gbootcms" "%PKG_DIR%\bin\" >nul

:: Config (port changed to 8081)
powershell -Command "(Get-Content '%APP_DIR%config\config.json') -replace '\"port\": 8080', '\"port\": %LINUX_PORT%' | Set-Content '%PKG_DIR%\config\config.json' -Encoding UTF8"

:: Database
copy "%APP_DIR%data\pbootcms.db" "%PKG_DIR%\data\" >nul

:: Templates
xcopy "%APP_DIR%template" "%PKG_DIR%\template" /E /I /Q /Y >nul

:: Static
xcopy "%APP_DIR%static" "%PKG_DIR%\static" /E /I /Q /Y >nul

:: Admin views
xcopy "%APP_DIR%apps\admin\view" "%PKG_DIR%\apps\admin\view" /E /I /Q /Y >nul

:: Start script
(
echo #!/bin/bash
echo cd "$(dirname "$0")"
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
echo [Step 4/4] Creating tarball...
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

echo [OK] Package created.
echo.
echo ========================================
echo  Package: dist\%PKG_NAME%.tar.gz
echo  Port:    %LINUX_PORT%
echo.
echo  Deploy steps:
echo  1. Upload to server:  scp dist\gbootcms-linux.tar.gz user@server:/opt/
echo  2. Extract:           cd /opt ^&^& tar xzf gbootcms-linux.tar.gz
echo  3. Rename:            mv gbootcms-linux gbootcms
echo  4. Quick run:         cd /opt/gbootcms ^&^& chmod +x start.sh ^&^& ./start.sh
echo  5. As service:        cp gbootcms.service /etc/systemd/system/
echo                        systemctl daemon-reload ^&^& systemctl enable gbootcms
echo                        systemctl start gbootcms
echo.
echo  Site:  http://your-server:8081
echo  Admin: http://your-server:8081/admin
echo ========================================
echo.
pause
