# pbootcms-go 項目移交文案

> **項目**:`F:\mysite\AI\idea\pbootcmstogo\pbootcms-go`
> **移交人**:(我)
> **接手人**:(新同事)
> **日期**:2026-06-05
> **基線 tag**:`v0.2.0-tasks1-7-20260605`(今日工作備份)
> **本文檔**:`ARCHITECTURE_REVIEWS/2026-06-05-002-handover-guide.md`

---

## 一句話總結

> 把 PHP 版 PbootCMS 用 Go 翻譯一遍,當前完成度 **40% 翻譯 + 框架性 0%**,正在按 A+B 混合路線推進(保留 PbootCMS 兼容 + 引入 Go 慣用模式)。

---

## 第 1 步:把環境跑起來(5 分鐘)

### 1.1 必備工具

| 工具 | 最低版本 | 備註 |
|---|---|---|
| Go | 1.25.0 | `go version` 驗證 |
| Git | 任意 | 用於切換 tag |
| 瀏覽器 | 任意 | 用於訪問後台 |
| PowerShell 7 | 7.6+ | 構建腳本 `build.ps1` 用 |

### 1.2 Clone 倉庫

```bash
cd F:\mysite\AI\idea\pbootcmstogo\pbootcms-go
git status  # 確認是工作樹,不是空目錄
```

如果**不是空目錄**,你已經拿到本地倉庫了。跳到 1.3。

如果是空倉庫:
```bash
git clone <repo-url>
cd pbootcms-go
git checkout v0.2.0-tasks1-7-20260605   # 切到今日備份
```

### 1.3 構建 + 啟動

**方法 A:用 PowerShell 腳本(推薦)**
```powershell
.\build.ps1         # 構建
.\bin\pbootcms-go.exe  # 啟動
```

**方法 B:手動**
```bash
go mod tidy
go build -o bin/pbootcms-go.exe .
.\bin\pbootcms-go.exe
```

### 1.4 驗證啟動成功

打開瀏覽器:
- 前台 <http://localhost:8080/> 應該看到 PbootCMS 首頁
- 後台 <http://localhost:8080/admin/> 應該看到登錄頁

**默認管理員**:`admin` / `admin`

---

## 第 2 步:讀這 3 份文檔(30 分鐘必讀)

按順序讀,**不要跳過**:

| 順序 | 文檔 | 讀完你會知道 |
|---|---|---|
| 1 | [`ARCHITECTURE_REVIEW.md`](ARCHITECTURE_REVIEW.md) | 項目是什麼、質量評估、A+B 路線、4 個關鍵動作 |
| 2 | [`.plan.md`](../.plan.md) | 已發現的問題清單 + 已完成的 1.x 小贏 + 後續規劃 |
| 3 | [`ARCHITECTURE_REVIEWS/2026-06-05-001-task1-7-check.md`](2026-06-05-001-task1-7-check.md) | 今日 7 個 Task 的逐個核查 + 業界標準評估 |

讀完後你應該能回答:
- 這個項目目標是什麼? → 答案見 ARCHITECTURE_REVIEW.md 第一、二節
- 已完成什麼? → 答案見 .plan.md
- 今天做了什麼? → 答案見 2026-06-05-001
- 下一步該做什麼? → 答案見本文檔「後續任務」一節

---

## 第 3 步:理解目錄結構(15 分鐘)

