# AI 助手指南 / AI Agent Guidelines

> **所有在本倉庫工作的 AI 助手必須閱讀本文檔並嚴格遵守。**
> AI 助手包括但不限於:Trae、Cursor、Copilot、Cline、Windsurf、ChatGPT、Claude 等。
> 最後更新：2026-07-08

---

## 一、防遺忘清單（寫代碼前必讀）

> 以下問題在開發過程中反覆出現，每次修改代碼前務必對照此清單。

### 1.1 寫代碼前的強制檢查流程

```
1. 讀 project_memory.md — 檢查硬約束（如 mcode 而非 modelcode）
2. 讀本文檔第一章 — 檢查 API 速查表和反模式
3. Grep 一個已實現的同類控制器 — 參考其模式
4. 確認方法/函數存在 — Grep 函數名，不要猜測
5. 確認模型欄位名 — 讀 Model struct，不要猜測
```

### 1.2 API 速查表（不要猜，查這裡）

#### BaseController 方法（apps/common/BaseController.go）

| 方法 | 簽名 | 用途 |
|------|------|------|
| `JSONOK` | `(c, data)` | 成功響應 `{"code":1,"data":...}` |
| `JSONOKMsg` | `(c, msg)` | 成功響應帶消息 `{"code":1,"msg":...}` |
| `JSONFail` | `(c, msg)` | 失敗響應 `{"code":0,"msg":...}` |
| `BatchSort` | `(c, modelPtr, sortCol, defaultSort)` | 通用批量排序 |
| `GetAdminUsername` | `(c) string` | 從 Session 獲取管理員用戶名 |
| `GetAdminUID` | `(c) int` | 從 Session 獲取管理員 UID |
| `GetAdminUcode` | `(c) string` | 從 Session 獲取管理員 ucode |
| `IsLogin` | `(c) bool` | 判斷登錄態 |
| `IsBatchSort` | `(c) bool` | 判斷是否批量排序請求 |

> **不存在的方法（曾誤用）**：`JSONErrMsg`、`GetAutoCode`、`AdminController.InitAdmin`

#### 後台控制器標準模式

```go
// 列表頁必須傳遞 list 標誌
common.Render(c, "module/template.html", gin.H{
    "list":   true,        // ← 必須！模板用 {if([$list])} 判斷
    "items":  items,
    "C":      "module",    // 當前控制器路徑（模板生成URL用）
})

// 修改頁必須傳遞 mod 標誌
common.Render(c, "module/template.html", gin.H{
    "mod":    true,        // ← 必須！模板用 {if([$mod])} 判斷
    "item":   item,
    "C":      "module",
})
```

> **遺忘後果**：模板的 `{if([$list])}` 和 `{if([$mod])}` 區塊不渲染，頁面空白。

#### 路由模式：*action 通配符 vs :id 參數

```go
// ❌ 錯誤：:id 不支援 PbootCMS 風格的狀態切換 URL
adminGroup.GET("/admin/xxx/mod/:id", ctrl.Mod)

// ✅ 正確：*action 支援 /mod/id/123/field/status/value/0 風格
adminGroup.Any("/admin/xxx/mod/*action", ctrl.Mod)
```

**判斷規則**：如果模板中有狀態切換圖標鏈接（`/mod/id/[value->id]/field/status/value/0`），必須用 `*action`。

#### Mod 方法解析 *action 參數

```go
func (ctrl *Controller) Mod(c *gin.Context) {
    params := helper.ParseWildcardAction(c.Param("action"))
    idStr := params["id"]       // /id/123 → "123"
    field := params["field"]    // /field/status → "status"
    value := params["value"]    // /value/0 → "0"

    // 單欄位切換（狀態開關）
    if field != "" && value != "" {
        model.DB.Model(&Model{}).Where("id = ?", id).Update(field, value)
        c.Redirect(302, "/admin/xxx/index")
        return
    }
    // ... 完整修改邏輯
}
```

#### 雙 MD5 密碼雜湊（兩行寫法，不要嵌套）

