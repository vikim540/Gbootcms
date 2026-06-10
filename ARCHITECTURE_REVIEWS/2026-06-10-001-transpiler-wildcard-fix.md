# Task 8-11 核查記錄 — 2026-06-10

> **記錄編號**：2026-06-10-001
> **記錄類型**：Transpiler + Wildcard Routes 修復核查
> **評測者視角**：專業架構師，僅只讀分析 + 修復驗證
> **被核查對象**：handover 文檔中「本週第一梯隊」中的 Transpiler + Wildcard Routes 修復

---

## 一、任務背景

handover 文檔 `2026-06-05-002-handover-guide.md` 第 7 步「第一梯隊」4 個任務中，第 1、2 個任務已於 2026-06-05 完成（見 `2026-06-05-001-task1-7-check.md`）。本輪工作聚焦於後續發現的 transpiler 缺陷與 wildcard 路由驗證。

具體三項：
- T1：修復 `processUrlConcat` 引號感知花括號追蹤
- T2：修復 `check_level` 在 `{if}` 內的處理
- T3：驗證 ContentSort Mod scode 查找

---

## 二、T1 詳細分析：processUrlConcat 引號感知

### 2.1 用戶描述的問題

> processUrlConcat 的花括號深度追踪不考虑单引号，导致 `{url./admin/'.C.'/mod/id/'.$val1->id.'}` 中 `'.C.'` 的 `{` 被错误计为模板变量开始，URL 在错误位置截断

### 2.2 實證：原代碼的實際問題

通過 debug logging 發現，**真正的根因不是 processUrlConcat 的花括號追蹤**（line 891-902 的 `inQuote` 邏輯已正確），而是下游 `splitUrlSegments` 對引號內容的處理：

```go
// 原代碼 line 964-975
if s[i] == '\'' {
    i++ // skip opening quote
    start := i
    for i < len(s) && s[i] != '\'' {
        i++
    }
    segments = append(segments, s[start:i])
    if i < len(s) {
        i++ // skip closing quote
    }
    continue
}
```

對於 `'.C.'`：
- 進入引號處理模式後，`start=1`（指向 `.`）
- 掃描到 `'`（`i=3`）
- `segments = s[1:3] = ".C"`（**包含前導 `.`**）
- 跳過閉引號後，外層 `if s[i] == '.'` 又跳過了 `C` 後面的 `.`
- 結果：`.C` 進入 segments → convertUrlSegment 返回原文 `.C`

對於 `'.$val1->id.'`：類似，得到 `.$val1->id`（前導 `.`）。

### 2.3 第二個 bug：$ 變量跳過 `-`

```go
// 原代碼 line 977-985
if s[i] == '$' {
    start := i
    i++ // skip $
    for i < len(s) && (isWordChar(s[i]) || s[i] == '>') {
        i++
    }
    segments = append(segments, s[start:i])
    continue
}
```

對於 `$val1->id`：遇到 `-` 停止（`-` 不是 wordChar 也不是 `>`），segments = `$val1`。這對第二輪 processUrlConcat 處理 get_btn_mod 生成的 URL 是 bug。

### 2.4 修復方案

```go
// 修復後的引號處理：剝皮引號內的前後 .，並跳過閉引號後的 PHP concat .
if s[i] == '\'' {
    i++
    start := i
    for i < len(s) && s[i] != '\'' {
        i++
    }
    quoted := s[start:i]
    quoted = strings.Trim(quoted, ".")
    segments = append(segments, quoted)
    if i < len(s) {
        i++ // skip closing quote
    }
    if i < len(s) && s[i] == '.' {
        i++ // skip PHP concat dot after closing quote
    }
    continue
}

// 修復後的 $ 變量處理：允許 `-` 用於箭頭運算符
if s[i] == '$' {
    start := i
    i++ // skip $
    for i < len(s) && (isWordChar(s[i]) || s[i] == '>' || s[i] == '-') {
        i++
    }
    segments = append(segments, s[start:i])
    continue
}
```

### 2.5 修復效果驗證

| 輸入 | 修復前 | 修復後 |
|---|---|---|
| `{url./admin/'.C.'/mod/id/'.$val1->id.'/field/status/value/0}` | `/admin/.C./mod/id/.$val1->id./field/status/value/0` | `/admin/{{ C }}/mod/id/{{ val1.ID }}/field/status/value/0` |
| `{url./admin/'.C.'/mod/{{ val1.ID }}/field/status/value/0}` | `/admin/.C./mod/{{ val1.ID }}/field/status/value/0` | `/admin/{{ C }}/mod/{{ val1.ID }}/field/status/value/0` |
| `{url./admin/{{ C }}/mod/$val1->id}`（第二輪） | `/admin/{{ C }}/mod/$val1`（截斷） | `/admin/{{ C }}/mod/{{ val1.ID }}` |

