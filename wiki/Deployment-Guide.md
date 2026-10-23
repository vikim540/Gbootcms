# 生產部署與自動化 (Deployment Guide)

Gbootcms 設計為單一極致輕量的二進制發行版，免除傳統 PHP 複雜的 PHP-FPM、擴展擴展庫安裝與繁重配置。

---

## 1. 跨平台一鍵打包 (`build-linux.bat`)

專案根目錄下的 `build-linux.bat` 會自動完成編譯、UPX 壓縮、資源精簡與目錄打包：

```powershell
.\build-linux.bat
```

執行後生成：
- **發行產物**：`dist/gbootcms-linux.tar.gz`
- **壓縮後總體積**：約 **12 MB**（包含二進制、Layui 靜態資源、基礎資料庫模板）
- **編譯標誌**：`-trimpath -ldflags="-s -w -buildid="`（剝離調試符號與構建路徑，安全且精簡）

---

## 2. 1Panel API 一鍵熱部署 (`deploy-1panel.ps1`)

透過 1Panel 面板的 REST API 快速通道，實現本機直接向線上伺服器推送部署包並熱重啟：

```powershell
# 1. 完整發行包上傳、遠端解壓、權限校準並重啟容器
pwsh -File tools/deploy-1panel.ps1 -Restart

# 2. 單檔案熱更新模式（僅替換單個 Go 模板或視圖檔案，無需重啟）
pwsh -File tools/deploy-1panel.ps1 -File "apps/admin/view/content/media.html"
```

---

## 3. Docker 容器化部署

### Dockerfile 範例

```dockerfile
# 構建階段
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o gbootcms .

# 運行階段（極小 Alpine 鏡像）
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Taipei
WORKDIR /app
COPY --from=builder /app/gbootcms .
COPY --from=builder /app/config ./config
COPY --from=builder /app/template ./template
COPY --from=builder /app/static ./static
COPY --from=builder /app/apps/admin/view ./apps/admin/view

EXPOSE 8080
VOLUME ["/app/data", "/app/static/upload"]
CMD ["./gbootcms"]
```

---

## 4. 守護進程 (Systemd) 範例

若直接在 Linux 主機（如 Ubuntu / Debian）以 systemd 管理：

```ini
[Unit]
Description=Gbootcms Service
After=network.target

[Service]
Type=simple
User=www
WorkingDirectory=/opt/gbootcms
ExecStart=/opt/gbootcms/bin/gbootcms
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```