```go
// ❌ 錯誤：嵌套括號導致編譯失敗
encPwd := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x", md5.Sum([]byte(password)))))

// ✅ 正確：兩行寫法
firstMd5 := fmt.Sprintf("%x", md5.Sum([]byte(password)))
encPwd := fmt.Sprintf("%x", md5.Sum([]byte(firstMd5)))
```

#### time.Time 欄位的模板顯示

```go
// 問題：pongo2 直接渲染 time.Time 會顯示時區和納秒
// 解決：在 Model 中加非 DB 欄位，控制器中預格式化

type Member struct {
    // ... DB 欄位 ...
    RegisterTimeStr string `gorm:"-" json:"register_time_str"` // 非DB，顯示用
}

// 控制器中
if !member.RegisterTime.IsZero() {
    member.RegisterTimeStr = member.RegisterTime.Format("2006-01-02 15:04:05")
}

// 模板中
<td>[value->register_time_str]</td>  <!-- 不用 [value->register_time] -->
```

#### 通知消息常量（apps/common/notice.go）

```go
// ✅ 正確：使用常量
mg.JSONOKMsg(c, common.NoticeAdd)
mg.JSONOKMsg(c, common.NoticeModify)
mg.JSONOKMsg(c, common.NoticeDelete)

// ❌ 錯誤：硬編碼字符串
mg.JSONOKMsg(c, "新增成功")
```

### 1.3 反模式清單（曾犯過的錯誤）

| # | 錯誤行為 | 正確做法 | 出現頻率 |
|---|---------|---------|---------|
| 1 | 不讀 docs 直接寫代碼 | 先讀本文件再動手 | 高 |
| 2 | 不讀同類控制器參考 | Grep 一個已實現的控制器（如 ContentSort） | 高 |
| 3 | 猜測方法名（如 JSONErrMsg） | Grep 確認方法存在 | 中 |
| 4 | 猜測模型欄位名 | 讀 Model struct 定義 | 中 |
| 5 | 用 `:id` 路由而非 `*action` | 檢查模板是否有狀態切換鏈接 | 中 |
| 6 | 嵌套 md5.Sum 導致編譯失敗 | 兩行寫法 | 低 |
| 7 | 直接渲染 time.Time | 加 `gorm:"-"` 字串欄位預格式化 | 低 |
| 8 | 硬編碼通知消息 | 引用 notice.go 常量 | 低 |
| 9 | 用簡體中文寫代碼/模板 | 全部繁體化（`/修改繁體文本`） | 中 |
| 10 | 忘記傳遞 `list`/`mod` 標誌 | Render 時必須傳 `gin.H{"list": true}` | 高 |
| 11 | 狀態切換鏈接缺少 `class="switch"` | 必須加 `class="switch"`（comm.js 用此選擇器攔截 AJAX） | 高 |
| 12 | 前台默認頭像路徑不一致 | 統一用 `/static/admin/images/logo.png`（與留言板一致） | 中 |
| 13 | 前台缺少欄目/內容瀏覽權限檢查 | 必須呼叫 `checkSortPermission`/`checkContentPermission` | 高 |
| 14 | SetSession 多次呼叫創建不同 session ID | 用 `c.Set("sessionID", sid)` 復用同一請求內的 ID | 高 |
| 15 | backurl 未 URL-encode | 必須用 `url.QueryEscape`，否則含查詢參數的 URL 會被截斷 | 中 |
| 16 | backurl 開放重定向風險 | 必須用 `isSafeRedirectURL` 驗證為相對路徑 | 高 |
| 17 | 標籤缺少 checkLabelLevel 屬性檢查 | pair 標籤的 showlogin/hidelogin/showgcode 等屬性會失效 | 高 |
| 18 | 缺少 `{pboot:mustlogin}` 整頁強制登入 | 需要會員才能看的頁面對未登入訪客暴露 | 高 |
| 19 | `{user:uid}` 不存在 | 模板無法取得會員 ID | 低 |
| 20 | backurl 在 POST body 中用 `c.Query` 讀取 | POST 請求的 backurl 要用 `c.DefaultPostForm` 讀取 | 中 |
| 21 | 路由大小寫不匹配導致 404 | 見下方「路由大小寫陷阱」 | 高 |
| 22 | ajaxlink 返回格式用 `msg` 而非 `data` | `comm.js` 的 ajaxlink 讀 `response.data`，必須用 `JSONOK(c, msg)` 返回 | 高 |
| 23 | pongo2 `{foreach $value->Field(key,value)}` 不支援 | `reForeach` 正則只匹配 `$varName`，不支援 `->`；必須在 controller 預計算為字串 | 高 |
| 24 | layui `form.on('submit()')` 攔截非 `lay-submit` 按鈕導致批量操作失敗 | 改用 jQuery `$(document).on('submit')` 統一攔截 | 高 |
| 25 | ExtField 同模型下 field 名稱重複導致數據覆蓋 | `ay_content_ext` 每列對應一個 field 名，重複會共用同一列；必須用 `CheckFieldUnique` 檢查 | 高 |
| 26 | 用戶輸入未過濾直接存入 DB | 必須經過 `common.FilterUserInput()`（XSS 防護） | 高 |
| 27 | 動態 SQL 表名/欄位名未驗證 | 必須用 `CheckVarType()` / `CheckColumnName()` 白名單驗證 | 高 |
| 28 | 會員登入未重新生成 Session ID | 必須呼叫 `common.RegenerateSessionID(c)`（防 Session Fixation） | 高 |
| 29 | AJAX 攔截回應只有 `msg` 缺少 `data` | `auth.go` 中 AJAX 回應必須同時包含 `data` + `msg` | 高 |
| 30 | 刪除操作用 GET 請求 | 刪除操作必須用 POST（防 CSRF） | 高 |
| 31 | 通知文案帶感嘆號 | 統一風格，通知文案**不加感嘆號** | 中 |

