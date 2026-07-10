# Gbootcms 開發技術文檔（整合版）

> 本文檔供新人接手參考，整合自「留言系統前基線版」與代碼實勘，已逐項核對 go.mod、config、session、controller、parser 等源碼。
> 對應源碼版本：基於 PbootCMS 3.2.12 PHP → Go 移植版
> 最後更新：2026-07-08

---

## 零、防遺忘清單（必讀）

> 以下問題在開發過程中反覆出現，每次修改代碼前務必對照此清單。

### 0.1 寫代碼前的強制檢查流程

```
1. 讀 project_memory.md — 檢查硬約束（如 mcode 而非 modelcode）
2. 讀本文檔第零章 — 檢查 API 速查表和反模式
3. Grep 一個已實現的同類控制器 — 參考其模式
4. 確認方法/函數存在 — Grep 函數名，不要猜測
5. 確認模型欄位名 — 讀 Model struct，不要猜測
```

### 0.2 API 速查表（不要猜，查這裡）

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

### 0.3 反模式清單（曾犯過的錯誤）

| # | 錯誤行為 | 正確做法 | 出現頻率 |
|---|---------|---------|---------|
| 1 | 不讀 docs 直接寫代碼 | 先讀 AI_GUIDELINES.md 和本文件 | 高 |
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
| 21 | pongo2 `{foreach $value->Field(key,value)}` 不支援 | `reForeach` 正則只匹配 `$varName`，不支援 `->`；在 controller 預計算為字串 | 高 |
| 22 | layui `form.on('submit()')` 攔截非 `lay-submit` 按鈕 | 改用 jQuery `$(document).on('submit')` 統一攔截 | 高 |
| 23 | ExtField 同模型 field 名稱重複導致數據覆蓋 | `ay_content_ext` 每列對應一個 field 名，必須用 `CheckFieldUnique` 檢查 | 高 |
| 24 | `c.Query()` 在 IndexCatchAll 後返回舊值 | `IndexCatchAll` 修改 `RawQuery` 後 gin 緩存失效，改用 `c.Request.URL.Query().Get()` | 高 |

### 0.4 後台狀態切換速查（class="switch"）

後台模板的狀態切換圖標**必須**加 `class="switch"`，否則 `comm.js` 無法攔截點擊事件，
瀏覽器會直接訪問 URL 並顯示 JSON 文本 `{"code":1,"msg":"修改成功"}`。

```html
<!-- ❌ 錯誤：缺少 class="switch"，點擊後瀏覽器直接顯示 JSON -->
<a href="/admin/xxx/mod/id/[value->id]/field/status/value/0"><i class='fa fa-toggle-on'></i></a>

<!-- ✅ 正確：comm.js 用 $('.switch') 攔截，發 AJAX 請求，切換圖標，不跳轉 -->
<a href="/admin/xxx/mod/id/[value->id]/field/status/value/0" class="switch"><i class='fa fa-toggle-on'></i></a>
```

**原理**：`static/admin/js/comm.js` 中 `$('.switch').on("click", ".fa-toggle-on", ...)` 用 `$.get()` 發送請求，然後修改 DOM 切換圖標狀態，`return false` 阻止跳轉。所有後台模板的狀態切換鏈接（status/required/istop/isrecommend 等）都必須加此 class。

### 0.5 前台默認頭像速查

前台所有用戶頭像為空時，統一使用 `/static/admin/images/logo.png`（與留言板 message provider 一致）：

```go
// ✅ 正確
headpic := c.Headpic
if headpic == "" {
    headpic = "/static/admin/images/logo.png"
}

// ❌ 錯誤：路徑不一致
headpic = "/static/images/logo.png"
```

### 0.6 模板引擎速查（不要混淆）

| 場景 | 引擎 | 語法 | 轉換器 |
|------|------|------|--------|
| 後台 admin view | pongo2 | `{if([$list])}` → `{% if List %}` | `core/basic/view.go` convertPbootToPongo2() |
| 前台 template/default | 自研 TagParser | `{gboot:xxx}` | `apps/common/parser/tags.go` |

