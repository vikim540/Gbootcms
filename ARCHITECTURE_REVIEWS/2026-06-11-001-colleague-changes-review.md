# 同事修改核查記錄 — 2026-06-10 ~ 06-11

> **記錄編號**：2026-06-11-001
> **記錄類型**：代碼變更核查 + 現狀盤點
> **核查者**：AI Agent
> **核查範圍**：`daa3287..aed59ab`（12 次提交，23 個文件，+1445/-401 行）

---

## 一、變更總覽

### 1.1 提交時間線

| 日期 | 提交數 | 主要工作 |
|---|---|---|
| 2026-06-10 10:00–12:00 | 2 | Transpiler splitUrlSegments 修復 + scode 路由修復 |
| 2026-06-10 14:00–15:00 | 2 | get_btn_mod/del URL + transpiler 大小寫 + 模板下拉框 |
| 2026-06-10 15:30–16:30 | 3 | URL去.html + 單頁模板選擇 + 側邊欄菜單調整 |
| 2026-06-10 17:00–18:30 | 4 | 欄目內容管理全鏈路修復 + 通知改 toast |
| 2026-06-11 09:48 | 1 | snakeToPascal compoundMap 第三副本 |

### 1.2 文件變更統計

| 文件 | 變更 | 類型 |
|---|---|---|
| `core/basic/view.go` | +281 行 | Transpiler 核心修復 |
| `apps/common/Render.go` | +65 行 | 路徑參數注入 + compoundMap |
| `apps/admin/service/content/ContentSortService.go` | +70 行 | scode/id fallback |
| `apps/home/controller/front.go` | +72 行 | 前台動態路由 |
| `apps/admin/helper/template_helpers.go` | +56 行 | compoundMap + ParseWildcardAction |
| `templates/admin/content/single.html` | 395→168 行 | 重寫為 pongo2 語法 |
| `static/admin/js/mylayui.js` | +27 行 | toast 通知 + 自動跳轉 |
| `apps/common/parser/providers.go` | +17 行 | URL .html 去除 + 動態 fallback |
| `apps/admin/controller/content/ContentSortController.go` | +23 行 | scode 保存修復 |
| `apps/admin/controller/content/SingleController.go` | +20 行 | mcode 查詢修復 |
| `templates/about.html` | 新建 22 行 | 單頁前端模板 |
| `main.go` | +7 行 | 前台路由註冊 |
| `AI_GUIDELINES.md` | 新建 102 行 | AI 助手指南 |

---

## 二、逐項核查

### 2.1 Transpiler 修復（core/basic/view.go）

**splitUrlSegments 重寫** ✅ 正確

同事採用與我之前思路一致的 PHP token-aware parser，關鍵改進：
- 引號內容 `strings.Trim(quoted, ".")` 去除 PHP concat 殘留的 `.`
- 閉引號後自動跳過 PHP concat `.`
- `$` 變量掃描允許 `-` 字符（支持 `$val1->id`）
- 支持 `get('mcode')` 函數調用作為獨立 token
- 支持 `{{ ... }}` pongo2 變量作為獨立 token

**convertUrlSegment 擴展** ✅ 正確
- 新增 bracket 變量：`[$var1->$var2]`、`[$var.field]`、`[$var]`
- 新增 PHP 常量識別：`C`、`URL`、`SITE_DIR` 等
- `get()` 正則放寬為 `get\(['"]?(\w+)['"]?\)`

**processBracketDynamicVars** ✅ 正確
- 僅在 `{fun=...}` 內轉換 `[$var1->$var2]`，避免污染 `{if}` 和 JS

**processPongo2Foreach 增強** ✅ 正確
- `[value]` → `{{ valN }}`、`[key]` → `{{ key }}` 在 foreach 循環內替換

**isLoopVar 擴展** ✅ 正確
- 新增 `value`、`key`、`v`、`k`（PHP foreach 參数名）

**convertArrowToDot 第三副本 compoundMap** ✅ 正確
- view.go、Render.go、template_helpers.go 三處 snakeToPascal 均同步 compoundMap

### 2.2 Render.go 修復

**parsePathParams** ✅ 正確
- 從 URL 路徑 `/admin/xxx/mod/scode/1/field/status/value/0` 解析 key-value 對
- 注入為 `get_scode`、`get_id` 等模板變量（模擬 PHP `$_GET`）

**compoundMap** ✅ 正確
- 處理 PbootCMS 無底線分隔的複合詞（`contenttpl` → `ContentTpl`）
- 三處 `snakeToPascal` 函數均已同步

### 2.3 ContentSort CRUD 修復

**ContentSortService scode/id fallback** ✅ 正確
- `GetSortByScode`、`UpdateSortByScode`、`UpdateSortByScodeField`、`DeleteSortByScode` 全部加了 id fallback
- 解決了 scode 為空或不存在時的靜默失敗

