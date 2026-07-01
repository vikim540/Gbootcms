@echo off
chcp 65001 >nul
title PbootCMS-Go Build & Restart

set APP_DIR=%~dp0
set BIN_PATH=%APP_DIR%bin\pbootcms-go.exe

echo ========================================
echo  PbootCMS-Go Builder + Restarter
echo ========================================
echo.

:: ---- Step 1: Build ----
echo [Step 1/3] Building...
cd /d "%APP_DIR%"
set GOPROXY=https://goproxy.cn,direct
go build -o "%BIN_PATH%" . 2>&1
if %ERRORLEVEL% neq 0 (
    echo.
    echo [FAILED] Build error. Check output above.
    pause
    exit /b 1
)
echo [OK] Build successful.
echo.

:: ---- Step 2: Stop old process ----
echo [Step 2/3] Checking for running process...
tasklist /FI "IMAGENAME eq pbootcms-go.exe" 2>nul | find /I "pbootcms-go.exe" >nul
if %ERRORLEVEL% equ 0 (
    echo [INFO] Found running process. Stopping...
    taskkill /F /IM pbootcms-go.exe >nul 2>&1
    if %ERRORLEVEL% equ 0 (
        echo [OK] Old process stopped.
    ) else (
        echo [WARN] Could not stop process. Try manually.
    )
) else (
    echo [INFO] No running process found.
)
echo.

:: Wait for port release
timeout /t 2 /nobreak >nul

:: ---- Step 3: Start new process ----
echo [Step 3/3] Starting new process...
start "PbootCMS-Go" "%BIN_PATH%"
if %ERRORLEVEL% equ 0 (
    echo [OK] Service started.
    echo.
    echo ========================================
    echo  Site:  http://localhost:8080
    echo  Admin: http://localhost:8080/admin
    echo ========================================
    echo.
) else (
    echo [FAILED] Could not start service.
    pause
    exit /b 1
)

pause