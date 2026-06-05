# build.ps1 - pbootcms-go native PowerShell build script
# Single-layer shell, $LASTEXITCODE and $env: variables are preserved naturally.

[CmdletBinding()]
param(
    [string]$ProjectRoot = "f:\mysite\AI\idea\pbootcmstogo\pbootcms-go",
    [string]$OutputDir   = "bin",
    [switch]$Run
)

$ErrorActionPreference = 'Stop'

Set-Location $ProjectRoot
Write-Host "=== Working in: $(Get-Location) ===" -ForegroundColor Cyan

# Ensure bin folder exists
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

# Pull deps
Write-Host ">>> go mod tidy" -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    throw "go mod tidy failed (exit=$LASTEXITCODE)"
}

# Build
$out = Join-Path $OutputDir "pbootcms-go.exe"
Write-Host ">>> go build -o $out ." -ForegroundColor Yellow
go build -o $out .
if ($LASTEXITCODE -ne 0) {
    throw "go build failed (exit=$LASTEXITCODE)"
}

Write-Host "---DONE--- build OK -> $out  exit=$LASTEXITCODE" -ForegroundColor Green

if ($Run) {
    Write-Host ">>> go run" -ForegroundColor Yellow
    & $out
    Write-Host "---DONE--- run exit=$LASTEXITCODE" -ForegroundColor Green
}