---

## 三、T2 分析：check_level 在 {if} 內的處理

### 3.1 用戶描述

> check_level('mod') 出現在 {if(check_level('mod'))} 中，不是 {fun=...} 格式，所以 reCheckLevel（僅匹配 {fun=check_level(...)}）不生效。processPongo2If → convertPongo2Condition 中沒有對 check_level 的處理。

### 3.2 實證

`core/basic/view.go:575` 已實現：

```go
// check_level('xxx') → true (permission check placeholder)
cond = regexp.MustCompile(`check_level\([^)]*\)`).ReplaceAllString(cond, "true")
```

**此項已實現**，T2 在 git HEAD 上已生效。

### 3.3 驗證

debug HTML 輸出檢查（content.html line 141）：

```html
{% if true %}
    <a class="layui-btn layui-btn-xs" ...>修改</a>
{% endif %}
```

→ `check_level('mod')` 已正確替換為 `true`。

---

## 四、T3 驗證：ContentSort Mod scode 查找

### 4.1 測試結果

| URL | HTTP | 內容驗證 |
|---|---|---|
| `/admin/contentsort/mod/scode/1` | 200 | 顯示「公司簡介」表單 |
| `/admin/contentsort/mod/scode/2` | 200 | 顯示對應欄目 |
| `/admin/contentsort/mod/scode/10001` | 200 | 「sort does not exist」（預期，DB 無此 scode）|
| `/admin/contentsort/mod/id/1` | 200 | 正常 |
| `/admin/contentsort/mod/1` | 200 | 正常（自動判定為 id） |

**T3 通過**：scode 查找邏輯正確（`ContentSortService.GetSortByScode`），無需修改。

---

## 五、整體頁面驗證（T4）

通過 `test_all_pages.ps1` 對 25 個後台管理頁面進行批量 HTTP 狀態碼 + 內容檢查：

| 維度 | 結果 |
|---|---|
| 編譯 | ✅ `go build` 通過 |
| 啟動 | ✅ `bin/pbootcms-go.exe` 啟動成功 |
| 25 個後台頁面 | ✅ 全部 200，無 pongo2 編譯失敗 |
| URL 正確性 | ✅ 操作按鈕 URL 全部正確生成（`/admin/{{ C }}/mod/...`） |

### 5.1 驗證的 25 個頁面

| # | 名稱 | 路由 | 結果 |
|---|---|---|---|
| 1 | Dashboard | /admin/index/home | 200 |
| 2 | Config | /admin/config/index | 200 |
| 3 | User | /admin/user/index | 200 |
| 4 | Menu | /admin/menu/index | 200 |
| 5 | Role | /admin/role/index | 200 |
| 6 | Syslog | /admin/syslog/index | 200 |
| 7 | Database | /admin/database/index | 200 |
| 8 | DeleCache | /admin/content/deleCache/index | 200 |
| 9 | Company | /admin/company/index | 200 |
| 10 | Site | /admin/site/index | 200 |
| 11 | Slide | /admin/slide/index | 200 |
| 12 | Label | /admin/label/index | 200 |
| 13 | Link | /admin/link/index | 200 |
| 14 | Tags | /admin/tags/index | 200 |
| 15 | Form | /admin/form/index | 200 |
| 16 | Message | /admin/message/index | 200 |
| 17 | Model | /admin/model/index | 200 |
| 18 | ExtField | /admin/extfield/index | 200 |
| 19 | Member | /admin/member/index | 200 |
| 20 | MemberGroup | /admin/membergroup/index | 200 |
| 21 | MemberField | /admin/memberfield/index | 200 |
| 22 | MemberComment | /admin/membercomment/index | 200 |
| 23 | Content | /admin/content/index?mcode=1 | 200 |
| 24 | Single | /admin/content/single/index?mcode=2 | 200 |
| 25 | ContentSort | /admin/contentsort/index | 200 |

---

## 六、goframe 啟發：中期重構方向

> 用戶在過程中提問：「goframe 是如何處理這種修改編輯的連接的？是否有值得借鑒的地方？」

goframe（特別是 `ghttp` + `gview` + `gdb`）處理「帶動態變量的 URL/連結生成」的思路是**顯式構造**，而不是字符串拼接解析：

### 6.1 goframe 的四個關鍵設計

| 維度 | goframe 做法 | 對本項目的啟發 |
|---|---|---|
| **路由參數化** | `s.BindHandler("/admin/content/mod/{id}/field/{field}/value/{value}", handler)` | 顯式聲明，避免模板裡的字符串拼接 |
| **DTO 結構體綁定** | `type ModReq struct { g.Meta "path:/admin/content/mod method:post"; ID int; Field string; Value string }` | 自動綁定，無需 `c.Query("id")` 一個個取 |
| **HTML 模板直出** | `<a href="/admin/content/mod/{{.id}}/field/status/value/0">` | **沒有 PHP 拼接語法**，URL 路徑常量直接寫死 |
| **路由前綴統一管理** | `s.Group("/admin", ...)` + 結構體反射 | 模塊名從結構體映射出來，**不寫在模板裡** |

