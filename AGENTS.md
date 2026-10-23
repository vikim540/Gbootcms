# AGENTS.md

> 本文件供 AI 代理（Claude / GPT / Cursor / Trae / Antigravity 等）在接手 Gbootcms 項目時快速建立上下文。
> 專案嚴格遵循**奧卡姆剃刀原則（如無必要，勿增實體）**，杜絕無意義的分層封裝與重複代碼。

---

## 項目簡介

Gbootcms 是基於 PbootCMS 3.2.12 的現代化 Go 語言移植版。保留原版資料庫結構（`ay_` 前綴）、標籤語法、URL 路由規則與後台 Layui UI，用純 Go 高並發技術棧替換傳統 PHP 後端。

- **模組名**：`gbootcms`（go.mod）
- **Go 版本**：1.25.0
- **資料庫**：SQLite（`github.com/glebarez/sqlite` 純 Go 驅動，無需 CGO），預設路徑 `data/0f13e78cfb83ebff93064bc3b9b7157d.db`
- **前台首頁**：http://localhost:8080/
- **後台管理**：http://localhost:8080/admin
- **預設帳號**：`admin` / `123456`

### 編譯與運行

```powershell
# 本地編譯（產物輸出至 bin/）
go build -trimpath -ldflags="-s -w" -o bin/gbootcms.exe .

# 運行
.\bin\gbootcms.exe

# 或使用一鍵腳本
.\build-run.bat

# Linux 跨平台極限壓縮打包（產物 < 13MB，自動 UPX 壓縮）
.\build-linux.bat
```

---

## 核心技術棧

| 層次 | 選型 | 說明 |
|------|------|------|
| **語言** | Go 1.25 | 單一二進制原生部署，`CGO_ENABLED=0` 零外部依賴 |
| **Web 核心** | Gin v1.12 | 路由、中介軟體管道、RESTful API |
| **ORM** | GORM v1.31 | 單一 `core/db/db.go` 入口，嚴格遵循 `ay_` 原版表結構 |
| **資料庫** | SQLite (glebarez 純 Go) | WAL 高並發模式（20 讀並發 + 單寫無阻塞），免裝 GCC |
| **圖片壓縮** | Pure-Go WebP (`gowebp`) | 伺服器端原生圖片 WebP 轉碼與雙三次平滑縮放，無外部 C 庫 |
| **後台模板** | Pongo2 v6.1 | Django 風格語義 AST 視圖渲染器 + 語法預處理器 |
| **前台模板** | 自研 TagParser | `{gboot:xxx}` 標籤體系 + fsnotify 熱重載 |
| **多語言** | 單模板 + i18n 字典 | 單套 HTML 模板 + `{i18n:...}` 動態替換，杜絕 72 個模板克隆 |
| **遙測監控** | Unified Telemetry Engine | `core/version/telemetry.go` 單一資料源，保證首頁與監控頁 100% 同步 |
| **後台樣式** | Layui + UnoCSS | UnoCSS 統一編譯輸出，收斂多層樣式疊加 |
| **配置** | Viper v1.21 | 環境變數前綴 `PBOOTCMS_GO_` |
| **日誌** | log/slog (stdlib) | 結構化日誌 |

---

## 系統目錄架構