> **關鍵區別**：後台模板用 `{$var->field}` 語法（pongo2 轉譯），前台模板用 `{gboot:xxx}` 和 `[prefix:field]` 語法。

### 0.7 會員系統速查

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

### 0.8 欄目/內容瀏覽權限速查

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

### 0.9 Session 速查（SetSession 復用機制）

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

### 0.10 checkLabelLevel 速查（標籤屬性權限）

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

### 0.11 `{pboot:mustlogin}` 速查（整頁強制登入）

模板中含 `{gboot:mustlogin}` 或 `{pboot:mustlogin}` 時，未登入訪客自動跳轉登入頁：

```html
<!-- 在模板頂部加此標籤，未登入訪客會被跳轉到 /login?backurl=當前URL -->
{gboot:mustlogin}
```

在所有渲染方法中，`checkMustLogin` 在 `p.Render` 之前檢查。登入後標籤本身回傳空字串（不影響渲染）。

### 0.12 backurl 安全速查

| 問題 | 修復 | 位置 |
|------|------|------|
| URL-encode | `url.QueryEscape(currentURL)` | front.go checkPageLevel |
| 開放重定向 | `isSafeRedirectURL(tourl)` 驗證相對路徑 | member.go Login |
| POST body 讀取 | `c.DefaultPostForm("backurl", c.Query("backurl"))` | member.go Login |

---

## 一、項目概述

Gbootcms 是將 PHP 版 PbootCMS 3.2.12 忠實移植為 Go 語言的企業網站管理系統。項目保留了原版的數據庫表結構（`ay_` 前綴）、模板語法、URL 路由規則和後台 UI（Layui），用 Go 技術棧替換了 PHP 後端。

### 技術棧一覽

| 層次 | 技術 | 版本 | 說明 |
|------|------|------|------|
| 語言 | Go | 1.25.0 | 單二進制部署 |
| Web 框架 | Gin | v1.12.0 | 路由分發、中間件、請求處理 |
| ORM | GORM | v1.31.1 | AutoMigrate，`ay_` 前綴策略 |
| 數據庫驅動 | glebarez/sqlite | v1.11.0 | **純 Go 驅動，無需 CGO**，文件 `data/pbootcms.db` |
| 後台模板 | Pongo2 | v6.1.0 | Django 風格模板 + 自研 PbootCMS 語法轉換器 |
| 前台模板 | 自研 TagParser | — | `{gboot:xxx}` 標籤語法 + fsnotify 熱重載 |
| 模板熱更新 | fsnotify | v1.10.1 | 前台模板文件變更自動重載 |
| 郵件 | go-mail | v0.7.3 | SMTP 發信，配置存 `ay_config` |
| 配置管理 | Viper | v1.21.0 | 環境變數支援，前綴 `PBOOTCMS_GO_` |
| 結構化日誌 | log/slog | (stdlib) | Go 1.21+ 標準庫，`apps/common/logger.go` |
| 前端 UI（後台） | Layui 2.5.4 + jQuery 1.12.4 | — | 與 PbootCMS 原版一致 |
| 前端 UI（前台） | Bootstrap 4 + Swiper 4 | — | 前台模板自帶 |

### 核心設計原則

1. **數據庫零改動**：所有表結構、字段名、表前綴 `ay_` 與 PbootCMS PHP 版完全一致，可直接遷移數據
2. **模板語法兼容**：後台保留 PbootCMS 原版 PHP 模板語法（`{foreach}`、`{if}`、`{$var->field}`），前台使用 `{gboot:xxx}` 標籤
3. **URL 100% 兼容**：通過路由重寫和 NoRoute 兜底，實現原版 URL 格式的完整支持

### 與原版的關鍵命名差異

