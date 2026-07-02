# AI 助手指南 / AI Agent Guidelines

> **所有在本倉庫工作的 AI 助手必須閱讀本文檔並嚴格遵守。**
> AI 助手包括但不限於:Trae、Cursor、Copilot、Cline、Windsurf、ChatGPT、Claude 等。

## 🔒 核心原則 / Core Principles

### 1. **絕對不要主動修改 `data/pbootcms.db`**
- ❌ **禁止**使用任何 SQL 命令直接 INSERT/UPDATE/DELETE 業務資料
- ❌ **禁止**使用 GORM 種子(seeder)重新初始化業務資料
- ❌ **禁止**通過 `go run` 腳本遷移/刪除/重建表結構
- ❌ **禁止**以「重置數據庫」、「整理數據」、「測試用」等理由清空資料表
- ✅ **允許**通過 Web UI 進行的操作(管理後台 CRUD、登入、登出、訪問計數等)
- ✅ **允許**應用程式正常產生的寫入(訪問計數 +1、新建留言、登入日誌等)

**為什麼?**
該資料庫是用戶手工測試積累的業務資料(欄目、內容、配置等),損壞後需耗時重建,
且用戶明確表態:**不允許修改 db 結構、欄位、業務數據。**
訪問計數等透明副作用是**正常且預期的**(如果 PbootCMS 在運行,db 必然會被寫入)。

### 2. **所有代碼修改必須用 diff 形式**
- ❌ 禁止用 `git reset --hard` 還原修改
- ❌ 禁止直接刪除用戶已配置的欄位/表
- ✅ 修改前先 `git status` 確認當前狀態
- ✅ 重要修改前 `git diff` 預覽
- ✅ 修復完成後 `git diff --stat` 確認變更範圍

### 3. **`.plan.md` 是用戶的活筆記**
- ✅ 可以在結尾追加今天的進度
- ❌ 不要刪除歷史記錄
- ❌ 不要重寫整個文件

## 🛡️ 驗證流程 / Verification Workflow

修改代碼後的標準驗證順序:

1. **`git status`** — 確認修改的文件清單
2. **`go build`** — 編譯通過
3. **重啟服務** — 關閉舊進程 → 啟動新進程
4. **HTTP 探活** — `Invoke-WebRequest -Uri 'http://localhost:8080/' -UseBasicParsing`
5. **業務路徑** — 至少驗證 1 個核心 API 正常

## 📁 項目結構速查 / Project Structure

| 路徑 | 用途 | 修改限制 |
|---|---|---|
| `apps/admin/` | 後台管理 API | 自由修改 |
| `apps/home/` | 前台渲染 | 自由修改 |
| `apps/common/parser/` | PbootCMS 標籤解析 | 自由修改 |
| `core/basic/view.go` | PHP 模板轉譯器 | 自由修改 |
| `templates/*.html` | 前台模板 | 自由修改 |
| `templates/admin/*.html` | 後台模板 | 自由修改 |
| `config/config.json` | 配置文件 | 自由修改 |
| **`data/pbootcms.db`** | **業務資料庫** | **🔒 僅 UI 寫入** |
| **`migrations/`** | **遷移腳本** | **🔒 不要新增/刪除** |
| **`.plan.md`** | 用戶活筆記 | **📝 追加,不重寫** |

## 🚨 邊界案例 / Edge Cases

### Q: 我需要測試某個功能,但 db 中沒有對應資料怎麼辦?
A: **提示用戶手動通過後台新增測試資料**,不要在腳本中 insert 模擬資料。

### Q: 我懷疑 db 結構有 bug,可以 migrate 嗎?
A: **不可以**。如果發現結構問題,**先在 issues 記錄,不要自動修復**。表結構是用戶手工管理。

### Q: 訪問計數導致 db 變化,需要 `git checkout` 還原嗎?
A: **不需要**。訪問計數是透明副作用。但如果你寫了腳本,可能污染了其他欄位,
此時可以用 `git checkout -- data/pbootcms.db` 還原到入庫版本(此版本是「測試基準線」)。

### Q: 我能用 `go run` 寫一個遷移腳本嗎?
A: **不要**。如果資料庫需要任何變更,必須由用戶親自確認並執行。

## 📋 修復記錄模板 / Fix Record Template

在 `.plan.md` 中追加修復記錄時使用:

```markdown
## 2026-06-10 [T1] scode 修改按鈕路由修復
- 問題: 點擊修改按鈕 → 404
- 根因: ParseWildcardAction 對 "123,scode" 格式不支持
- 修復: 添加 _lookup_by 標記,Controller 識別後用 scode 查找
- 文件: apps/admin/helper/template_helpers.go, apps/admin/controller/content/ContentSortController.go
- 驗證: curl /admin/contentsort/mod/1,scode → 200
```

## ✅ 確認清單 / Checklist

每次任務結束前:
- [ ] `git status` 確認沒有意外修改 db
- [ ] `git diff --stat` 確認變更範圍合理
- [ ] 所有修改的文件在允許列表中
- [ ] 沒有執行 `rm`、`drop`、`truncate`、`delete from` 等危險命令
- [ ] 用戶確認滿意後再 git commit

## 🔗 相關文檔 / Related Docs

- `ARCHITECTURE_REVIEW.md` — 項目整體架構
- `.plan.md` — 開發進度活筆記
- `build.ps1` — 構建腳本
- `bin/headroom-task-context.mjs` — headroom-ai 任務上下文壓縮
- `runtime/headroom_cache/` — 壓縮緩存目錄
