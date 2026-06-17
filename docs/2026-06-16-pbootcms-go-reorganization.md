# pbootcms-go 安全重組計劃（2026-06-16）

> **狀態**：待執行（Plan 已批准）
> **範圍**：`F:\mysite\AI\idea\pbootcmstogo\pbootcms-go\`
> **約束**：不動 PHP 源、不恢復 `static/upload/`、保留 `docs/`

---

## 摘要

針對 pbootcms-go 進行「止血 → 釐清 → 整理 → 文檔化」四階段重組，零檔案刪除（untracked 殘留由用戶手動處理），所有變更用 `git mv` 保留歷史。

---

## Current State Analysis（read-only 評估的 10 個關鍵事實）

| # | 事實 | 證據 |
|---|------|------|
| F1 | `static/admin/` 是後台靜態資源的事實來源 | `Render.go:34-35` 寫死 `AppThemeDir = "/static/admin"`, `CoreDir = "/static/admin"` |
| F2 | `static/admin/` 的子目錄（css/js/images/font-awesome/layui/extend）**完全對應** PHP `apps/admin/view/default/` 下的子目錄 | 對比 `pbootcms-go/static/admin/layui/css/layui.css` 與 `PbootCMS-3.2.12/apps/admin/view/default/layui/css/layui.css` |
| F3 | `static/upload/202606/20260616135149_0870.png` 與 `20260616140143_4587.png` 是用戶新上傳的檔案，但**不在 git 內**（已被 `.gitignore` 排除） | `git status` 沒有 `static/upload/`；`.gitignore:28` 寫明 `static/upload/` |
| F4 | `data/pbootcms.db` 的「修改」是 SQLite WAL/header 重寫，**實際內容不變**（536576→536576 bytes） | `git diff --stat data/pbootcms.db` 顯示 `Bin 536576 -> 536576 bytes, 0 insertions(+), 0 deletions(-)` |
| F5 | 前台模板的 `{gboot:sitetplpath}` **已經** 對應 `/template/default/static` URL 與 `template/default/static/` 磁盤路徑 | `providers.go:107-112` 與 `main.go:55` `r.Static("/template/default/static", "template/default/static")` |
| F6 | 後台模板用 `{APP_THEME_DIR}/layui/layui.js` 等絕對路徑，與 `static/admin/layui/` 磁盤位置**強耦合** | 後台 `head.html:9-14` |
| F7 | `template/default/comm/head.html:15` 引用了不存在的 `/static/css/md-content.css` | `head.html:15` 與磁盤上 `static/css/` 不存在的事實 |
| F8 | `template/default/static/css/md-content.css` **存在**（commit 9dc8c2c 把 md-content.css 移到此處） | `git ls-files` 顯示該檔案被追蹤 |
| F9 | `tools/checkdb/`、`apps/home/model/`、`.uploads/` 是 untracked 殘留 | `git status --porcelain` 與 `ls` |
| F10 | `apps/home/model/`、`.uploads/` 是 untracked 目錄佔位 | `git ls-files --others --exclude-standard` |

---

## 重要：用戶意圖的修正

**用戶原意**：「重組目錄結構，按 PHP 3.2.12 的規範：後台模板 `apps/admin/view/default/`，子目錄 css/ js/ images/ font-awesome/ layui/ 與 default/ 同級」

**修正**：
- PHP 3.2.12 確實把後台靜態資源放在 `apps/admin/view/default/{css,js,...}`
- 但 Go 版**有意偏離**這個結構：把靜態資源集中到 `static/admin/`，並把 `Render.go` 中的 `AppThemeDir` 設為 `/static/admin`
- 這個偏離是**經過設計的合理優化**（靜態資源與模板分離、便於 CDN 部署、單一 `r.Static` mount）
- **不**把後台資源移回 `apps/admin/view/`（會破壞後台所有頁面，需同步改 `Render.go` 與 `main.go`，超出本次重組範疇）
- **做法**：在 `apps/admin/view/STATIC_FILES.md` 明確記錄這個設計決策

---

## Proposed Changes（具體步驟）

### Phase A：止血（無風險，純修正損壞引用）

#### A1. 修正 `template/default/comm/head.html:15`

- **現況**：第 15 行引用 `/static/css/md-content.css`（不存在）
- **目標**：改為 `{gboot:sitetplpath}/css/md-content.css`（與其他 CSS 引用一致）
- **方法**：`SearchReplace` 工具替換單行字串
- **風險**：🟢 極低

#### A2. 處理 `data/pbootcms.db` 的假修改

- **現況**：`git diff` 顯示改了但 `0 insertions(+), 0 deletions(-)`，是 SQLite WAL 重寫
- **方案**：採用「方案 C」（最保守）
  1. 在 `data/` 加 `README.md` 說明 db 用途
  2. 把 `data/pbootcms.db` 從 staging 取消（`git restore --staged`）
- **禁止操作**：`git checkout -- data/pbootcms.db`（會還原舊 header）
- **風險**：🟡 低

---

### Phase B：目錄結構釐清（新增文檔，不移動大目錄）

#### B1. 在 `apps/admin/view/` 加 `STATIC_FILES.md`

- **內容**：說明後台靜態資源為何放在 `static/admin/` 而非 `view/` 同級
- **風險**：🟢 零（純新增文檔）

#### B2. 在 `template/default/` 加 `STATIC_FILES.md`

- **內容**：說明前台模板 `static/` 子樹的 URL 約定（與 PbootCMS PHP 一致）
- **風險**：🟢 零

#### B3. 在 `static/upload/` 加 `README.md`

- **內容**：說明此目錄為上傳產物、不入庫、用戶自行備份
- **方法**：
  1. 創建 `static/upload/README.md`
  2. 修改 `.gitignore`：把 `static/upload/` 改為 `static/upload/*` + `!static/upload/README.md`
- **風險**：🟡 中（改 `.gitignore`），前置步驟：先 read-only 確認 `static/upload/` 內只有 2 個 PNG
- **驗證**：`Get-ChildItem -Recurse static/upload` 應只看到 2 個 PNG 與新加的 `README.md`

---

### Phase C：清理「無用文件」（全部用 `git mv`）

#### C1. 移動 `ARCHITECTURE_REVIEWS/` 到 `docs/history/architecture-reviews/`

- **方法**：`git mv ARCHITECTURE_REVIEWS docs/history/architecture-reviews`
- **保留歷史**：5 個評測文件完整保留
- **風險**：🟡 低（路徑變更可能破壞外部連結；用戶本地內部引用已無）

#### C2. 移動 `.plan.md` 到 `docs/history/early-restructure-plan.md`

- **內容**：`.plan.md` 是早期重構計畫，內容已併入後續 commits 9dc8c2c/9296d8f/1241067/73725a7
- **方法**：`git mv .plan.md docs/history/early-restructure-plan.md`
- **風險**：🟡 低

#### C3. 處理 untracked 殘留（不在 AI 計劃中）

- `tools/checkdb/`、`apps/home/model/`、`.uploads/`：建議用戶手動決定去留
- 不在本次 AI 操作範圍

---

### Phase D：建立最終匯總文檔

#### D1. 在 `docs/` 加 `REORGANIZATION_REPORT.md`

- **章節**：
  1. 執行摘要
  2. 現況診斷（F1-F10）
  3. 目標目錄結構
  4. 為什麼不做「後台資源移到 view/ 同級」
  5. 執行的 commit 清單
  6. commit 圖
  7. 保留給用戶手動處理的項目
  8. 避免重複犯錯的策略
  9. 相關檔案清單
  10. 附錄
- **方法**：用 `Write` 工具建立完整文檔
- **風險**：🟢 零

---

## Assumptions & Decisions

### 假設
- 用戶希望在 git 內完成所有重組
- `static/admin/` 保留不動（已分析過為何不可移）
- `data/pbootcms.db` 保持追蹤（不影響開發者本地的 admin/admin 帳號）

### 決策
- **每個 commit 一個獨立變更**：便於 `git revert`
- **不使用 `rm` / `rmdir` / `git rm` / `git reset --hard` / `git checkout --`**（除非有磁盤確認步驟）
- **所有移動使用 `git mv`**（保留歷史）
- **任何「刪除」操作前**先 `Get-ChildItem -Recurse` 確認檔案清單
- **不動 PHP 源**（`PbootCMS-3.2.12/`、`.zip`）
- **不恢復 `static/upload/`**（用戶自己解決）
- **保留 `docs/`**（用戶每次檢查，最後手動刪除）
- **不刪 `static/admin/`**（已分析為何不可移）

### 風險矩陣

| 階段 | 操作 | 風險 | 緩解 |
|------|------|------|------|
| A1 | 改 `head.html:15` | 🟢 極低 | `git revert` 一鍵回滾 |
| A2 | `data/pbootcms.db` 取消 stage | 🟡 低 | 嚴格用 `git restore --staged`，不碰磁盤 |
| B1 | 在 `apps/admin/view/` 加文檔 | 🟢 零 | n/a |
| B2 | 在 `template/default/` 加文檔 | 🟢 零 | n/a |
| B3 | 改 `.gitignore` 允許 `static/upload/README.md` | 🟡 中 | 先 read-only 確認內容 |
| C1 | 移動 `ARCHITECTURE_REVIEWS/` | 🟡 低 | `git mv` 保留歷史 |
| C2 | 移動 `.plan.md` | 🟡 低 | `git mv` 保留歷史 |
| D1 | 建立匯總文檔 | 🟢 零 | n/a |

### 絕對禁止
- ❌ `rm -rf static/`
- ❌ `git reset --hard HEAD~5`
- ❌ `git checkout -- <path>`（特別是 `data/pbootcms.db` 與 `static/upload/`）
- ❌ 修改 `Render.go` 的 `AppThemeDir`/`CoreDir`（會破壞後台所有頁面）
- ❌ 動 `F:\mysite\AI\idea\pbootcmstogo\docs\`、`F:\mysite\AI\idea\pbootcmstogo\PbootCMS-3.2.12\`、`*.zip`

---

## 給用戶保留的項目

不動的（用戶自己管理）：
- `F:\mysite\AI\idea\pbootcmstogo\docs\`（用戶每次檢查，最後手動刪除）
- `F:\mysite\AI\idea\pbootcmstogo\PbootCMS-3.2.12\`（唯讀參考源）
- `F:\mysite\AI\idea\pbootcmstogo\PbootCMS-3.2.12.zip`
- `F:\mysite\AI\idea\pbootcmstogo\PbootCMS-V3.2.12.zip`
- `F:\mysite\AI\idea\pbootcmstogo\.trae\`

用戶手動處理（不在 AI 計劃內）：
- `static/upload/202606/20260616135149_0870.png` 與 `20260616140143_4587.png` 備份恢復
- `runtime/*.png` 與 `pongo2_debug_*.html` 磁盤清理
- `tools/checkdb/` 內容確認

---

## Verification Steps（執行後）

每個 Phase 結束後：

### Phase A 驗證
```bash
# A1
curl -I http://localhost:8080/template/default/static/css/md-content.css
# 預期：200
# 瀏覽器 DevTools Network 沒有 404

# A2
git status data/pbootcms.db
# 預期：unstaged 或消失
```

### Phase B 驗證
```bash
# B1, B2
cat apps/admin/view/STATIC_FILES.md
cat template/default/STATIC_FILES.md
# 預期：文檔可讀

# B3
Get-ChildItem -Recurse static/upload
# 預期：2 個 PNG + 1 個 README.md
git status
# 預期：static/upload/README.md 顯示為 tracked
```

### Phase C 驗證
```bash
# C1
git log --follow docs/history/architecture-reviews/2026-06-05-001-task1-7-check.md
# 預期：可看到原始 commit

# C2
git log --follow docs/history/early-restructure-plan.md
# 預期：可看到原始 .plan.md 的 commit
```

### Phase D 驗證
```bash
# D1
cat docs/REORGANIZATION_REPORT.md
# 預期：完整文檔可讀
```

### 整體驗證
```bash
# 1. 前台能訪問
curl -I http://localhost:8080/
curl -I http://localhost:8080/aboutus
curl -I http://localhost:8080/article
# 預期：全部 200

# 2. 後台能訪問
curl -I http://localhost:8080/admin/
# 預期：200

# 3. 靜態資源能訪問
curl -I http://localhost:8080/static/admin/css/comm.css
curl -I http://localhost:8080/template/default/static/bootstrap/css/bootstrap.min.css
curl -I http://localhost:8080/static/upload/202606/20260616135149_0870.png
# 預期：全部 200

# 4. git 歷史完整
git log --oneline -10
# 預期：看到所有新 commit
```

---

## 預期 commit 清單

| # | Commit | 階段 |
|---|--------|------|
| 1 | `fix(template): 修正 comm/head.html 中 md-content.css 路徑` | A1 |
| 2 | `docs(data): 說明 pbootcms.db 維護注意事項` | A2 |
| 3 | `docs(admin): 說明後台靜態資源為何放在 static/admin/` | B1 |
| 4 | `docs(template): 說明前台模板 static/ 子樹的 URL 約定` | B2 |
| 5 | `chore(static): static/upload/ 加入 README；放寬 .gitignore 允許 README` | B3 |
| 6 | `docs: 整理歷史文檔，移到 docs/history/` | C1+C2 |
| 7 | `docs: 新增 2026-06-16 目錄重組完整匯總報告` | D1 |

---

## 避免重複犯錯的策略

1. **執行前 read-only 確認**：每個 `rm`/`mv`/`rmdir` 前先用 `Get-ChildItem -Recurse` 列出檔案清單
2. **每個 commit 一個獨立變更**：避免「一次改 5 個東西」
3. **不使用 `git reset --hard` / `git checkout --`**：所有還原走 `git revert <commit>` 與 `git restore --staged`
4. **遵循 `AppThemeDir = /static/admin` 這個事實**：後台資源的 URL 約定已寫死在 Render.go
5. **遇到「該移到哪」的問題時，先看 `git log --follow <file>` 找最近的遷移決策**
6. **永遠不在 AI 計劃中包含 `rm -rf`**：刪除操作統一建議用戶手動處理
7. **每次 commit 前跑 `git diff --stat` 與 `git status`**：防止「假修改」（如 SQLite WAL 重寫）混入