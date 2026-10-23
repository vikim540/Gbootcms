# Gbootcms 全景開發與維運手冊 (Guide)

> 本手冊為 Gbootcms 的核心使用指南。Gbootcms 是現代化 Go 語言原生內容管理系統，專為高並發、極致輕量與生產級穩定性而設計。
> 專案嚴格遵循**奧卡姆剃刀原則（如無必要，勿增實體）**，杜絕無意義的分層封裝與重複代碼。

---

## 目錄導覽

1. [快速入門 (Quick Start)](#1-快速入門-quick-start)
2. [系統核心架構 (Architecture)](#2-系統核心架構-architecture)
3. [伺服器端原生圖片 WebP 壓縮 (Image Compression)](#3-伺服器端原生圖片-webp-壓縮-image-compression)
4. [前台模板標籤語法 (TagParser)](#4-前台模板標籤語法-tagparser)
5. [多語言與路徑劃分架構 (Multilingual)](#5-多語言與路徑劃分架構-multilingual)
6. [SEO & GEO 搜尋引擎配套 (SEO & GEO)](#6-seo--geo-搜尋引擎配套-seo--geo)
7. [生產部署與 1Panel 自動化 (Deployment)](#7-生產部署與-1panel-自動化-deployment)
8. [開發規範與避坑清單 (Best Practices)](#8-開發規範與避坑清單-best-practices)

---

## 1. 快速入門 (Quick Start)

### 1.1 環境準備
- **編譯工具**：Go 1.25+
- **驅動特性**：純 Go SQLite 驅動，`CGO_ENABLED=0`，無需任何 GCC/MinGW 編譯環境。

### 1.2 本地編譯與啟動
```powershell
# 編譯二進制產物至 bin/ 目錄（剝離符號資訊）
go build -trimpath -ldflags="-s -w" -o bin/gbootcms.exe .

# 啟動服務
.\bin\gbootcms.exe

# 或使用一鍵自動化腳本
.\build-run.bat
```

### 1.3 預設後台位址與憑據
- **後台入口**：`http://localhost:8080/admin`
- **預設帳號**：`admin`
- **預設密碼**：`123456`（首次登入後請立即前往「用戶管理」修改）

---

## 2. 系統核心架構 (Architecture)

```
[客戶端請求] ──► [Gin 高並發路由器] ──┬──► [前台 TagParser 標籤解析 (路徑前綴 + 語言模板)]
                                       └──► [後台 Pongo2 渲染器 (Layui + UnoCSS 統一樣式)]
                                                           │
                                                           ▼
                                            [core/db 唯一資料庫入口]
                                            (SQLite WAL 讀寫分離連線池: 20讀/1寫)
```

### 2.1 唯一資料庫連線池 (`core/db/db.go`)
全站嚴格維護單一 DB 實例，禁止在業務模組層重複建立連接：
```go
// core/db/db.go 核心配置
db.Exec("PRAGMA journal_mode = WAL;")     // WAL 模式：高並發讀與單寫互不排隊
db.Exec("PRAGMA synchronous = NORMAL;")  // 平衡效能與資料安全
db.Exec("PRAGMA busy_timeout = 5000;")    // 寫入超時重試 5 秒
db.Exec("PRAGMA cache_size = -64000;")   // 64MB 頁面快取
```

### 2.2 唯一遷移入口 (`apps/admin/model/migrate.go`)
收斂所有模型 AutoMigrate 為單一執行入口：
```go
func MigrateAll() error {
    if err := system.AutoMigrate(DB); err != nil { return err }
    if err := content.AutoMigrate(DB); err != nil { return err }
    if err := member.AutoMigrate(DB); err != nil { return err }
    return nil
}
```

---

## 3. 伺服器端原生圖片 WebP 壓縮 (Image Compression)

徹底淘汰傳統前端 Canvas/JS 偽壓縮，實作純 Go 伺服器端原生處理管道（`apps/common/imgcompress`）：
- **全入口覆蓋**：後台單圖/多圖、UEditor 富文本貼圖、RESTful API 上傳均自動觸發。
- **純 Go 零 CGO**：基於 `github.com/KarpelesLab/gowebp` 與 `golang.org/x/image/draw.BiCubic`。
- **極致壓縮**：實測 890KB PNG 轉為 WebP 僅 130KB（體積縮減 85%），大幅降低 CDN 與伺服器頻寬開銷。

```go
// 控制器內調用範例
newPath, err := imgcompress.CompressToWebP(savePath, imgcompress.Options{
    Quality:  80,   // 壓縮品質 1-100
    MaxWidth: 1920, // 超過最大寬度自動等比雙三次平滑縮放
})
```

---

## 4. 前台模板標籤語法 (TagParser)

前台視圖模板（`template/default/`）採用標籤解析器，語法直觀易懂：

### 4.1 導航標籤 (`{gboot:nav}`)
```html
<ul class="navbar">
    <li><a href="/">首頁</a></li>
    {gboot:nav parent=0}
    <li class="{if('[nav:scode]'=='{gboot:sort:tcode}')}active{/if}">
        <a href="[nav:link]">[nav:name]</a>
    </li>
    {/gboot:nav}
</ul>
```

### 4.2 內容列表標籤 (`{gboot:list}`)
```html
{gboot:list scode=2 num=10 order='date desc'}
<div class="article-card">
    <a href="[list:link]">
        <img src="[list:ico]" alt="[list:title]">
        <h3>[list:title]</h3>
        <p>[list:description]</p>
        <span class="time">[list:date style='Y-m-d']</span>
    </a>
</div>
{/gboot:list}

<div class="page-bar">
    {gboot:pagebar}
</div>
```

---

## 5. 多語言與路徑劃分架構 (Multilingual)

### 5.1 路由路徑前綴劃分
前台依據 URL 前綴自動匹配與切換當前語言上下文（`acode`）：
- `/`：默認繁體中文站點 (`tc`)
- `/en/`：英文站點 (`en`)
- `/sc/`：簡體中文站點 (`sc`)

由 `acodeplugin` 自動解析路徑前綴並注入 `Request Context`，GORM 查詢自動完成多語言數據過濾。

### 5.2 語言專屬模板目錄
各語言使用專屬模板目錄（`template/default`、`template/en`、`template/sc`），直接套用標準 `{gboot:xxx}`、`{content:xxx}` 等標籤與原生文案，保持語句自然流暢，杜絕中間層自造標籤。
公共靜態資源統一由 `template/default/static/` 維護並由後端自動映射。

---

## 6. SEO & GEO 搜尋引擎配套 (SEO & GEO)

### 6.1 動態 Host 感知 Robots.txt
- **測試/預發環境**：系統自動感知請求網域，強制輸出 `Disallow: /` 並附加 `X-Robots-Tag: noindex, nofollow` HTTP 標頭，杜絕測試站被爬蟲收錄導致主站權重受損。
- **生產環境**：自動放行並動態綁定當前網域之 `Sitemap: https://{Host}/sitemap.xml`。

### 6.2 結構化資料 JSON-LD (`{gboot:schema}`)
輸出標準 Schema.org 規範之 `WebSite` 與 `Article` 結構化資料，且保證 `mainEntityOfPage.@id` 嚴格等於當前頁面之 `<link rel="canonical">`。

### 6.3 雙向跨語言關聯 (`{gboot:hreflang}`)
基於資料庫明確的 `rel_id` 欄位進行雙向精確映射，嚴禁按文章發布位次瞎猜，無精確關聯時不輸出該語言標籤。

### 6.4 AI 大模型檢索端點 (`/llms.txt`)
標準提供 `/llms.txt` 與 `/llms-full.txt` 端點，供現代 AI 檢索代理（SearchGPT、Claude、Perplexity）以極低消耗完成全站知識提取。

---

## 7. 生產部署與 1Panel 自動化 (Deployment)

### 7.1 Linux 一鍵極限壓縮打包 (`build-linux.bat`)
```powershell
.\build-linux.bat
# 輸出：dist/gbootcms-linux.tar.gz
# 總體積約 12MB（含 UPX 壓縮二進制、Layui 靜態資源、基礎資料庫模板）
```

### 7.2 1Panel API 一鍵熱部署 (`deploy-1panel.ps1`)
```powershell
# 一鍵上傳部署包、伺服器端解壓、資料庫/運行環境遞迴賦權 (0777) 並熱重啟容器：
pwsh -File tools/deploy-1panel.ps1 -Restart

# 單視圖/模板檔案熱更新（無需重啟）：
pwsh -File tools/deploy-1panel.ps1 -File "apps/admin/view/content/media.html"
```

---

## 8. 開發規範與避坑清單 (Best Practices)

1. **資料庫零破壞**：嚴格保留原版 `ay_` 前綴表結構與既有欄位。
2. **所有 DB 寫入必須檢查 `.Error`**：禁止忽略 `Create`/`Update`/`Delete` 傳回之錯誤。
3. **多語言模型查詢使用 `WithContext`**：使 GORM `AcodePlugin` 自動注入當前語言分區過濾條件。
4. **狀態開關標籤加 `class="switch"`**：供 `comm.js` 攔截 AJAX 請求，防止瀏覽器跳轉顯示原始 JSON。
5. **通知文案統一常量**：使用 `common.NoticeAdd`、`common.NoticeModify`、`common.NoticeDelete`，不帶感嘆號。