### 1.4 路由大小寫陷阱（Gin + 模板引擎）

**Gin 路由是大小寫敏感的**，但模板引擎的 `{url.路徑}` 解析會把路徑轉成**全小寫**
（`view.go` 的 `strings.ToLower`）。兩者不匹配時導致 404。

```
模板寫：{url./admin/deleCache/index}
解析為：/admin/delecache/index   ← 全小寫
路由寫：adminGroup.GET("/content/deleCache/index", ...)  ← 大寫 C
結果：404（大小寫不匹配）
```

**規則**：
1. 模板中 `{url.路徑}` 的路徑會被轉全小寫
2. `route.go` 中註冊的路由路徑也必須用全小寫，或同時註冊小寫別名
3. 若需保留大小寫路徑（如兼容舊連結），同時註冊兩個：

```go
// 原始路徑（保留大小寫）
adminGroup.GET("/content/deleCache/index", dc.Index)
// 別名（全小寫，匹配模板引擎輸出）
adminGroup.GET("/delecache/index", dc.Index)
```

### 1.5 後台狀態切換速查（class="switch"）

後台模板的狀態切換圖標**必須**加 `class="switch"`，否則 `comm.js` 無法攔截點擊事件，
瀏覽器會直接訪問 URL 並顯示 JSON 文本。

```html
<!-- ❌ 錯誤：缺少 class="switch"，點擊後顯示 {"code":1,"msg":"修改成功"} -->
<a href="/admin/xxx/mod/id/[value->id]/field/status/value/0"><i class='fa fa-toggle-on'></i></a>

<!-- ✅ 正確：comm.js 用 $('.switch') 攔截，發 AJAX 請求，切換圖標，不跳轉 -->
<a href="/admin/xxx/mod/id/[value->id]/field/status/value/0" class="switch"><i class='fa fa-toggle-on'></i></a>
```

**原理**：`static/admin/js/comm.js` 中 `$('.switch').on("click", ".fa-toggle-on", ...)` 用 `$.get()` 發送請求，然後修改 DOM 切換圖標狀態，`return false` 阻止跳轉。所有後台模板的狀態切換鏈接（status/required/istop/isrecommend 等）都必須加此 class。

### 1.6 前台默認頭像速查