前台模板標籤前綴由 PHP 版的 `{pboot:xxx}` 改為 Go 版的 `{gboot:xxx}`（**g**o-boot），但語義與屬性與原版一致。後台模板仍沿用 pongo2 的 `{{ }}` 語法。這是新人最容易混淆的一點：**前台走自研 `gboot` 解析器，後台走 pongo2**。

---

## 二、目錄結構與 PHP 映射

```
pbootcms-go/
├── main.go                          # 程序入口，啟動流程，路由註冊
├── go.mod / go.sum                  # Go 模組定義
├── build-run.bat                    # 編譯腳本
├── config/
│   ├── config.go                    # 配置結構體與載入邏輯（sync.Once 單例）
│   └── config.json                  # 配置文件（端口、資料庫路徑等）
├── core/
│   ├── db/db.go                     # GORM + glebarez/sqlite 資料庫初始化
│   ├── basic/view.go                # 後台模板引擎（pongo2 + PHP 語法轉換）
│   └── mediaplugin/plugin.go        # GORM 媒體快取失效插件
├── apps/
│   ├── route/route.go               # 後台路由集中註冊
│   ├── common/
│   │   ├── BaseController.go        # 基礎控制器（JSON 回應、批量排序等）
│   │   ├── Render.go                # 後台模板渲染入口（AppThemeDir 定義）
│   │   ├── session.go               # 自實現記憶體 Session（PbootGo Cookie）
│   │   ├── notice.go                # 通知訊息常量（硬約束：禁止硬編碼）
│   │   ├── captcha.go               # 驗證碼生成與校驗
│   │   ├── middleware/
│   │   │   ├── auth.go              # 後台認證中間件
│   │   │   ├── gzip.go              # Gzip/Brotli 壓縮中間件
│   │   │   ├── html_cache.go        # HTML 快取中間件
│   │   │   ├── ip_filter.go         # IP 黑白名單中間件
│   │   │   ├── path_rewrite.go      # PbootCMS URL 重寫映射表
│   │   │   ├── redirect.go          # HTTPS 跳轉與主域名跳轉中間件
│   │   │   ├── site_status.go       # 關站檢查中間件
│   │   │   └── spider_log.go        # 蜘蛛訪問日誌中間件
│   │   └── parser/                  # 前台模板標籤解析引擎
│   │       ├── tags.go              # TagParser 核心解析管線
│   │       ├── engine.go            # TemplateStore 模板存儲與熱重載
│   │       ├── providers.go         # 所有標籤 Provider 註冊
│   │       ├── if_eval.go           # 條件表達式求值
│   │       └── convert.go           # 輔助解析函數
│   ├── admin/
│   │   ├── controller/              # 後台控制器
│   │   │   ├── IndexController.go   # 登入/首頁/上傳/驗證碼
│   │   │   ├── content/             # 內容管理（15個控制器）
│   │   │   ├── system/              # 系統管理（10個控制器）
│   │   │   └── member/              # 會員管理（4個控制器）
│   │   ├── model/                   # 數據模型
│   │   │   ├── db.go                # 全域 DB 實例 + 類型別名
│   │   │   ├── content/             # 內容模型
│   │   │   ├── system/              # 系統模型
│   │   │   ├── member/              # 會員模型
│   │   │   └── seed/seed.go         # 種子資料初始化
│   │   ├── service/content/         # 業務服務層（MVC+S）
│   │   ├── helper/                  # 模板輔助函數
│   │   └── view/                    # 後台 HTML 模板
│   └── home/
│       └── controller/
│           ├── front.go             # 前台控制器（FrontController）
│           ├── member.go            # 會員前台控制器（登入/註冊/中心）
│           └── comment.go           # 評論控制器（CommentController：Add/My/Del）
├── template/default/                # 前台模板目錄
│   ├── comm/                        # 公共模板（head/foot/page等）
│   ├── member/                      # 會員前台模板（login/register/ucenter/umodify）
│   ├── index.html, list.html, content.html, search.html ...
│   └── static/                      # 前台靜態資源
├── static/                          # 全域公共資源
│   ├── admin/                       # 後台 CSS/JS/圖片/Layui/font-awesome
│   ├── upload/                      # 用戶上傳文件（gitignore）
│   └── images/                      # 全域圖片
├── data/pbootcms.db                 # SQLite 資料庫文件
├── runtime/                         # 執行時快取/除錯產物
├── bin/                             # 編譯產物
└── docs/                            # 文檔
```