```
pbootcms-go/
├── main.go                          # 程序入口、中斷監聽與優雅關閉
├── config/
│   ├── config.go                    # 配置結構體與 Viper 單例載入
│   └── config.json                  # 配置文件（端口、DB 路徑等）
├── core/
│   ├── db/db.go                     # 全站唯一 DB 初始化與 WAL 連線池配置
│   ├── basic/view.go                # 後台模板引擎（pongo2 + PbootCMS 語法轉換）
│   ├── version/telemetry.go         # 系統版本資訊與統一遙測引擎（進程/記憶體/協程）
│   ├── acodeplugin/                 # 多語言區域隔離 GORM 插件（自動注入 acode 條件）
│   └── mediaplugin/                 # GORM 媒體快取失效與髒標記插件
├── apps/
│   ├── route/route.go               # 集中路由註冊與小寫別名映射
│   ├── api/                         # RESTful API（JWT + API Key 雙認證，/api/v1/ 前綴）
│   ├── common/
│   │   ├── BaseController.go        # 基礎控制器（JSON 回應、分頁、批量排序）
│   │   ├── Render.go                # 後台模板渲染入口
│   │   ├── session.go               # 記憶體安全 Session
│   │   ├── notice.go                # 繁體中文通知訊息常量
│   │   ├── captcha.go               # 驗證碼生成與校驗
│   │   ├── turnstile.go             # Cloudflare Turnstile 人機校驗
│   │   ├── attribution.go           # 營銷歸因（已排除靜態與系統爬蟲路徑）
│   │   ├── imgcompress/             # 伺服器端純 Go 圖片 WebP 壓縮與縮放管道
│   │   ├── middleware/              # 認證、Gzip、快取、IP過濾等中介軟體
│   │   └── parser/                  # 前台標籤解析器、JSON-LD、動態 Sitemap、Robots
│   ├── admin/
│   │   ├── controller/              # 後台各模組控制器
│   │   │   ├── content/             # 內容管理（Content/Sort/Single/Slide/Media/Link 等）
│   │   │   ├── system/              # 系統管理（Config/Menu/Role/User/Syslog/Area 等）
│   │   │   └── member/              # 會員管理（Member/Group/Field/Comment）
│   │   ├── model/
│   │   │   ├── model.go             # 門面：轉發 core/db.DB 並導出各子模組類型別名
│   │   │   ├── migrate.go           # 唯一收斂的資料庫 AutoMigrate 初始化入口
│   │   │   ├── content/             # 內容資料模型
│   │   │   ├── system/              # 系統資料模型
│   │   └── member/                  # 會員資料模型
│   │   ├── helper/                  # 分頁生成、路徑解析輔助函數
│   │   ├── seed/                    # 菜單初始種子數據
│   │   └── view/                    # 後台 HTML 模板（Layui + UnoCSS）
│   └── home/
│       └── controller/              # 前台首頁、列表、詳情、搜索、留言、會員控制器
├── template/                        # 前台模板目錄（default/、en/、sc/，標準 {gboot:xxx} 標籤）
├── static/                          # 全域靜態資源與上傳目錄 (static/upload/)
├── data/                            # SQLite 資料庫檔案
├── bin/                             # 編譯產物輸出目錄
└── docs/                            # 專案技術文檔
```

---

## 核心設計原則與奧卡姆剃刀

1. **唯一 DB 連線（禁止多層封裝）**：
   - 資料庫連線和 WAL 參數嚴格僅在 `core/db/db.go` 初始化。
   - `apps/admin/model/model.go` 僅作為類型別名導出與門面，嚴禁在業務層再次初始化 DB 或保留重複的連接池。
2. **單一模板多語言（禁止物理複製）**：
   - 前台僅保留 `template/default/`，杜絕任何 `template/en/`、`template/sc/` 等物理克隆。
   - 頁面靜態 UI 文案透過 `{i18n:KeyName}` 標籤在渲染時自動載入 `template/default/i18n/{acode}.json`。
3. **伺服器端原生圖片 WebP 壓縮**：
   - 禁止在前端使用 Canvas、JS 偽壓縮，所有圖片上傳經由 `apps/common/imgcompress/` 在伺服器端純 Go 轉碼 WebP 並自適應縮放至 `image_max_width`。
4. **統一遙測資料源**：
   - 後台首頁（`IndexController.Home`）與伺服器資訊頁（`SiteController.Server`）之進程運行時間、記憶體佔用、Goroutines 數，必須統一調用 `core/version/telemetry.go` 的 `GetTelemetry()`，嚴禁各自重複編寫計算邏輯。