前台所有用戶頭像為空時，統一使用 `/static/admin/images/logo.png`（與留言板 provider 一致）：

```go
// ✅ 正確
headpic := c.Headpic
if headpic == "" {
    headpic = "/static/admin/images/logo.png"
}

// ❌ 錯誤：路徑不一致
headpic = "/static/images/logo.png"
```

### 1.7 模板引擎速查（不要混淆）

| 場景 | 引擎 | 語法 | 轉換器 |
|------|------|------|--------|
| 後台 admin view | pongo2 | `{if([$list])}` → `{% if List %}` | `core/basic/view.go` convertPbootToPongo2() |
| 前台 template/default | 自研 TagParser | `{gboot:xxx}` | `apps/common/parser/tags.go` |

> **關鍵區別**：後台模板用 `{$var->field}` 語法（pongo2 轉譯），前台模板用 `{gboot:xxx}` 和 `[prefix:field]` 語法。

#### pongo2 foreach 限制（重要）

pongo2 的 `reForeach` 正則只匹配 `{foreach $varName(key,value)}`，其中 `varName` 只允許 `[\w_]+`，**不支援 `->` 語法**。

```html
<!-- ❌ 錯誤：reForeach 不匹配 $value->Scodes，導致 {/foreach} 變成孤立 endfor -->
{foreach $value->Scodes(skey,sval)}
    [sval]
{/foreach}

<!-- ✅ 正確：在 controller 中預計算為字串，模板直接輸出 -->
<!-- controller: listFields[i]["ScodeDisplay"] = "├ 行業動態、├ 公司動態" -->
<td>[value->ScodeDisplay]</td>
```

**規則**：如果需要在模板中遍歷 struct 的某個欄位（如陣列），改為在 controller 中預計算為顯示字串。

### 1.8 ExtField 擴展字段速查

#### scode 適用欄目功能

`ay_extfield` 表的 `scode` 列存儲適用欄目代碼（逗號分隔，空=全展示）：

```go
// 過濾邏輯：scode 為空匹配所有欄目，否則檢查目標欄目是否在列表中
content.ScodeMatches(fieldScode, targetScode)

// 多選存儲：scode = "3,4"（逗號分隔）
// 全展示：scode = ""
```

#### 選項規範化

```go
// NormalizeOptions 將換行符轉為逗號，支援回車或逗號分隔選項
options := content.NormalizeOptions(c.PostForm("value"))
// 輸入 "選項A\n選項B\n選項C" → 輸出 "選項A,選項B,選項C"
```

#### field 唯一性檢查

```go
// 同一模型下 field 名稱必須唯一（共用 ay_content_ext 同一物理列）
if content.CheckFieldUnique(mcode, field, excludeID) {
    ef.JSONFail(c, "字段名稱在該模型下已存在")
    return
}
```

#### layui form.on('submit()') 衝突

```javascript
// ❌ 錯誤：form.on('submit()') 會攔截 lay-submit 按鈕，但批量操作按鈕（複製/移動/刪除）沒有 lay-submit
form.on('submit()', function(data){ ... });

// ✅ 正確：用 jQuery 統一攔截所有表單提交（mylayui.js）
$(document).on('submit', 'form:not(#dologin)', function(e) {
    // 用 button._clicked 識別點擊的提交按鈕
    var $btn = $form.find('button._clicked');
    // ...
});
```

### 1.9 會員系統速查

#### Session 鍵名（前台會員）

| 鍵名 | 類型 | 用途 |
|------|------|------|
| `pboot_uid` | int | 會員 ID |
| `pboot_ucode` | string | 會員編號 |
| `pboot_username` | string | 用戶名 |
| `pboot_useremail` | string | 郵箱 |
| `pboot_usermobile` | string | 手機 |
| `pboot_gid` | string | 等級 ID |
| `pboot_gcode` | string | 等級編號 |
| `pboot_gname` | string | 等級名稱 |

#### 會員模型關鍵欄位

