# PbootCMS-Go

> PbootCMS 3.2.12 的 Go 語言移植版 — 保留原版數據庫結構、模板語法、URL 路由，用 Go 技術棧替換 PHP 後端。

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-v1.12-00ADD8)](https://gin-gonic.com/)
[![GORM](https://img.shields.io/badge/GORM-v1.31-00ADD8)](https://gorm.io/)
[![SQLite](https://img.shields.io/badge/SQLite-pureGo-003B57)](https://github.com/glebarez/sqlite)

## 快速開始

```powershell
# 編譯
go build -o bin/pbootcms-go.exe .

# 運行
.\bin\pbootcms-go.exe

# 或用構建腳本
.\build.ps1 -Run
```

- 前台：http://localhost:8080/
- 後台：http://localhost:8080/admin
- 預設帳號：`admin` / `admin`

## 技術棧

| 層次 | 選型 | 說明 |
|------|------|------|
| 語言 | Go 1.25 | 單二進制部署，無需 CGO |
| Web 框架 | Gin v1.12 | 路由、中間件、請求處理 |
| ORM | GORM v1.31 | AutoMigrate，`ay_` 前綴 |
| 數據庫 | SQLite (glebarez 純 Go) | 無需安裝 CGO/GCC |
| 後台模板 | Pongo2 v6.1 | Django 風格 + PbootCMS 語法轉換器 |
| 前台模板 | 自研 TagParser | `{gboot:xxx}` 標籤 + fsnotify 熱重載 |
| 後台 UI | Layui 2.5.4 + jQuery | 與 PbootCMS 原版一致 |
| 前台 UI | Bootstrap 4 + Swiper 4 | 前台模板自帶 |

## 核心特性

### 與 PbootCMS PHP 版的兼容性

- **數據庫零改動**：表結構、字段名、`ay_` 前綴完全一致，可直接遷移數據
- **URL 100% 兼容**：路由重寫 + NoRoute 兜底，支援原版 URL 格式
- **模板語法兼容**：後台保留 PbootCMS PHP 模板語法，前台使用 `{gboot:xxx}` 標籤
- **密碼雙 MD5**：`md5(md5(password))` 與原版用戶數據兼容

### 雙模板引擎架構

| 場景 | 引擎 | 語法 |
|------|------|------|
| 後台 admin view | pongo2 + 語法轉換器 | `{if([$list])}` → `{% if List %}` |
| 前台 template/default | 自研 TagParser | `{gboot:xxx}` + `[prefix:field]` |

## 已實現功能

### 後台管理

- 管理員登入/登出（密碼雙 MD5 + 驗證碼 + 登入鎖定）
- 內容管理：文章/產品增刪改查、批量排序、擴展字段
- 欄目管理：樹形結構、自定義 URL、模板選擇
- 單頁管理、站點信息、公司信息
- 幻燈片、友情連結、自定義標籤、內鏈標籤
- 內容模型、擴展字段、自定義表單
- 媒體庫（圖片管理）、留言管理
- 系統配置（70+ 配置項）、菜單管理、角色權限
- 用戶管理、數據庫備份、系統日誌、區域管理
- **會員管理**：會員列表/新增/修改/刪除（含批量操作）
- **會員等級**：等級列表/新增/修改/刪除（含狀態切換、刪除保護）
- **會員欄位**：欄位列表/新增/修改/刪除（含必填切換、狀態切換）
- **文章評論**：評論列表/詳情/回覆/刪除（含批量審核、Excel 匯出）

### 前台展示

- 首頁（輪播圖、導航、產品展示）
- 列表頁（分頁、ext_ 篩選、子分類導航）
- 內容詳情頁（擴展字段、麵包屑、上一篇/下一篇）
- 搜索頁、標籤頁、留言頁、單頁
- 模板熱重載（fsnotify）、.html 偽靜態 URL
- 訪問量統算、標題顏色渲染
- **會員系統**：登入/註冊/登出/個人中心/資料修改
  - 三合一登入（用戶名/郵箱/手機）
  - 驗證碼條件顯示（配置驅動）
  - 10 秒防刷註冊
  - 雙 MD5 密碼兼容
  - AJAX 表單提交

## 目錄結構

```
pbootcms-go/
├── main.go                    # 程序入口
├── config/                    # 配置文件
├── core/                      # 核心引擎（DB、模板、媒體插件）
├── apps/
│   ├── route/                 # 路由註冊
│   ├── common/                # 公共組件（Session、BaseController、Parser）
│   ├── admin/                 # 後台（Controller/Model/View/Service）
│   └── home/                  # 前台（FrontController + MemberController）
├── template/default/          # 前台模板
│   ├── comm/                  # 公共模板
│   ├── member/                # 會員前台模板
│   └── static/                # 前台靜態資源
├── static/                    # 全域靜態資源（admin/upload/images）
├── data/pbootcms.db           # SQLite 數據庫
├── bin/                       # 編譯產物
└── docs/                      # 文檔
```

## 開發文檔

| 文檔 | 說明 |
|------|------|
| [開發技術文檔](docs/0701pbootcms-go-dev-guide.md) | 完整技術文檔（含防遺忘清單、API 速查、反模式） |
| [AI 開發指南](docs/AI_GUIDELINES.md) | AI 輔助開發約定 |
| [架構評測報告](docs/ARCHITECTURE_REVIEW.md) | 架構分析與建議 |

## 開發約束

1. **數據庫零改動** — 不修改、不刪除原版表結構和字段
2. **繁體中文** — 所有修改的代碼和模板使用繁體中文
3. **構建產物** — 輸出到 `bin/` 目錄，不滯留根目錄
4. **文檔** — 所有 `.md` 文件放在 `docs/` 文件夾
5. **通知消息** — 使用 `notice.go` 常量，禁止硬編碼
6. **通知常量** — 使用 `common.NoticeAdd` / `common.NoticeModify` / `common.NoticeDelete`

## 開發環境

- Go 1.25+
- GOPROXY：`https://goproxy.cn,direct`
- 無需 CGO（純 Go SQLite 驅動）
- PowerShell 7（構建腳本）

## 路線圖

- [x] 階段 0：會員模型修復（Member/MemberGroup/MemberField/MemberComment）
- [x] 階段 1：前台會員系統（登入/註冊/登出/個人中心/資料修改）
- [x] 後台會員管理（等級/欄位/會員/評論）
- [ ] 階段 2：會員中心增強（積分系統、等級自動升級）
- [ ] 階段 3：前台評論系統（提交/列表/我的評論/刪除）
- [ ] 階段 4：密碼找回 + 郵件驗證

## License

基於 PbootCMS 3.2.12 移植，遵循原版授權。