**ContentSortController.Mod** ✅ 正確
- `scode` 不再無條件寫入 updates（避免空字符串覆蓋）
- `_lookup_by` marker 支持 `123,scode` URL 格式
- type=1 時創建初始內容用 URL scode 作 fallback

### 2.4 Single 頁面重寫

**single.html 模板** ✅ 正確（但有犧牲）
- 從 395 行 PHP 語法重寫為 168 行 pongo2 語法
- 繞過了 transpiler，直接寫 pongo2 模板
- **犧牲**：移除了部分功能（圖片上傳、輪播多圖、附件、標題顏色、時間選擇器、UEditor）
- 目前僅保留：標題、來源、副標題、SEO、狀態、內容

**SingleController.Index** ✅ 正確
- 改用 `mcode` 查詢欄目（不再硬編碼 `type=2`）
- 用子查詢 `MAX(id) GROUP BY scode` 取每個欄目最新內容

### 2.5 前台路由

**動態路由** ✅ 正確
- `/sort/:scode` → `SortByScode`（欄目列表頁）
- `/content/:id` → `ContentByID`（內容詳情頁）
- `sortToMap` / `contentToMap` 在 urlname 為空時 fallback 到動態路由

**URL 去 .html** ✅ 正確
- `/{urlname}.html` → `/{urlname}`
- about.html 新建為單頁模板

### 2.6 UI 改進

**mylayui.js toast 通知** ✅ 正確
- 改為非阻塞 toast（左下角，fa-rocket 圖標）
- 保存成功後 1.5 秒自動跳轉（讀取 `returnto` hidden input）

### 2.7 文檔

**AI_GUIDELINES.md** ✅ 重要
- 核心規則：**絕對不要主動修改 data/pbootcms.db**
- 驗證流程：git status → go build → 重啟 → HTTP 探活 → 業務路徑
- 修復記錄模板

---

## 三、當前系統狀態

### 3.1 編譯狀態
- ✅ `go build` 通過
- ⚠️ `go vet ./...` 在 `bin/` 目錄有 redeclared 警告（bin/dbcheck_*.go 是獨立腳本，不影響主程序）

### 3.2 後台管理頁面（同事驗證 25 頁全部 200）
- ✅ Dashboard、Config、User、Menu、Role、Syslog 等系統頁面
- ✅ Content、Single、ContentSort 內容管理頁面
- ✅ Company、Site、Slide、Label 等內容頁面
- ✅ Member、MemberGroup、MemberField、MemberComment 會員頁面

### 3.3 已知殘留問題

| # | 問題 | 嚴重度 | 說明 |
|---|---|---|---|
| 1 | single.html 功能精簡化 | 中 | 移除了圖片上傳、輪播、附件、顏色、時間等高級功能 |
| 2 | content.html 仍用 PHP 語法 | 低 | transpiler 可處理，但 URL 中 `$value->id` 轉 `value.ID` 而非 `val1.ID`（isLoopVar 已修但未驗證） |
| 3 | `check_level` 硬編碼 `true` | 中 | RBAC 權限校驗未實現 |
| 4 | 錯誤處理 `_, _ :=` | 低 | 大量 error 被忽略 |
| 5 | `snakeToPascal` 三處副本 | 低 | compoundMap 需手動同步（已有 3 個副本） |
| 6 | `bin/*.go` vet 警告 | 低 | 獨立腳本被 `go vet ./...` 掃描到 |

---

## 四、待辦事項（從同事文檔彙總）

### 高優先級
| # | 待辦 | 來源 |
|---|---|---|
| 1 | RBAC 權限校驗（check_level 真實實現）| ARCH_REVIEWS 8.6 |
| 2 | XSS 防護（html.EscapeString 統一）| ARCH_REVIEWS 8.8 |
| 3 | Member/System Service 層抽離 | ARCH_REVIEWS 8.3 |
| 4 | 補齊 ExtField 時間戳字段 | ARCH_REVIEWS 8.4 |

### 中優先級
| # | 待辦 | 來源 |
|---|---|---|
| 5 | single.html 恢復高級功能（圖片、附件、輪播）| 本次核查發現 |
| 6 | gurl_build helper 演進 | ARCH_REVIEWS 8.1 |
| 7 | 密碼 bcrypt 升級 | ARCH_REVIEWS 8.7 |
| 8 | `, _ :=` 錯誤處理補全 | ARCH_REVIEWS 8.5 |
| 9 | snakeToPascal 統一抽取到 common 包 | 本次核查發現 |

### 低優先級
| # | 待辦 | 來源 |
|---|---|---|
| 10 | bin/*.go 移出 vet 掃描範圍 | 本次核查發現 |
| 11 | content.html 模板驗證（transpiler 路徑）| 本次核查發現 |

---

**記錄完。**