---

## 三、啟動流程

`main.go` 的 `main()` 函數按以下固定順序執行：

1. `config.Load("config/config.json")` — 載入配置
2. `model.InitDB(cfg)` — 初始化資料庫
3. `system.AutoMigrate()` → `content.AutoMigrate()` → `member.AutoMigrate()` — 自動遷移
4. `seed.Init()` — 種子資料初始化（冪等）
5. `basic.InitViewEngine(...)` — 初始化後台模板引擎（pongo2）
6. `parser.NewTemplateStore(...)` — 初始化前台模板引擎
7. `gin.Default()` + 中間件 + 路由註冊 — 建立 Gin 引擎
8. `r.Run(":8080")` — 啟動 HTTP 服務

### 種子資料

`seed.Init()` → `SeedData()` 邏輯：
1. 每次啟動冪等呼叫 `content.EnsureContentExtTable()` 確保 `ay_content_ext` 存在
2. 檢查 `ay_user` 是否非空：非空則跳過首次種子，但仍調 `ensureMenuVersion()` + `ensureMemberConfigs()`
3. `ensureMenuVersion()`：以 `mcode='M1007'` 為版本標誌，缺失則重建
4. `ensureMemberConfigs()`：確保 17 項會員配置存在（用於已有資料庫的版本升級）
5. 首次種子：管理員 admin/admin、站點信息、36 條菜單、超級管理員角色、2 個會員組、2 個內容模型、29+ 項系統配置

---

## 四至十六章

> 第四章至第十六章的內容與之前版本一致，包含：配置系統、資料庫層、路由系統、後台管理系統、後台模板引擎、前台模板引擎與標籤系統、前台路由與模板映射、擴展子系統、開發約束與規範、開發指南、開發環境與構建、已實現功能清單、關鍵文件索引。
>
> 詳細內容待補充，亦可參見 `docs/AI_GUIDELINES.md`。

---

## 十七、會員系統（2026-07-02 新增）

### 階段 0：模型修復（已完成）

四個會員模型全部對齊 PbootCMS 原版資料庫：

| 模型 | 表名 | 關鍵修復 |
|------|------|---------|
| Member | `ay_member` | 24 欄位完整對齊（useremail/usermobile/login_count 等） |
| MemberGroup | `ay_member_group` | Gcode/Gname/Lscore/Uscore/Description |
| MemberField | `ay_member_field` | Name/Length/Required/Description/Sorting |
| MemberComment | `ay_member_comment` | Comment 欄位名 + CommentView JOIN 結構體 |

### 階段 1：前台會員系統（已完成）

#### 前台路由

| 方法 | 路徑 | 控制器方法 | 說明 |
|------|------|-----------|------|
| GET/POST | `/login` | `fc.Login` | 會員登入 |
| GET/POST | `/register` | `fc.Register` | 會員註冊 |
| GET | `/logout` | `fc.Logout` | 會員登出 |
| GET | `/ucenter` | `fc.Ucenter` | 個人中心 |
| GET/POST | `/umodify` | `fc.Umodify` | 資料修改 |

#### 前台模板

```
template/default/member/
├── left.html      # 側邊欄導航
├── login.html     # AJAX 登入表單
├── register.html  # AJAX 註冊表單
├── ucenter.html   # 會員資訊展示
├── umodify.html   # AJAX 資料修改表單
├── mycomment.html # 我的評論頁
└── retrieve.html  # 密碼找回頁
```

#### 會員配置項（ay_config 表）