| 欄位 | DB 列名 | 類型 | 備註 |
|------|---------|------|------|
| GID | `gid` | string | 存儲等級 ID（數字字串） |
| Useremail | `useremail` | string | 不是 `email` |
| Usermobile | `usermobile` | string | 不是 `mobile` |
| LoginCount | `login_count` | int | 不是 `logincount` |
| LastLoginIP | `last_login_ip` | string | 不是 `lastloginip` |
| LastLoginTime | `last_login_time` | string | 字串類型，非 time.Time |
| RegisterTime | `register_time` | time.Time | 顯示需預格式化 |
| Headpic | `headpic` | string | 小寫 p，不是 HeadPic |
| Gname | — | string | `gorm:"-"` 非DB欄位，JOIN 顯示用 |

#### 會員等級模型關鍵欄位

| 欄位 | DB 列名 | 備註 |
|------|---------|------|
| Gcode | `gcode` | 不是 `code` |
| Gname | `gname` | 不是 `name` |
| Lscore | `lscore` | 積分下限 |
| Uscore | `uscore` | 積分上限 |

### 1.9 欄目/內容瀏覽權限速查

前台所有渲染方法（ListPage、ContentPage、SortByScode、ContentByID、renderSortPage）都**必須**呼叫權限檢查：

```go
// 欄目權限檢查
if !fc.checkSortPermission(c, &sort) {
    return // checkPageLevel 已寫入 response（跳轉登入或顯示提示）
}

// 內容權限檢查（ContentPage 和 ContentByID 需同時檢查欄目+內容兩層）
if !fc.checkContentPermission(c, &ct) {
    return
}
```

**權限模型**：
- `ay_content_sort.gid` / `ay_content.gid` → 指向 `ay_member_group.id`
- `ay_member_group.gcode` → 等級編號（如 1=初級, 2=中級, 3=高級）
- `gtype` → 比較運算子（1小於/2小於等於/3等於/4大於等於/5大於，預設4）
- session `pboot_gcode` → 訪客的等級編號（int 類型）

**deny 邏輯**：gtype 決定比較方式，deny=true 時：
- 已登入（uid>0）→ 顯示 gnote 提示文字
- 未登入 → 302 跳轉 `/login?backurl=<當前URL>`

### 1.10 Session 速查（SetSession 復用機制）

**關鍵**：同一請求內多次呼叫 `SetSession` 時，必須復用同一個 session ID。`getSessionID` 優先從 `c.Get("sessionID")` 取，避免每次建立新 ID 導致資料分散：

```go
func SetSession(c *gin.Context, key string, value interface{}) {
    sid := getSessionID(c)
    if sid == "" {
        sid = createSessionID()
        c.SetCookie("PbootGo", sid, ...)
        c.Set("sessionID", sid) // 存入 context，後續同請求內復用
    }
    // ...
}
```

**注意**：`pboot_gcode` 存入 session 時必須轉為 `int`（`GetSessionInt` 讀取）。

### 1.11 checkLabelLevel 速查（標籤屬性權限）

所有 pair 標籤（`{gboot:list}`、`{gboot:nav}`、`{gboot:slide}` 等）支援 14 種權限屬性，
在 `processPairTags` 中統一檢查，權限不足時整個區塊回傳空字串：

| 屬性 | 說明 | 範例 |
|------|------|------|
| `showlogin=1` | 登入後顯示 | `{gboot:list scode=1 showlogin=1}` |
| `hidelogin=1` | 登入後隱藏 | `{gboot:nav hidelogin=1}` |
| `showgcode=1,2` | 指定等級顯示（逗號分隔） | `{gboot:list showgcode=2,3}` |
| `hidegcode=3` | 指定等級隱藏 | `{gboot:nav hidegcode=1}` |
| `showucode=U001` | 指定用戶顯示 | `{gboot:list showucode=U001,U002}` |
| `hideucode=U001` | 指定用戶隱藏 | `{gboot:nav hideucode=U001}` |
| `showgcodelt=3` | 等級<3顯示 | `{gboot:list showgcodelt=3}` |
| `showgcodegt=1` | 等級>1顯示 | `{gboot:list showgcodegt=1}` |
| `showgcodele=3` | 等級<=3顯示 | `{gboot:list showgcodele=3}` |
| `showgcodege=1` | 等級>=1顯示 | `{gboot:list showgcodege=1}` |
| `hidegcodelt/gt/le/ge` | 等級比較隱藏 | 同上 |