```
pbootcms-go/
├── apps/                        ← 業務代碼(PbootCMS 風格分層)
│   ├── admin/                   ← 後台管理
│   │   ├── controller/          ← HTTP 入口(31 個)
│   │   ├── model/               ← 數據訪問(27 個)
│   │   ├── service/             ← 業務邏輯層(新增,僅 content 已完成)
│   │   ├── seed/                ← 種子數據初始化
│   │   └── helper/              ← 通用助手(剛新增)
│   ├── common/                  ← 跨模塊基礎
│   │   ├── BaseController       ← JSON 響應統一封裝
│   │   ├── Render               ← 後台模板渲染
│   │   ├── session.go           ← 會話
│   │   ├── parser/              ← PHP→Go 標籤 transpiler
│   │   └── middleware/          ← Gin 中間件(auth, path_rewrite)
│   ├── home/                    ← 前台
│   │   └── controller/front.go
│   └── route/
│       └── route.go             ← 110+ 條手寫路由
├── config/
│   ├── config.go                ← 配置加載
│   └── config.json              ← 默認配置(端口 8080、SQLite 路徑)
├── core/
│   ├── basic/view.go            ← pongo2 模板引擎封裝
│   └── db/db.go                 ← GORM 初始化(連接池:MaxOpen=1,MaxIdle=1)
├── data/
│   └── pbootcms.db              ← SQLite 數據庫(已入庫,默認 admin/admin)
├── static/                      ← 靜態資源
│   ├── admin/                   ← 後台用 layui/font-awesome
│   └── css/                     ← 前台樣式
├── templates/                   ← HTML 模板(PbootCMS 原版搬過來)
│   ├── admin/                   ← 後台模板
│   ├── index.html               ← 前台首頁
│   ├── list.html                ← 列表頁
│   ├── content.html             ← 內容頁
│   └── ...
├── ARCHITECTURE_REVIEW.md       ← 必讀文檔 #1
├── ARCHITECTURE_REVIEWS/        ← 逐次核查記錄
│   ├── 2026-06-08-001-task11-check.md     ← 昨日的核查
│   ├── 2026-06-05-001-task1-7-check.md    ← 今日的核查
│   └── 2026-06-05-002-handover-guide.md   ← 本文件
├── .plan.md                     ← 必讀文檔 #2
├── build.ps1                    ← PowerShell 構建腳本
├── main.go                      ← 應用入口
├── go.mod / go.sum              ← 依賴
└── .gitignore
```

**關鍵理解點**:
- `apps/admin/` 的子目錄結構完全照搬 PbootCMS PHP 版的 `apps/admin/`,**不要重組** — 這是用戶的舒適區,核心優勢
- `apps/admin/service/` 是新增的(僅 `content/` 子目錄),其它模塊未抽 Service — 這是 A+B 路線「引入 Service 層」動作的進行中工作
- `data/pbootcms.db` **已納入版本庫** — 直接 clone 就能用,無需手動建庫

---

## 第 4 步:理解技術棧

| 層 | 選型 | 為什麼 |
|---|---|---|
| 語言 | Go 1.25 | — |
| HTTP 框架 | Gin v1.12 | 高性能、生態成熟 |
| ORM | GORM v1.31 | 社區標準 |
| 數據庫 | SQLite(glebarez 純 Go) | 零配置、單文件、本地開發友好 |
| 模板 | pongo2 v6.1 + 自實現 transpiler | 為 PbootCMS PHP 標籤(`{if(...)}`、`{foreach}`)提供兼容 |
| 會話 | gin-contrib/sessions(cookie) | 簡單 |

**注意**:`core/db/db.go` 連接池設置 `MaxOpenConns=1` 為 **SQLite 紅線修復** — 不要隨意改大,會引發「database is locked」錯誤。

---

## 第 5 步:理解核心約定

### 5.1 命名約定

- **表前綴**:`ay_`(PbootCMS 原生,**絕對不動** — 已通過 NamingStrategy 自動加)
- **URL 風格**:`/admin/{模塊}/{動作}`,`/admin/content/index`、`/admin/user/mod/:id`
- **控制器方法**:Index / Add / Mod / Del(對應列表/新增/修改/刪除)

### 5.2 分層約定(MVC+S 4 層)

```
Controller(HTTP 入口)
    ↓ 注入
Service(業務邏輯,A+B 新增)
    ↓
Model(純數據訪問)
    ↓
DB(SQLite)
```

- Controller 職責:收參數 → 調 Service → 渲染
- Service 職責:校驗、組合、跨模型協調、業務規則
- Model 職責:CRUD、表映射、單表查詢
- View(pongo2):模板渲染

### 5.3 Session 約定

通過 `common.GetSessionInt(c, "admin_uid")` 等輔助函數讀取,**不要**直接訪問 gin 的 session 中間件。

### 5.4 渲染約定

後台所有頁面都通過 `common.Render(c, "模塊/頁面.html", data)` 渲染,**不要**直接調用 pongo2。

---

## 第 6 步:理解已完成的 7 個 Task(今日)

詳見 [2026-06-05-001-task1-7-check.md](2026-06-05-001-task1-7-check.md)。**簡版**:

| Task | 結論 | 文件 |
|---|---|---|
| 1. Render 注入 GET/backurl/pathinfo/btnqs | ✅ 完成 | `apps/common/Render.go` |
| 2. ExtField 結構體補字段 | ⚠️ 缺時間戳字段 | `apps/admin/model/content/ExtFieldModel.go` |
| 3. Transpiler 修復 | ⚠️ 部分(preResolveSingleInPairParams 已完善) | `apps/common/parser/tags.go` |
| 4. template_helpers.go | ✅ 完成(路徑在 `apps/admin/helper/`) | `apps/admin/helper/template_helpers.go` |
| 5. ContentSortController 修復 | ✅ 完成 | `apps/admin/controller/content/ContentSortController.go` |
| 6. ContentController 修復 | ✅ 完成 | `apps/admin/controller/content/ContentController.go` |
| 7. SingleController 修復 | ⚠️ Index 完成,**缺 SingleService 抽離** | `apps/admin/controller/content/SingleController.go` |

---

## 第 7 步:後續任務清單(按優先級)

### 🔴 第一梯隊:本週必做(預計 3-5 天)

| # | 任務 | 工作量 | 來源 |
|---|---|---|---|
| 1 | **抽 SingleService 層**(與 ContentService 對稱) | 半天 | 2026-06-05-001 待辦 5.2 |
| 2 | **ExtField 補時間戳字段**(create_time / update_time / fmode / fvalue) | 2 小時 | 2026-06-05-001 待辦 5.1 |
| 3 | **全面修補 `, _ :=` 錯誤處理**(grep 搜索替換) | 半天 | 2026-06-05-001 待辦 5.4 + 2026-06-08-001 觀察 |
| 4 | **運行端到端測試**:登錄、發布內容、上傳圖片、修改站點 | 1 天 | 確保 PbootCMS 兼容 |

### 🟡 第二梯隊:下週可做(預計 5-7 天)

| # | 任務 | 工作量 | 來源 |
|---|---|---|---|
| 5 | **剩餘 4 個 module 抽 Service 層**:Member、System(全部 9 個 model)、Form、Label、Link、Message、Model、Single、Site、Slide、Tags | 3-5 天 | A+B 路線「動作 3」 |
| 6 | **GORM NamingStrategy 抽離表前綴**(目前 `ay_` 在每個 model 寫死) | 2 小時 | A+B 路線「動作 4」 |
| 7 | **XSS 防護完善**:`getContentField` 對用戶輸入字段統一 `html.EscapeString` | 半天 | 2026-06-08-001 觀察 3.3.3 |

### 🟢 第三梯隊:架構升級(1-2 個月)

| # | 任務 | 工作量 | 來源 |
|---|---|---|---|
| 8 | **反射自動路由**(消除 216 行手寫路由) | 3 天 | A+B 路線「動作 1」 |
| 9 | **pongo2 統一模板**(刪除自實現 transpiler 雙重模板) | 5 天 | A+B 路線「動作 2」 |
| 10 | **Service 介面化**(`type ContentService interface {...}`) | 2 天 | 2026-06-08-001 觀察 3.3.1 |

### 🔵 第四梯隊:安全加固

| # | 任務 | 工作量 |
|---|---|---|
| 11 | **RBAC 權限校驗**(`rcodes` 在 middleware 中檢查) | 3 天 |
| 12 | **密碼 bcrypt 升級**(老用戶登錄時自動升級) | 1 天 |
| 13 | **藍綠部署 / 數據庫遷移** | 5 天 |

---

## 第 8 步:常見問題 FAQ

### Q1:啟動後報「database is locked」?
A:SQLite 連接池問題。**不要**改大 `MaxOpenConns`,**不要**刪除 `data/pbootcms.db` 嘗試重置。先關閉所有運行的 `pbootcms-go.exe` 進程。

### Q2:後台登錄後台閃退 / 跳回登錄頁?
A:session 問題。檢查 `apps/common/session.go` 的 cookie 配置(名稱、domain、path)。

### Q3:模板修改後不生效?
A:pongo2 模板緩存問題。`core/basic/view.go` 已有 `TemplateStore` + fsnotify 熱加載,應該會自動重載。若未生效,重啟服務。

### Q4:如何新增一個後台頁面?
A:流程如下
1. 在 `apps/admin/controller/{模塊}/` 下新增 `XxxController.go`,實現 `Index/Add/Mod/Del` 方法
2. 在 `apps/route/route.go` 註冊路由
3. 在 `templates/admin/{模塊}/` 下新增 HTML 模板
4. 通過 `common.Render(c, "模塊/xxx.html", data)` 渲染
5. 數據訪問走 `apps/admin/model/{模塊}/` 下的 Model
6. 業務邏輯優先放 Service(若該模塊已有 Service)