| 配置名 | 預設值 | 說明 |
|--------|--------|------|
| register_status | 1 | 註冊開關 |
| register_type | 1 | 註冊類型（1=帳號/2=郵箱/3=手機） |
| register_check_code | 1 | 註冊驗證碼開關 |
| register_verify | 0 | 註冊審核開關 |
| register_score | 0 | 註冊贈送積分 |
| register_gcode | (空) | 註冊預設等級 |
| register_title | 會員註冊 | 註冊頁標題 |
| login_status | 1 | 登入開關 |
| login_check_code | 1 | 登入驗證碼開關 |
| login_title | 會員登錄 | 登入頁標題 |
| ucenter_title | 個人中心 | 個人中心頁標題 |
| umodify_title | 資料修改 | 資料修改頁標題 |
| comment_status | 1 | 評論開關 |
| comment_check_code | 1 | 評論驗證碼開關 |
| comment_verify | 1 | 評論審核開關 |
| comment_anonymous | 0 | 匿名評論開關 |
| home_upload_ext | jpg,jpeg,png,gif,xls,xlsx,doc,docx,ppt,pptx,rar,zip,pdf,txt | 前台上傳允許的副檔名 |

### 後台會員管理（已完成）

#### 後台路由

| 路徑 | 說明 | 路由模式 |
|------|------|---------|
| `/admin/member/index` | 會員列表 | GET |
| `/admin/member/add` | 新增會員 | GET/POST |
| `/admin/member/mod/*action` | 修改會員 | Any（通配符） |
| `/admin/member/del` | 刪除會員 | POST |
| `/admin/member/group/index` | 等級列表 | GET |
| `/admin/member/group/mod/*action` | 修改等級 | Any（通配符） |
| `/admin/member/field/index` | 欄位列表 | GET |
| `/admin/member/field/mod/*action` | 修改欄位 | Any（通配符） |
| `/admin/member/comment/index` | 評論列表 | GET |

#### 後台模板

```
apps/admin/view/member/
├── group.html    # 會員等級（列表/新增/修改）
├── field.html    # 會員欄位（列表/新增/修改）
├── member.html   # 會員管理（列表/新增/修改，含批量操作）
└── comment.html  # 文章評論（列表/詳情/回覆）
```

### 階段 2-4

- **階段 2**（規劃中）：會員中心增強功能（積分系統、等級自動升級）
- **階段 3**（已完成）：前台評論系統（提交/列表/我的評論/刪除）
- **階段 4**（已完成）：密碼找回 + 郵件驗證 + 帳號檢查

---

## 十八、開發防遺忘機制

### 問題分析

在開發過程中，以下問題反覆出現：

1. **忘記項目使用 pongo2** — 文檔已明確記載，但仍每次重新「發現」
2. **忘記 `list`/`mod` 標誌模式** — 已有控制器已建立此模式，但新控制器總是遺漏
3. **忘記方法不存在** — 使用 `JSONErrMsg` 等不存在的方法，應先 Grep 確認
4. **忘記 `*action` 通配符路由** — ContentSort 已使用此模式，但新路由仍用 `:id`
5. **忘記雙 MD5 嵌套語法問題** — Go 不支援深層嵌套的函數調用括號
6. **忘記 time.Time 需要預格式化** — pongo2 直接渲染會顯示時區和納秒
7. **忘記使用繁體中文** — `/修改繁體文本` 要求所有修改文件使用繁體

### 根因

1. **沒有前置檢查流程** — 不系統性地檢查項目約定就寫代碼
2. **不讀已有代碼** — 應至少閱讀一個同類已實現的控制器
3. **不利用現有文檔** — AI_GUIDELINES.md 和本文件包含答案
4. **不查 project_memory.md** — 記憶文件有硬約束
5. **不 Grep 確認存在性** — 使用方法/函數前不確認其存在

### 解決方案

1. **本文件第零章**：寫代碼前必讀的防遺忘清單
2. **API 速查表**：常用方法和模式的快速參考
3. **反模式清單**：曾犯過的錯誤及正確做法
4. **強制檢查流程**：5 步前置檢查流程
5. **記憶文件**：project_memory.md 記錄硬約束和經驗教訓

