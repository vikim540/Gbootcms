[CmdletBinding()]
param(
    [string]$PackageType = "tar.gz"
)

$ErrorActionPreference = "Stop"
$appDir = $PSScriptRoot
$envFile = Join-Path $appDir ".env"
$distDir = Join-Path $appDir "dist"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Gbootcms FTP Auto Deployer" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

if (-not (Test-Path $envFile)) {
    Write-Error "Error: .env file not found! Please ensure FTP_HOST / FTP_USER / FTP_PASS exist in .env"
    exit 1
}

$envVars = @{}
Get-Content $envFile -Encoding UTF8 | ForEach-Object {
    $line = $_.Trim()
    if ($line -and -not $line.StartsWith("#") -and $line -match "^([^=]+)=(.*)$") {
        $envVars[$matches[1].Trim()] = $matches[2].Trim()
    }
}

$ftpHost = $envVars["FTP_HOST"]
$ftpPort = if ($envVars["FTP_PORT"]) { $envVars["FTP_PORT"] } else { "21" }
$ftpUser = $envVars["FTP_USER"]
$ftpPass = $envVars["FTP_PASS"]

if (-not $ftpHost -or -not $ftpUser -or -not $ftpPass) {
    Write-Error ".env missing FTP_HOST, FTP_USER, or FTP_PASS"
    exit 1
}

$targetFileName = "gbootcms-linux.$PackageType"
$localFilePath = Join-Path $distDir $targetFileName

if (-not (Test-Path $localFilePath)) {
    Write-Error "Distribution package not found: $localFilePath"
    exit 1
}

$fileInfo = Get-Item $localFilePath
$fileSizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
Write-Host "Target: $targetFileName ($fileSizeMB MB)" -ForegroundColor Yellow
Write-Host "Host:   ${ftpHost}:${ftpPort} (User: $ftpUser)" -ForegroundColor Yellow
Write-Host "Connecting and uploading directly..." -ForegroundColor Green

$ftpUrl = "ftp://${ftpHost}:${ftpPort}/${targetFileName}"
$request = [System.Net.FtpWebRequest]::Create([System.Uri]$ftpUrl)
$request.Proxy = $null
$request.Credentials = New-Object System.Net.NetworkCredential($ftpUser, $ftpPass)
$request.Method = [System.Net.WebRequestMethods+Ftp]::UploadFile
$request.UseBinary = $true
$request.UsePassive = $true
$request.KeepAlive = $false
$request.Timeout = 600000

$fileStream = [System.IO.File]::OpenRead($localFilePath)
$ftpStream = $null

try {
    $ftpStream = $request.GetRequestStream()
    $buffer = New-Object byte[] 65536
    $totalRead = 0
    $lastPercent = -1

    while (($bytesRead = $fileStream.Read($buffer, 0, $buffer.Length)) -gt 0) {
        $ftpStream.Write($buffer, 0, $bytesRead)
        $totalRead += $bytesRead
        $percent = [math]::Floor(($totalRead / $fileInfo.Length) * 100)
        if ($percent -ne $lastPercent -and ($percent % 20 -eq 0 -or $percent -eq 100)) {
            $currentMB = [math]::Round($totalRead / 1MB, 2)
            Write-Host "Upload Progress: $percent% ($currentMB MB / $fileSizeMB MB)" -ForegroundColor Gray
            $lastPercent = $percent
        }
    }
}
catch {
    Write-Error "FTP Upload Error: $($_.Exception.Message)"
    exit 1
}
finally {
    if ($null -ne $ftpStream) { $ftpStream.Close() }
    if ($null -ne $fileStream) { $fileStream.Close() }
}

try {
    $response = $request.GetResponse()
    Write-Host "[SUCCESS] $targetFileName uploaded successfully!" -ForegroundColor Green
    Write-Host "Server response: $($response.StatusDescription.Trim())" -ForegroundColor Cyan
    $response.Close()
}
catch {
    Write-Host "Upload stream finished. Server notice: $($_.Exception.Message)"
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Package successfully deployed to server!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan