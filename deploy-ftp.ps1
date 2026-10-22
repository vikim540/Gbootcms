[CmdletBinding()]
param(
    [string]$PackageType = "tar.gz",
    [switch]$SkipRestart
)

$ErrorActionPreference = "Stop"
$appDir = $PSScriptRoot
$envFile = Join-Path $appDir ".env"
$distDir = Join-Path $appDir "dist"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Gbootcms Auto Deployer (FTP + 1Panel)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 啟用 TLS 1.2 並在需要時忽略自簽名證書 (相容 Windows PowerShell 5.1 與 PS 7)
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12 -bor [System.Net.SecurityProtocolType]::Tls11 -bor [System.Net.SecurityProtocolType]::Tls
[System.Net.ServicePointManager]::ServerCertificateValidationCallback = { $true }

# 1. 讀取 .env 配置
if (-not (Test-Path $envFile)) {
    Write-Error "Error: .env not found!"
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

$apiKey = $envVars["ONEPANEL_API_KEY"]
$panelUrl = $envVars["ONEPANEL_URL"]

# 2. 準備待上傳檔案
$targetFileName = "gbootcms-linux.$PackageType"
$localFilePath = Join-Path $distDir $targetFileName

if (-not (Test-Path $localFilePath)) {
    Write-Error "Package not found: $localFilePath"
    exit 1
}

$fileInfo = Get-Item $localFilePath
$fileSizeMB = [math]::Round($fileInfo.Length / 1MB, 2)
Write-Host "[1/3] 上傳部署包: $targetFileName ($fileSizeMB MB)" -ForegroundColor Yellow
Write-Host "目標主機: ${ftpHost}:${ftpPort} (使用者: $ftpUser)" -ForegroundColor Gray

# 3. FTP 上傳
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
        if ($percent -ne $lastPercent -and ($percent % 25 -eq 0 -or $percent -eq 100)) {
            $currentMB = [math]::Round($totalRead / 1MB, 2)
            Write-Host "上傳進度: $percent% ($currentMB MB / $fileSizeMB MB)" -ForegroundColor Gray
            $lastPercent = $percent
        }
    }
}
catch {
    Write-Error "FTP 上傳失敗: $($_.Exception.Message)"
    exit 1
}
finally {
    if ($null -ne $ftpStream) { $ftpStream.Close() }
    if ($null -ne $fileStream) { $fileStream.Close() }
}

try {
    $response = $request.GetResponse()
    Write-Host "[OK] FTP 上傳成功！" -ForegroundColor Green
    $response.Close()
} catch {}

# 4. 呼叫 1Panel API 自動解壓並重啟容器
if ($apiKey -and -not $SkipRestart) {
    Write-Host "[2/3] 正在透過 1Panel API 自動解壓部署包..." -ForegroundColor Yellow
    
    $timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds().ToString()
    $rawStr = "1panel" + $apiKey + $timestamp
    $md5 = [System.Security.Cryptography.MD5]::Create()
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($rawStr)
    $hash = $md5.ComputeHash($bytes)
    $token = [System.BitConverter]::ToString($hash).Replace("-", "").ToLower()

    $headers = @{
        "1Panel-Token" = $token
        "1Panel-Timestamp" = $timestamp
        "Host" = "sjc.pig.my.id"
    }

    # 解壓檔案
    $decompressBody = @{
        "path" = "/opt/1panel/www/sites/gocms.icun.eu.org/index/$targetFileName"
        "dst" = "/opt/1panel/www/sites/gocms.icun.eu.org/index"
        "type" = $PackageType
    } | ConvertTo-Json

    try {
        $decRes = Invoke-RestMethod -Uri "https://sjc.pig.my.id:2096/api/v2/files/decompress" -Headers $headers -Method Post -Body $decompressBody -ContentType "application/json" -TimeoutSec 15
        Write-Host "[OK] 伺服器解壓完成！" -ForegroundColor Green
    }
    catch {
        Write-Host "[警告] API 解壓請求失敗: $($_.Exception.Message)" -ForegroundColor Red
    }

    # 重啟運行環境 gbootcms_go_1_26 (Runtime ID 2)
    Write-Host "[3/3] 正在透過 1Panel API 重啟 Go 容器..." -ForegroundColor Yellow
    $restartBody = @{
        "id" = 2
        "operate" = "restart"
    } | ConvertTo-Json

    try {
        $reRes = Invoke-RestMethod -Uri "https://sjc.pig.my.id:2096/api/v2/runtimes/operate" -Headers $headers -Method Post -Body $restartBody -ContentType "application/json" -TimeoutSec 30
        Write-Host "[OK] 容器重啟完成！" -ForegroundColor Green
    }
    catch {
        Write-Host "[警告] API 重啟請求失敗: $($_.Exception.Message)" -ForegroundColor Red
    }

    # 健康檢查
    Start-Sleep -Seconds 5
    try {
        $chk = Invoke-WebRequest -Uri "https://gocms.icun.eu.org" -TimeoutSec 10
        Write-Host "網站運行狀態: HTTP $($chk.StatusCode) OK (https://gocms.icun.eu.org)" -ForegroundColor Green
    } catch {
        Write-Host "網站狀態檢查異常: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " 全自動流水線部署完成！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