---

## 十九、ExtField 擴展字段系統（2026-07-08 新增）

### 19.1 數據結構

`ay_extfield` 表新增 `scode TEXT DEFAULT ''` 欄位，用於存儲適用欄目（逗號分隔，空=全展示）：

```sql
-- EnsureExtFieldScodeColumn() 自動執行
ALTER TABLE ay_extfield ADD COLUMN scode TEXT DEFAULT '';
```

`ay_content_ext` 表的每個物理列對應一個 field 名稱，同一模型下 field 名稱必須唯一。

### 19.2 核心函數

```go
// 過濾：返回適用於指定欄目的字段（全展示 + 匹配的）
content.GetExtFieldsByModelCodeAndScode(mcode, scode string) []ExtField

// 檢查：scode 是否匹配（空=全展示，逗號分隔多選）
content.ScodeMatches(fieldScode, targetScode string) bool

// 規範化：換行符轉逗號，清理空項
content.NormalizeOptions(options string) string

// 唯一性：同模型下 field 名稱檢查（excludeID 用於修改時排除自身）
content.CheckFieldUnique(mcode, field string, excludeID int) bool

// 遷移：舊 || 格式自動遷移到 scode 列（冪等）
content.MigrateScodeFromValue()
```

### 19.3 模板注意事項

1. **pongo2 foreach 不支援 `->` 語法**：不能在模板中遍歷 struct 欄位，必須在 controller 預計算為字串（如 `ScodeDisplay`）
2. **scode 下拉用 checkbox 列表**：JS 根據 mcode 動態生成對應欄目的 checkbox，支援多選
3. **修改頁回顯**：controller 傳遞 `extfieldScodes` 陣列，JS 中的 `selectedScodes` 用於回顯

### 19.4 內容頁過濾

`ContentController.contentTemplateData` 接受 scode 參數，傳遞給 `BuildExtFieldTemplateData`：

```go
// 新增頁：scode=""（不顯示指定欄目的字段）
data := cc.contentTemplateData(mcode, "", sorts, nil)

// 修改頁：scode=內容的實際欄目
data := cc.contentTemplateData(mcode, scodeVal, sorts, contentMap)
```

---

## 二十、配置與日誌（2026-07-08 新增）

### 20.1 Viper 配置管理

`config/config.go` 使用 Viper 管理配置，支援環境變數覆蓋：

```go
// 環境變數前綴：PBOOTCMS_GO_
// 範例：PBOOTCMS_GO_DATABASE_TYPE=mysql 覆蓋 database.type
viper.SetEnvPrefix("PBOOTCMS_GO")
viper.AutomaticEnv()
viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
```

`SessionKey` 從配置讀取，不再硬編碼。

### 20.2 slog 結構化日誌

`apps/common/logger.go` 封裝 Go 1.21+ 標準庫的 `log/slog`：

```go
import "pbootcms-go/apps/common"

common.Logger.Info("操作成功", "user", username, "action", "login")
common.Logger.Error("操作失敗", "err", err)
```

使用 `SetLogLevel(level string)` 設定日誌級別（debug/info/warn/error）。

### 20.3 會員輸入驗證

`MemberController.go` 使用正則表達式驗證會員輸入：

```go
// 用戶名：3-20 字元，字母數字下劃線
// 郵箱：標準 email 格式
// 手機：台灣/香港/大陸格式
```

### 20.4 內容批量操作

`mylayui.js` 中使用 jQuery `$(document).on('submit')` 統一攔截表單提交，解決 layui `form.on('submit()')` 與非 `lay-submit` 按鈕的衝突：

- 複製/移動/刪除按鈕不需要 `lay-submit` 屬性
- 用 `button._clicked` class 識別點擊的提交按鈕
- GET 表單直接放行，POST 表單走 AJAX
