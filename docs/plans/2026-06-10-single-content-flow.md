# 欄目內容管理全鏈路修復計劃

**Goal:** 實現「公司簡介」等單頁內容的完整管理流程:後台編輯器修改 → 保存 → 前端頁面即時顯示

**Architecture:** 
- 單頁內容存儲在 `ay_content` 表,通過 scode 關聯 `ay_content_sort`
- 後台通過 `SingleController.Mod()` 保存,前端通過 `renderSortPage()` 加載並用 `{content:xxx}` 標籤渲染
- 模板 `about.html` 需使用 `{content:content}` 而非 `{sort:content}`

**Tech Stack:** Go + Gin + GORM + pongo2 + UEditor

---

## 問題清單

| # | 問題 | 嚴重度 | 文件 |
|---|------|--------|------|
| 1 | `about.html` 用 `{sort:content}` 顯示正文,應為 `{content:content}` | 致命 | `templates/about.html` |
| 2 | `admin/single.html` 用 PHP 語法,pongo2 無法解析 | 致命 | `templates/admin/content/single.html` |
| 3 | `SingleController.Mod` 缺少 picstitle/outlink 保存 | 中 | `SingleController.go` |

## 修復順序

### Step 1: 修復 about.html 前端模板
- `{sort:content}` → `{content:content}`
- `{sort:name}` → `{content:title}`
- SEO 標籤改用 `{content:keywords}` / `{content:description}`

### Step 2: 重寫 admin/single.html 為 pongo2 語法
- 列表視圖:顯示單頁內容列表(標題、欄目、狀態、操作)
- 編輯視圖:標題、UEditor 內容、SEO 字段、狀態

### Step 3: 補全 SingleController.Mod 保存字段
- 添加 picstitle、outlink、update_user

### Step 4: 驗證全鏈路
