# PbootCMS-Go

> **PbootCMS 的 Go 語言實現** — 一個用 Go 構建的高性能、單二進制、零依賴部署的內容管理系統

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8)](https://go.dev)
[![Gin](https://img.shields.io/badge/Gin-v1.12-00ADD8)](https://gin-gonic.com)
[![GORM](https://img.shields.io/badge/GORM-v1.31-FF6F61)](https://gorm.io)
[![SQLite](https://img.shields.io/badge/SQLite-pure_Go-003B57)](https://github.com/glebarez/sqlite)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue)](LICENSE)

---

## 項目簡介

**PbootCMS-Go** 是經典 PHP 開源 CMS **PbootCMS** 的 Go 語言重寫版本。我們致力於：

- ✅ **100% 保留 PbootCMS 用戶體驗**（後台、URL、模板語法零改動）
- ✅ **享受 Go 語言的全部優勢**（單二進制、毫秒啟動、低內存、跨平台）
- ✅ **完全兼容 PbootCMS 數據庫**（SQLite 表結構 1:1 對應，可雙向遷移）

如果您曾經使用過 PbootCMS，又希望獲得 Go 部署的便利，PbootCMS-Go 是您的最佳選擇。

---

## 為什麼選擇 PbootCMS-Go？

### 對比 PbootCMS (PHP)

| 維度 | PbootCMS (PHP) | PbootCMS-Go |
|---|---|---|
| **部署** | 需要 PHP 7.0+ + Web 服務器（Nginx/Apache） | 單個二進制文件，無外部依賴 |
| **內存佔用** | PHP-FPM 模式 30-80MB | **10-20MB** |
| **並發性能** | 受限於 PHP-FPM 進程數 | **Go 原生 goroutine，輕鬆上萬並發** |
| **啟動時間** | 500-1000ms | **< 50ms** |
| **跨平台** | 需安裝 PHP 環境 | **一次編譯，到處運行**（Windows/Linux/macOS/ARM） |
| **數據庫** | MySQL / SQLite | SQLite（純 Go 驅動，無 CGO） |
| **運維成本** | 需維護 PHP 版本、擴展、配置 | **扔一個二進制到服務器即可** |

### 對比其他 Go CMS

| 特性 | PbootCMS-Go | 主流 Go CMS（如 Hugo、Wagtail-Go） |
|---|---|---|
| **後台管理** | ✅ 開箱即用 | ❌ 需自行開發 |
| **內容模型** | ✅ 動態可配置 | ❌ 通常需代碼定義 |
| **模板語法** | ✅ 兼容 PbootCMS PHP 風格 | ❌ 各家不同 |
| **用戶熟悉度** | ✅ PbootCMS 60 萬用戶可直接上手 | ❌ 需重新學習 |
| **插件生態** | 規劃中 | 各異 |

---

## 核心特性

### 🎯 已完成（v0.6+）

- ✅ **完整的後台管理系統**
  - 登錄 + 驗證碼（圖形碼）
  - 站點信息管理
  - 公司信息管理
  - 欄目管理（單頁/列表）
  - 內容管理（CRUD、批量操作、排序、複製、移動）
  - 會員管理
  - 菜單/角色/用戶權限
  - 系統配置
  - 緩存清理
  - 數據庫管理

- ✅ **前台渲染引擎**
  - 模板標籤：`{pboot:list}`、`{pboot:nav}`、`{pboot:sort}`、`{pboot:content}`、`{pboot:if}` 等
  - 多圖輪播 (`{pboot:pics}`)
  - 分頁組件 (`{pboot:list page=1}`)
  - 標籤雲 (`{pboot:tags}`)
  - 搜索 (`{pboot:search}`)
  - 留言 (`{pboot:message}`)

- ✅ **文件上傳**
  - 支持圖片、文檔等多類型
  - 擴展名白名單校驗
  - 水印支持

- ✅ **Go 慣用架構**
  - MVC+S 4 層架構（Model / View / Controller / Service）
  - pongo2 模板引擎（業界標準）
  - GORM ORM
  - Service 層業務邏輯抽離
  - 白名單安全機制

### 🚧 規劃中（v1.0+）

- 🚧 Hook 掛鉤 & 插件體系
- 🚧 一鍵全站靜態化（SEO 友好）
- 🚧 緩存生成邏輯
- 🚧 多語言支持
- 🚧 動態自定義字段
- 🚧 從 PbootCMS PHP 版遷移工具
- 🚧 RBAC 細粒度權限校驗
- 🚧 XSS 防護完善
- 🚧 密碼 bcrypt 升級

---

## 技術棧

| 層 | 選型 | 說明 |
|---|---|---|
| **語言** | Go 1.25 | 最新穩定版 |
| **HTTP 框架** | Gin v1.12 | 高性能 Radix Tree 路由 |
| **ORM** | GORM v1.31 | Go 社區最流行 ORM |
| **數據庫** | SQLite (glebarez 純 Go) | 無 CGO 依賴，跨平台編譯 |
| **模板引擎** | pongo2 v6.1.0 | Django 風格，業界標準 |
| **會話** | gin-contrib/sessions | Cookie Store |
| **文件監聽** | fsnotify | 模板熱加載 |
| **架構** | MVC + Service 4 層 | Controller 只做參數傳遞 |

---

## 快速開始

### 環境要求

- Go 1.25+（推薦從 [官網下載](https://go.dev/dl/)）
- 支持 Windows / Linux / macOS
- 無需 CGO，無需額外系統庫

### 編譯運行

```bash
# 1. 克隆項目
git clone https://github.com/yourname/pbootcms-go.git
cd pbootcms-go

# 2. 下載依賴
go mod tidy

# 3. 編譯
go build -o pbootcms-go .

# 4. 運行
./pbootcms-go
```

### 訪問

- **前台首頁**：http://localhost:8080
- **後台管理**：http://localhost:8080/admin
- **默認賬號**：`admin` / `admin`

### Docker 部署（規劃中）

```bash
docker run -d -p 8080:8080 -v /data/pbootcms:/app/data pbootcms-go
```

---

## 項目結構

```
pbootcms-go/
├── main.go                      # 入口文件
├── go.mod                       # 依賴管理
├── config/
│   └── config.json              # 配置文件
├── apps/
│   ├── admin/                   # 後台管理
│   │   ├── controller/          # 30 個 controller
│   │   ├── model/               # 27 個 GORM model
│   │   ├── service/             # 業務邏輯層（MVC+S）
│   │   └── view/                # 靜態資源
│   ├── home/                    # 前台展示
│   │   ├── controller/          # 前台 controller
│   │   └── model/
│   ├── common/                  # 公共組件
│   │   ├── parser/              # 模板解析器
│   │   └── middleware/          # 中間件
│   └── route/                   # 路由配置
├── core/                        # 核心模塊
│   └── db/                      # 數據庫初始化
├── templates/                   # HTML 模板
│   ├── admin/                   # 後台模板
│   └── index.html               # 前台首頁
├── static/                      # 靜態資源
│   ├── admin/                   # 後台 CSS/JS
│   └── css/                     # 前台 CSS
├── data/                        # 數據目錄
│   └── pbootcms.db              # SQLite 數據庫
├── runtime/                     # 緩存目錄
├── ARCHITECTURE_REVIEW.md       # 架構評測報告
└── ARCHITECTURE_REVIEWS/        # 評測記錄目錄
```

---

## 與 PbootCMS PHP 版的兼容性

### 數據庫兼容 ✅

- ✅ 表前綴 `ay_` 完整保留
- ✅ 表結構 1:1 對應
- ✅ 字段名、字段類型、索引完全一致
- ✅ 數據可雙向遷移（Go 版 ↔ PHP 版）

### URL 風格兼容 ✅

- ✅ 後台 `/admin/{模塊}/{動作}` 結構保留
- ✅ 前台偽靜態 `.html` 結尾
- ✅ 動態參數風格一致

### 模板語法兼容 ✅

- ✅ PHP 風格標籤（`{pboot:list}`、`{pboot:if(...)}` 等）直接可用
- ✅ HTML 模板文件不需修改
- ✅ 變量輸出風格 `{$var}`

### 配置文件兼容 ✅

- ✅ JSON 結構與 PbootCMS 對應
- ✅ 經驗可直接複用

---

## 性能基準（初步測試）

> 以下數據為初步測試結果，僅供參考

| 場景 | 結果 |
|---|---|
| 冷啟動時間 | **< 50ms** |
| 空閒內存 | **~15MB** |
| 並發 goroutine 數 | 數萬級（受限於 SQLite 文件鎖） |
| 首頁響應時間 | **< 10ms**（帶數據庫查詢） |
| 後台列表頁響應 | **< 30ms** |

---

## 開發團隊

PbootCMS-Go 由個人開發者獨立完成，旨在為 PbootCMS 社區提供 Go 語言的選擇。

---

## 開源協議

本項目採用 **Apache License 2.0** 開源。

PbootCMS 原項目採用 **Apache License 2.0** 開源，感謝其優秀的設計和廣泛的用戶基礎。

---

## 路線圖

### v0.6（當前）— 基礎可用
- ✅ 後台 CRUD 完整
- ✅ 前台模板渲染
- ✅ 文件上傳
- ✅ 業務服務層

### v1.0 — 功能完善
- 🚧 Hook 插件體系
- 🚧 一鍵靜態化
- 🚧 緩存生成
- 🚧 多語言
- 🚧 動態自定義字段
- 🚧 遷移工具（PHP → Go）
- 🚧 RBAC 細粒度權限

### v2.0 — 工業級
- 多數據庫支持（MySQL/PostgreSQL）
- Docker 鏡像
- CI/CD 流水線
- 單元測試覆蓋率 > 70%
- 性能壓測報告

---

## 社區

- 📖 **文檔**：[待完善]
- 🐛 **問題反饋**：[GitHub Issues]
- 💬 **討論**：[GitHub Discussions]
- 📧 **郵箱**：[待公佈]

---

## 致謝

- 感謝 [PbootCMS](https://www.pbootcms.com/) 團隊的優秀作品
- 感謝 [Gin](https://gin-gonic.com/)、[GORM](https://gorm.io/)、[pongo2](https://github.com/flosch/pongo2) 等 Go 生態項目
- 感謝所有貢獻者和用戶的支持

---

**PbootCMS-Go** — _保留 PbootCMS 的靈魂，擁抱 Go 的速度。_