### 6.2 為什麼本項目短期不能直接學

goframe 沒有 PHP→Go 翻譯包袱，而 PbootCMS 模板是 PHP 版本搬過來的，**不能改 HTML**。所以我們**只能在 Go transpiler 層做一次轉換**，無法直接換模板語法。

### 6.3 中期可行方案

| 步驟 | 工作 |
|---|---|
| 1 | 保留當前 transpiler 修復（短期穩定） |
| 2 | 引入 `gurl.Build(modName, "mod", id, "field", "status", "value", 0)` helper |
| 3 | 將模板裡的 `{url./admin/'.C.'/mod/'.$val1->id.'/...}` 逐步替換為 `{{ gurl_build("mod", val1.ID, "field", "status", "value", 0) }}` |
| 4 | 明確的 helper 函數，不再走字符串拼接解析 |

**本質**：從「PHP 字符串拼接 → 正則提取 → pongo2 重組」三步 hack，演進為「顯式 helper 調用 → 模板渲染」一步到位。

### 6.4 業界標準對標

| 框架 | URL 生成方式 |
|---|---|
| Rails | `link_to "Edit", edit_article_path(@article)` |
| Django | `{% url 'article-edit' article.id %}` |
| Laravel | `route('article.edit', ['id' => $article->id])` |
| Spring | `@GetMapping("/articles/{id}/edit")` + `Model.addAttribute` |
| **goframe** | `s.BindHandler("/articles/{id}/edit", handler)` + struct tag |

**共同點**：URL 是結構化的，不是字符串拼接。

---

## 七、整體評分

| 維度 | 評分 | 說明 |
|---|---|---|
| 編譯 | ⭐⭐⭐⭐⭐ | `go build` 一次通過 |
| 啟動 | ⭐⭐⭐⭐⭐ | 啟動正常，無 database lock |
| 25 頁面可用性 | ⭐⭐⭐⭐⭐ | 全部 200，無編譯錯誤 |
| URL 正確性 | ⭐⭐⭐⭐⭐ | 操作按鈕 URL 全部正確（`{{ C }}` `{{ val1.ID }}`）|
| 代碼健壯性 | ⭐⭐⭐ (3/5) | transpiler 仍是字符串拼接 hack，未來需要 gurl helper 演進 |
| 可維護性 | ⭐⭐⭐ (3/5) | splitUrlSegments 仍是正則解析邏輯，複雜度高 |

**整體**：⭐⭐⭐⭐ (4/5)。**本週第一梯隊的 transpiler+wildcard 修復全部完成**。

---

## 八、後續待辦

| # | 待辦 | 優先級 |
|---|---|---|
| 8.1 | 將 transpiler 中所有字串拼接邏輯包裝為 `gurl_build` helper，逐步替換模板 | ⭐⭐⭐ 中 |
| 8.2 | 重構 splitUrlSegments 為 PHP token-aware parser（取代正則拼接）| ⭐⭐ 低 |
| 8.3 | 補齊第二梯隊：Member/System 模塊的 Service 層抽離 | ⭐⭐⭐⭐ 高 |
| 8.4 | 補齊 ExtField 時間戳字段（create_time / update_time / fmode / fvalue）| ⭐⭐ 中 |
| 8.5 | 全面修補 `, _ :=` 錯誤處理 | ⭐⭐⭐ 中 |
| 8.6 | RBAC 權限校驗（`check_level` 從 true 替換為真實校驗）| ⭐⭐⭐⭐ 高 |
| 8.7 | 密碼 bcrypt 升級 | ⭐⭐⭐ 中 |
| 8.8 | XSS 防護完善（`getContentField` 用戶輸入字段統一 `html.EscapeString`）| ⭐⭐⭐⭐ 高 |

---

## 九、附錄：本次修改的文件清單

| 文件 | 變更 |
|---|---|
| `core/basic/view.go` | 修復 `splitUrlSegments` 引號處理（剝皮 `.`）+ `$` 變量允許 `-` |
| `runtime/pongo2_debug_*.html` | 自動生成（debug 輸出，驗證用）|
| `runtime/server.err.log` | 啟動日誌（驗證用）|

**未修改**：controller、model、service、route（transpiler 修復即可恢復 URL 正確性，無需改業務層）。

---

**記錄完。**
**下一步**：等待用戶確認是否啟動 8.1-8.8 改進項。