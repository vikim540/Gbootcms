# 快速上手 (Quick Start)

本指南介紹如何在本機環境編譯、運行與除錯 Gbootcms。

---

## 1. 環境需求

- **Go 編譯器**：Go 1.25+
- **CGO 狀態**：`CGO_ENABLED=0`（採用純 Go SQLite 驅動，免裝 GCC / MinGW）
- **作業系統**：Windows / Linux / macOS

---

## 2. 獲取代碼與依賴

```bash
git clone https://github.com/vikim540/Gbootcms.git
cd Gbootcms
go mod download
```

---

## 3. 本地編譯與運行

### Windows (PowerShell)
```powershell
# 編譯精簡二進制（輸出至 bin/）
go build -trimpath -ldflags="-s -w" -o bin/gbootcms.exe .

# 運行程序
.\bin\gbootcms.exe

# 或使用一鍵批次腳本
.\build-run.bat
```

### Linux / macOS (Bash)
```bash
# 編譯二進制
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/gbootcms .

# 賦予執行權限並運行
chmod +x bin/gbootcms
./bin/gbootcms
```

---

## 4. 訪問後台

- **前台展示**：`http://localhost:8080/`
- **後台登入**：`http://localhost:8080/admin`
- **預設管理員**：`admin` / `123456`

---

## 5. 常見管理命令範例

```powershell
# 執行單元測試
go test -v ./...

# 檢查資料庫狀態
sqlite3 data/0f13e78cfb83ebff93064bc3b9b7157d.db "SELECT count(*) FROM ay_content;"

# 清理運行時快取（靜態 HTML 快取）
# 後台任何文章修改均會自動觸發清除，亦可手動重啟服務
```