5. **SEO & GEO 動態環境感知與精準匹配**：
   - 爬蟲訪問測試域名時強制輸出 `Disallow: /` 與 `X-Robots-Tag: noindex`，生產域名輸出標準 Sitemap 與 `Allow: /`。
   - 跨語言 `hreflang` 僅基於 `rel_id` 或完全相同的 `urlname`/`filename` 精確匹配，嚴禁按欄目文章位次猜測。
   - JSON-LD 的 `mainEntityOfPage.@id` 必須與 `<link rel="canonical">` 絕對一致。

---

## 硬約束（不可違反）

1. **資料庫零破壞** — 不刪改原版 PbootCMS 表結構和既有欄位，表前綴必須為 `ay_`。
2. **自定義欄位用 `mcode`** — 查詢和插入自定義欄位時用 `mcode` 而非 `modelcode`。
3. **表單 action URL 用小寫** — 如 `/admin/content/mod`，防止 POST 重定向遺失請求體。
4. **通知消息用常量** — 使用 `common.NoticeAdd` / `common.NoticeModify` / `common.NoticeDelete`，禁止硬編碼字串，文案不加感嘆號。
5. **後台狀態切換加 `class="switch"`** — `comm.js` 依賴此樣式類別攔截 AJAX 請求，防止瀏覽器跳轉顯示 JSON。
6. **所有 DB 寫入必須檢查 `.Error`** — 禁止忽略 `Create`/`Update`/`Delete` 的錯誤回傳。
7. **所有含 acode 欄位的表操作必須使用 `WithContext`** — 使 GORM `AcodePlugin` 自動注入語言過濾，杜絕跨語言資料外洩。
8. **前台 include 路徑用 `common/`** — `{include file='common/head.html'}`。
9. **正則表達式全局預編譯** — 前台 `TagParser` 正則使用 `sync.Once` 全局預編譯，嚴禁請求級動態編譯。
10. **HTML 頁面快取安全** — 記憶體快取層受 GORM 鉤子聯動，任何後台寫入操作自動呼叫 `ClearHTMLCache()`。
11. **後端圖片 WebP 壓縮零 CGO** — 圖片處理由 `apps/common/imgcompress` 完成，保持 `CGO_ENABLED=0`，支援自動縮放與品質壓縮。
12. **繁體中文規範** — 所有新增的代碼註解、後台 UI、通知訊息與文檔，統一使用繁體中文。

---

## 前台標籤語法速查（TagParser）

| 標籤 | 用途 | 範例 |
|------|------|------|
| `{gboot:site:xxx}` | 站點資訊 | `{gboot:site:title}`、`{gboot:site:logo}` |
| `{gboot:company:xxx}` | 公司資訊 | `{gboot:company:name}`、`{gboot:company:address}` |
| `{gboot:nav}` | 導航選單 | `{gboot:nav parent=0}[nav:name]{/gboot:nav}` |
| `{gboot:list}` | 文章列表 | `{gboot:list scode=2 num=10}[list:title]{/gboot:list}` |
| `{gboot:content:xxx}` | 內容詳情 | `{gboot:content:title}`、`{gboot:content:content}` |
| `{gboot:sort:xxx}` | 欄目資訊 | `{gboot:sort:name}`、`{gboot:sort:link}` |
| `{i18n:Key}` | 多語言字典 | `{i18n:Search}`、`{i18n:Submit}`、`{i18n:Home}` |
| `{gboot:schema}` | JSON-LD | 輸出標準 Schema.org 結構化資料 |
| `{gboot:hreflang}` | 多語言對齊 | 輸出精確匹配的 `<link rel="alternate" hreflang="...">` |

---

## 常用開發命令

```powershell
# 1. 執行單元測試
go test ./...

# 2. 啟動本機開發
go run main.go

# 3. 編譯 Linux amd64 發行包
.\build-linux.bat

# 4. 一鍵部署至 1Panel / 線上環境
pwsh -File tools/deploy-1panel.ps1 -Restart
```

---

## Git 規範

- 倉庫：`https://github.com/vikim540/Gbootcms.git`
- 主分支：`main`
- 規範：Conventional Commits（`feat:` / `fix:` / `refactor:` / `docs:` / `chore:`）