### 1.12 `{pboot:mustlogin}` 速查（整頁強制登入）

模板中含 `{gboot:mustlogin}` 或 `{pboot:mustlogin}` 時，未登入訪客自動跳轉登入頁：

```html
<!-- 在模板頂部加此標籤，未登入訪客會被跳轉到 /login?backurl=當前URL -->
{gboot:mustlogin}
```

在所有渲染方法中，`checkMustLogin` 在 `p.Render` 之前檢查。登入後標籤本身回傳空字串（不影響渲染）。

### 1.13 安全防護速查（XSS / SQL注入 / Session / CSRF）

> 對齊 PbootCMS PHP `escape_string()` + `filter()` + `checkKey()`，使用 Go stdlib 等價方案。

#### XSS 防護

```go
import "gbootcms/apps/common"

// 用戶輸入經過 FilterUserInput 處理（對齊 PbootCMS filter()）
// 1. 清除 hex 括號 x3c/x3e
// 2. 過濾 {gboot:if} / {gboot:sql} 標籤
// 3. HTML 轉義（html.EscapeString = PHP htmlspecialchars(ENT_QUOTES, UTF-8)）
comment := common.FilterUserInput(c.PostForm("comment"))
```

> **規則**：所有用戶提交的富文字內容（評論、留言）必須經過 `FilterUserInput`。

#### SQL 注入防護

```go
// 動態表名/欄位名必須用白名單正則驗證（對齊 PbootCMS checkKey()）
if !common.CheckVarType(tableName) {
    return fmt.Errorf("非法表名")
}
if !common.CheckColumnName(fieldName) {
    return fmt.Errorf("非法欄位名")
}
// GORM 參數化查詢已防護值注入，只需驗證標識符
db.Where("id = ?", id).Find(&items)
```

| 函數 | 正則 | 用途 |
|------|------|------|
| `CheckIdentifier` | `^[\w\.\-]+$` | 通用 SQL 識別符 |
| `CheckVarType` | `^[\w\-\.]+$` | 表名驗證（對齊 PbootCMS var 類型） |
| `CheckColumnName` | `^[a-zA-Z][\w]+$` | 欄位名驗證（必須字母開頭） |

#### Session 安全

| 防護項 | 實現 | 位置 |
|--------|------|------|
| Session Fixation | 會員登入時 `RegenerateSessionID(c)` | `member.go` Login |
| Cookie HttpOnly | `HttpOnly: true`（防 XSS 竊取） | `session.go` SetCookie |
| Session TTL | 24h 最大生命週期 + 2h 閒置超時 + 10min 清理週期 | `session.go` sessionEntry |
| 過期清理 | `cleanupExpiredSessions()` 後台協程 | `session.go` init() |

#### CSRF 防護

| 防護項 | 實現 | 位置 |
|--------|------|------|
| 後台表單 | 32字節 `crypto/rand` token + POST 中間件驗證 | `auth.go` + `formcheck` |
| 評論刪除 | 僅允許 `POST`（GET 返回 404） | `main.go` 路由 + `comment.go` |
| 上傳豁免 | `csrfExemptPaths` 白名單 | `auth.go` |

#### 權限控制

| 防護項 | 實現 | 位置 |
|--------|------|------|
| clearsession 權限 | **不在** `publicPermPaths` 中，需特定權限 | `auth.go` |
| 非標準動作 | 附屬於控制器 `index` 瀏覽權限 | `auth.go` |
| 評論刪除 | 驗證 `uid` 所有權（`WHERE id=? AND uid=?`） | `comment.go` Del |
| 密碼儲存 | bcrypt + 雙 MD5 向後相容 + 登入自動升級 | `common` HashPassword |

#### AJAX 回應格式規則