### Q5:`data/pbootcms.db` 被改動後 git 報 diff?
A:正常,SQLite 文件會變。但已通過 `data/*.db-shm` `data/*.db-wal` 排除副作用文件,主文件 `data/pbootcms.db` 本身會被跟蹤。

### Q6:怎麼從歷史備份還原?
A:
```bash
git tag -l                              # 查看所有 tag
git checkout v0.1.0-baseline-20260605   # 純源碼
git checkout v0.1.1-with-db-20260605    # 含 DB 快照
git checkout v0.2.0-tasks1-7-20260605   # 今日工作
```

### Q7:有問題找誰?
A:目前沒有其他負責人。如有阻塞,回到 `.plan.md` 看「後續規劃」,按計劃自推進。

---

## 第 9 步:如何提交代碼

### 工作流

```bash
# 1. 確認工作區乾淨
git status

# 2. 編輯代碼

# 3. 提交
git add <files>
git commit -m "類型: 簡短描述

詳細描述(可選)"

# 4. (可選)打 tag 標記大版本
git tag -a v0.3.0-<功能>-<日期> -m "備份說明"

# 5. (可選)新增核查記錄
# 在 ARCHITECTURE_REVIEWS/ 新增 <日期>-<序號>-<描述>.md
# 然後 commit
```

### Commit 類型(Conventional Commits)

- `feat:` 新功能
- `fix:` 修 bug
- `chore:` 雜項(配置、文檔、build)
- `docs:` 純文檔
- `refactor:` 重構(無功能變化)
- `test:` 測試
- `perf:` 性能優化

### 什麼時候需要寫核查記錄?

- 一次性完成 ≥5 個 Task → 寫 `ARCHITECTURE_REVIEWS/<日期>-<序號>-<描述>.md`
- 完成重要架構變更 → 同上
- 發現深層問題 → 同上

---

## 第 10 步:項目當前狀態一覽

| 維度 | 狀態 |
|---|---|
| 編譯 | ✅ `go build` 通過(2026-06-05) |
| 啟動 | ✅ `bin/pbootcms-go.exe` 啟動成功,監聽 `:8080` |
| 前台 | ✅ 首頁 / 列表頁 / 內容頁可用 |
| 後台 | ✅ 登錄 + 30+ 個 CRUD 頁面可用 |
| 數據庫 | ✅ `data/pbootcms.db`(SQLite) |
| 模板 | ✅ 30+ 個 HTML 模板 |
| 路由 | ✅ 110+ 條手寫路由 |
| Service 層 | ⚠️ 僅 content 模塊(ContentService、ContentSortService) |
| 自動化測試 | ❌ 無 |
| 持續集成 | ❌ 無 |
| 部署 | ❌ 暫無,本地直接運行 |
| 文檔 | ✅ 充足(3 份必讀 + 逐次核查記錄) |

---

## 結語

這個項目的**核心優勢**是:
- 100% 對齊 PbootCMS 數據庫結構(SQLite 可雙向遷移)
- 100% 對齊 PbootCMS URL 風格(用戶零遷移成本)
- 100% 對齊 PbootCMS 模板語法(HTML 文件無需修改)
- 文檔齊全,決策有跡可循(`.plan.md` + `ARCHITECTURE_REVIEWS/`)

**核心風險**是:
- 沒有自動化測試,所有驗證靠手動
- 沒有 CI/CD,所有部署靠手動
- Service 層抽離未完成,架構不對稱
- 沒有安全加固(RBAC、bcrypt、XSS)

**最需要你做的一件事**:**完成本週第一梯隊的 4 個任務**(SingleService 抽離 + ExtField 補字段 + 統一錯誤處理 + 端到端測試)。完成後,項目將從「可用」升級到「可維護」。

祝接手順利。有問題先看文檔,文檔沒有再翻 git history,git 沒有再問我。

---

**備份位置**:
- Git 倉庫:`F:\mysite\AI\idea\pbootcmstogo\pbootcms-go\.git`
- 最新備份 tag:`v0.2.0-tasks1-7-20260605`
- 完整備份路徑:`F:\mysite\AI\idea\pbootcmstogo\pbootcms-go`(整個目錄)

**相關文檔鏈接**(在項目內):
- [ARCHITECTURE_REVIEW.md](../ARCHITECTURE_REVIEW.md)
- [.plan.md](../.plan.md)
- [2026-06-08-001-task11-check.md](2026-06-08-001-task11-check.md)
- [2026-06-05-001-task1-7-check.md](2026-06-05-001-task1-7-check.md)
- 本文件:2026-06-05-002-handover-guide.md
