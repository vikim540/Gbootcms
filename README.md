# Gbootcms

> 基於 PbootCMS 3.2.12 的現代化 Go 語言原生移植版 — 保留原版數據庫結構（`ay_` 前綴）、相容模板語法與 URL 路由，採用純 Go 高並發技術棧全面取代傳統 PHP 後端。

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-v1.12-00ADD8)](https://gin-gonic.com/)
[![GORM](https://img.shields.io/badge/GORM-v1.31-00ADD8)](https://gorm.io/)
[![SQLite](https://img.shields.io/badge/SQLite-pureGo_WAL-003B57)](https://github.com/glebarez/sqlite)
[![UPX](https://img.shields.io/badge/Binary-<13MB-green)](https://upx.github.io/)

---

## 快速開始

### 本地編譯與運行

```powershell
# 編譯二進制（輸出至 bin/）
go build -trimpath -ldflags="-s -w" -o bin/gbootcms.exe .

# 啟動服務
.\bin\gbootcms.exe

# 或使用一鍵腳本
.\build-run.bat
```

- **前台展示**：http://localhost:8080/
- **後台管理**：http://localhost:8080/admin
- **預設帳號**：`admin` / `123456`（首次登入後請立即修改）

### Linux 跨平台極簡打包

```powershell
# 一鍵打包 Linux amd64 極限壓縮發行版（自動 UPX 壓縮，體積 < 13MB）
.\build-linux.bat
# 產物路徑：dist/gbootcms-linux.tar.gz
```

---

## 核心技術棧

| 層次 | 選型 | 說明 |
| :--- | :--- | :--- |
| **程式語言** | Go 1.25 | 單一二進制原生部署，`CGO_ENABLED=0` 零依賴 |
| **Web 核心** | Gin v1.12 | 現代化高並發 HTTP 路由器與中介軟體管道 |
| **ORM 映射** | GORM v1.31 | 單一 `core/db/db.go` 入口，嚴格遵循 `ay_` 原版表結構 |
| **資料庫引擎** | Pure-Go SQLite 3 | WAL 高並發讀寫日誌模式，免裝 GCC / CGO |
| **圖片壓縮管道** | Pure-Go WebP (`gowebp`) | 伺服器端原生自動圖片轉碼與縮放，無外部 C 庫 |
| **後台模板引擎** | Pongo2 v6.1 | Django 風格語義 AST 視圖渲染器 + 語法預處理器 |
| **前台解析引擎** | 自研 TagParser | `{gboot:xxx}` 標籤體系 + fsnotify 模板熱重載 |
| **多語言字典** | i18n Dictionary Provider | 單套 HTML 模板字典化渲染，杜絕物理多語系檔案克隆 |
| **遙測監控** | Unified Telemetry Engine | `core/version/telemetry.go` 單一資料源，保證首頁與監控頁 100% 一致 |

---

## 最新架構特性

### 1. 伺服器端原生圖片壓縮與 WebP 管道
- **完全取代前端偽 Canvas 壓縮**：後端 `IndexController.Upload()` 內建 `imgcompress` 處理管道。
- **全入口覆蓋**：管理後台單圖/多圖、UEditor 富文本貼圖、RESTful API 上傳均自動觸發。
- **純 Go 零 CGO**：採用 `github.com/KarpelesLab/gowebp` 與 `golang.org/x/image/draw` 進行高品質平滑縮放與 WebP 轉碼，檔案體積減幅達 60%~85%，無縫支援透明通道與 EXIF。

### 2. SEO & GEO 搜尋引擎基礎配套
- **動態環境感知 Robots.txt**：自動根據 `Host` 判斷測試域名（強制輸出 `Disallow: /` 及 `X-Robots-Tag: noindex`）與生產域名（輸出 `Allow: /` 及 Sitemap 索引）。
- **XML Sitemap 多語系覆蓋**：`/sitemap.xml` 總索引自動分發每語言獨立 Sitemap（`/sitemap-{acode}.xml`）。
- **LLMs.txt 大模型標準索引**：標準 `/llms.txt` 端點支援 AI Agent 快速精準檢索。
- **JSON-LD 結構化資料**：自動輸出標準 Schema.org 規範之 `Article`、`WebSite` 標註，消除重複內容。
- **精準雙向 hreflang**：基於跨語言關聯 ID（`rel_id`）精確映射多語系對應文章，嚴禁無依據之位次猜測。

### 3. 多語言單一模板架構（奧卡姆剃刀）
- 徹底刪除冗餘的 72 個多語系物理 HTML 模板（`template/en/`、`template/sc/`）。
- 全站收斂為單一 `template/default/`，透過動態 `{i18n:...}` 多語言字典於運行時完成國際化替換，維護成本降低 66%。

### 4. 媒體庫健全度與防禦
- **路徑雙向對齊**：徹底解決數據庫路徑（`/upload/...`）與磁盤物理路徑（`static/upload/...`）不一致導致的誤判與清理誤刪。
- **系統檔案過濾**：自動忽略 `.gitkeep`、`.DS_Store` 等隱藏維護檔案。
- **WebP 尺寸解析**：內建 WebP 解碼驅動，正確讀取現代格式尺寸資訊。

---

## 目錄結構

```
pbootcms-go/
├── main.go                          # 服務入口、優雅關閉與路由分發
├── config/                          # 系統配置（Viper，支援環境變數覆蓋）
├── core/                            # 核心底層引擎
│   ├── db/                          # 唯一資料庫連線與 WAL 設定
│   ├── basic/                       # 後台 Pongo2 視圖渲染器與標籤預處理
│   ├── version/                     # 系統版本資訊與統一遙測引擎 (telemetry.go)
│   ├── acodeplugin/                 # 多語言區域自動隔離 GORM 插件
│   └── mediaplugin/                 # 媒體快取與髒標記 GORM 插件
├── apps/
│   ├── route/                       # HTTP 路由表註冊與別名映射
│   ├── api/                         # RESTful API 端點（JWT + API Key 雙認證）
│   ├── common/                      # 公共組件
│   │   ├── imgcompress/             # 伺服器端純 Go 圖片 WebP 壓縮轉碼引擎
│   │   ├── parser/                  # 前台標籤解析、JSON-LD、Sitemap、i18n
│   │   ├── middleware/              # 認證、SEO隔離、快取、安全頭中介軟體
│   │   ├── session.go               # 記憶體狀態管理
│   │   └── captcha.go               # 點陣運算驗證碼
│   ├── admin/                       # 後台管理模組 (Controller/Model/View/Service)
│   └── home/                        # 前台展示模組 (FrontController)
├── template/default/                # 前台唯一模板目錄（字典化支援多語言）
├── static/                          # 靜態資源（後台樣式、上傳素材）
├── data/                            # SQLite 資料庫檔案
├── bin/                             # 編譯產物
└── dist/                            # Linux 發行壓縮包
```

---

## 開發約束

1. **資料庫零改動**：不得隨意更改或刪除原版 `ay_` 資料表結構。
2. **奧卡姆剃刀**：如無必要，勿增實體。禁止為單一功能疊床架屋，優先收斂於既有核心模組。
3. **無 CGO 依賴**：全專案嚴格保持 `CGO_ENABLED=0`，確保交叉編譯與單一二進制移植性。
4. **標準繁體中文**：所有註釋、代碼文檔與介面文字遵循簡練專業規範，拒絕虛偽敘事與廢話文學。

---

## License

基於 PbootCMS 3.2.12 開源協議移植，保留原作者版權聲明。