| 場景 | 回應方式 | 前端讀取 | 顯示方式 |
|------|---------|---------|---------|
| ajaxlink 連結 | `JSONOK(c, data)` | `response.data` | `layer.msg()` |
| 表單提交 | `JSONOKMsg` / `JSONFail` | `res.msg` | `showNotify()` |
| 權限攔截 | 雙欄位 `data` + `msg` | 視場景 | 視場景 |

> **規則**：`auth.go` 中的 AJAX 攔截回應必須同時包含 `data` 和 `msg` 欄位，通知文案**不加感嘆號**。

---

## 二、核心原則 / Core Principles

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

---

## 三、驗證流程 / Verification Workflow

修改代碼後的標準驗證順序:

1. **`git status`** — 確認修改的文件清單
2. **`go build`** — 編譯通過
3. **重啟服務** — 關閉舊進程 → 啟動新進程
4. **HTTP 探活** — `Invoke-WebRequest -Uri 'http://localhost:8080/' -UseBasicParsing`
5. **業務路徑** — 至少驗證 1 個核心 API 正常

---

## 四、項目結構速查 / Project Structure

| 路徑 | 用途 | 修改限制 |
|---|---|---|
| `apps/admin/` | 後台管理 API | 自由修改 |
| `apps/home/` | 前台渲染 + 會員系統 | 自由修改 |
| `apps/common/parser/` | PbootCMS 標籤解析 | 自由修改 |
| `core/basic/view.go` | PHP 模板轉譯器 | 自由修改 |
| `template/default/` | 前台模板 | 自由修改 |
| `apps/admin/view/` | 後台模板 | 自由修改 |
| `config/config.json` | 配置文件 | 自由修改 |
| **`data/pbootcms.db`** | **業務資料庫** | **🔒 僅 UI 寫入** |

### 技術棧速查

| 層次 | 選型 | 說明 |
|------|------|------|
| 語言 | Go 1.25 | 單二進制部署，無需 CGO |
| Web 框架 | Gin v1.12 | 路由、中間件、請求處理 |
| ORM | GORM v1.31 | AutoMigrate，`ay_` 前綴 |
| 數據庫 | SQLite (glebarez 純 Go) | 無需 CGO/GCC |
| **後台模板** | **Pongo2 v6.1** | Django 風格 + PbootCMS 語法轉換器 |
| **前台模板** | **自研 TagParser** | `{gboot:xxx}` 標籤 + fsnotify 熱重載 |
| 後台 UI | Layui 2.5.4 + jQuery | 與 PbootCMS 原版一致 |
| 前台 UI | Bootstrap 4 + Swiper 4 | 前台模板自帶 |

---

## 五、邊界案例 / Edge Cases

### Q: 我需要測試某個功能,但 db 中沒有對應資料怎麼辦?
A: **提示用戶手動通過後台新增測試資料**,不要在腳本中 insert 模擬資料。

### Q: 我懷疑 db 結構有 bug,可以 migrate 嗎?
A: **不可以**。如果發現結構問題,**先在 issues 記錄,不要自動修復**。表結構是用戶手工管理。

### Q: 訪問計數導致 db 變化,需要 `git checkout` 還原嗎?
A: **不需要**。訪問計數是透明副作用。但如果你寫了腳本,可能污染了其他欄位,
此時可以用 `git checkout -- data/pbootcms.db` 還原到入庫版本(此版本是「測試基準線」)。

### Q: 我能用 `go run` 寫一個遷移腳本嗎?
A: **不要**。如果資料庫需要任何變更,必須由用戶親自確認並執行。

---

## 六、確認清單 / Checklist

每次任務結束前:
- [ ] `git status` 確認沒有意外修改 db
- [ ] `git diff --stat` 確認變更範圍合理
- [ ] 所有修改的文件在允許列表中
- [ ] 沒有執行 `rm`、`drop`、`truncate`、`delete from` 等危險命令
- [ ] 用戶確認滿意後再 git commit

---

## 七、相關文檔 / Related Docs

- `pbootcms-go-dev-guide.md` — 完整開發技術文檔（含防遺忘清單詳細版、會員系統、開發指南）
- `.plan.md` — 開發進度活筆記
- `build-run.bat` — 構建腳本
