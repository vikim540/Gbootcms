# 多語言區域隔離架構

> Gbootcms 多語言系統設計文檔 — 基於 GORM Plugin 數據隔離 + URL 前綴路由

## 架構概覽

```
請求 /sc/article
  ↓
URL 規範化中間件 (main.go)
  ├─ 檢測 /sc 前綴 → isValidAcode("sc") → true
  ├─ 剝離前綴 → c.Request.URL.Path = "/article"
  ├─ 存入 context → middleware.SetURLAcode(ctx, "sc")
  └─ HandleContext 重新路由
  ↓
AcodeMiddleware (acode.go)
  ├─ InjectAcode: 讀取 URL acode → acodeplugin.WithAcode(ctx, "sc")
  └─ c.Request = c.Request.WithContext(ctx)
  ↓
FrontController (front.go)
  ├─ buildContext: ctx.Ctx = c.Request.Context(), ctx.CurrentPath = "/article"
  ├─ model.DB.WithContext(ctx.Ctx) → AcodePlugin 自動 WHERE acode = 'sc'
  └─ 跨區查詢: acodeplugin.SkipAcode(ctx)
  ↓
TagParser (providers.go)
  ├─ {gboot:nav} → 查 ay_content_sort WHERE acode='sc'（僅 11 欄目）
  ├─ {gboot:content scode=1} → 查 ay_content WHERE acode='sc' AND scode=1
  └─ {gboot:language} → 生成保持當前頁面的切換連結
```

## 核心組件

### 1. GORM AcodePlugin（`core/acodeplugin/plugin.go`）

GORM Callback Plugin，自動在所有涉及 `acode` 欄位的表上注入區域過濾條件。

| 回調 | 行為 |
|------|------|
| `BeforeCreate` | 自動填充 `acode` 值（從 context 讀取） |
| `BeforeFind` (Query) | 自動追加 `WHERE acode = ?` |
| `BeforeUpdate` | 自動追加 `WHERE acode = ?` |
| `BeforeDelete` | 自動追加 `WHERE acode = ?` |

**檢測機制**：通過 `schema.LookUpField("acode")` 判斷模型是否有 `acode` 欄位，有則過濾，無則跳過。

**跳過隔離**：`acodeplugin.SkipAcode(ctx)` 返回一個帶特殊標記的 context，Plugin 檢測到此標記後不做任何過濾。

### 2. URL 前綴路由（`main.go`）

```go
r.Use(func(c *gin.Context) {
    // 跳過後台、靜態資源
    if strings.HasPrefix(path, "/admin") { c.Next(); return }

    // 檢測語言前綴
    segments := strings.SplitN(trimmed, "/", 2)
    if isValidAcode(segments[0]) {
        acode := segments[0]
        c.Request.URL.Path = "/" + segments[1] // 剝離前綴
        ctx := middleware.SetURLAcode(c.Request.Context(), acode)
        c.Request = c.Request.WithContext(ctx)
        r.HandleContext(c) // 重新路由
        c.Abort()
        return
    }
    c.Next()
})
```

**設計要點**：
- 使用 `HandleContext` 重新路由，因為 Gin 路由匹配先於中間件
- `isValidAcode` 使用 `sync.Once` 線程安全初始化，緩存合法 acode 列表
- 默認語言（`is_default=1`）不加前綴，URL 為 `/`

### 3. InjectAcode 中間件（`apps/common/middleware/acode.go`）

優先級：URL 前綴 > 後台 session > 域名匹配 > 默認區域

```go
func InjectAcode(c *gin.Context) {
    acode := ""
    // 1. URL 前綴（由 main.go 中間件設置）
    if urlAcode, ok := c.Request.Context().Value(urlAcodeKey{}).(string); ok {
        acode = urlAcode
    }
    // 2. 後台 session
    if acode == "" { acode = common.GetSessionString(c, "acode") }
    // 3. 域名匹配
    if acode == "" { acode = matchDomainToAcode(ctx, c.Request.Host) }
    // 4. 默認區域（DB 查詢 is_default=1，sync.Once 緩存）
    if acode == "" { acode = getDefaultAcode() }

    ctx := acodeplugin.WithAcode(c.Request.Context(), acode)
    c.Request = c.Request.WithContext(ctx)
}
```

### 4. 全局後台注入（`apps/common/Render.go`）

```go
if uid > 0 {
    // 所有後台頁面共享區域切換數據
    var allAreas []model.Area
    model.DB.WithContext(acodeplugin.SkipAcode(ctx)).Find(&allAreas)
    data["Areas"] = allAreas
    data["CurrentAcode"] = currentAcode
    data["OneArea"] = len(allAreas) <= 1
    data["DefaultAcode"] = defaultAcode
    data["HomeURL"] = homeURL // 默認區域 / ，非默認 /{acode}/
}
```

### 5. 語言切換保持當前頁面（`providers.go`）

`{gboot:language}` 標籤根據 `ctx.CurrentPath` 構建保持當前頁面的切換連結：

| 當前頁面 | 繁體中文 (默認) | 簡體中文 | English |
|---------|----------------|---------|---------|
| `/` (首頁) | `/` | `/sc/` | `/en/` |
| `/aboutus` | `/aboutus` | `/sc/aboutus` | `/en/aboutus` |
| `/sc/article` | `/article` | `/sc/article` | `/en/article` |

```go
// 默認區域: /{currentPath}
// 非默認: /{acode}{currentPath}
link := currentPath
if a.Acode != defaultAcode {
    link = "/" + a.Acode + currentPath
}
```

### 6. 前台模板標籤

| 標籤 | 用途 | 範例 |
|------|------|------|
| `{gboot:language}...{/gboot:language}` | 語言切換循環 | `{gboot:language}<a href="[language:link]">[language:name]</a>{/gboot:language}` |
| `{gboot:sitepath}` | 當前語言首頁路徑 | `/` 或 `/sc/` 或 `/en/` |
| `{gboot:homename}` | 當前語言「首頁」文字 | 首頁 / 首页 / Home |
| `{gboot:morename}` | 當前語言「查看更多」文字 | 查看更多 / 查看更多 / View More |
| `{gboot:acode}` | 當前語言代碼 | sc / tc / en |

### 7. 前台 Bootstrap 下拉切換器（`template/default/comm/head.html`）

```html
<div class="btn-group ml-2">
  <button class="btn btn-outline-secondary btn-sm dropdown-toggle" data-toggle="dropdown">
    <i class="fa fa-globe"></i>
    {gboot:language}{gboot:if([language:active])}[language:name]{/gboot:if}{/gboot:language}
  </button>
  <div class="dropdown-menu dropdown-menu-right">
    {gboot:language}
      <a class="dropdown-item {gboot:if([language:active])}active{/gboot:if}" href="[language:link]">[language:name]</a>
    {/gboot:language}
  </div>
</div>
```

## 數據流

### 涉及 acode 的數據表

| 表 | 說明 |
|----|------|
| ay_area | 區域定義（acode, name, domain, is_default） |
| ay_content | 內容文章 |
| ay_content_sort | 欄目分類 |
| ay_company | 公司信息（每區域一條） |
| ay_site | 站點設置（每區域一條，含 theme） |
| ay_link | 友情連結 |
| ay_slide | 幻燈片 |
| ay_tags | 標籤 |
| ay_message | 留言 |
| ay_role_area | 角色-區域權限映射 |
| ay_user.acodes | 用戶可管理的區域列表 |

### 不涉及 acode 的表

| 表 | 原因 |
|----|------|
| ay_member | 會員按 ucode 識別，不按區域隔離 |
| ay_member_comment | 評論歸屬會員，不按區域隔離 |
| ay_member_group | 會員等級全局共享 |
| ay_config | 配置全局共享 |
| ay_menu | 後台菜單全局共享 |
| ay_model | 內容模型全局共享 |
| ay_extfield | 擴展字段全局共享 |

## 安全設計

### 區域切換權限驗證

```go
func (ic *IndexController) Area(c *gin.Context) {
    code := c.PostForm("acode")
    // 驗證用戶是否有權切換到此區域
    userAcodes := common.GetSessionString(c, "user_acodes")
    if !isAllowed(userAcodes, code) {
        c.JSON(403, gin.H{"msg": "無權限切換到此區域"})
        return
    }
    common.SetSession(c, "acode", code)
}
```

### 線程安全

- `validAcodes` 使用 `sync.Once` 初始化，避免 concurrent map writes panic
- `defaultAcodeCache` 使用 `sync.Once` 緩存默認區域
- `InjectAcode` 無狀態，每次創建新 context

## SEO 最佳實踐

### hreflang 標籤（已實現）

在 `comm/head.html` 中使用 `{gboot:hreflang}` 標籤自動生成：

```html
<link rel="alternate" hreflang="zh-Hans" href="https://example.com/sc/aboutus" />
<link rel="alternate" hreflang="zh-Hant" href="https://example.com/aboutus" />
<link rel="alternate" hreflang="en" href="https://example.com/en/aboutus" />
<link rel="alternate" hreflang="x-default" href="https://example.com/aboutus" />
```

**三條鐵律**（均已滿足）：
1. 必須自引用（每個頁面包含指向自己的 hreflang）
2. 必須雙向對稱（A 指向 B，B 必須指回 A）
3. 必須使用正確的 ISO 代碼（zh-Hans / zh-Hant / en）

**acode 到 hreflang 映射**：`sc → zh-Hans`，`tc → zh-Hant`，`en → en`

### canonical 標籤（已實現）

在 `comm/head.html` 中使用 `{gboot:canonical}` 標籤，指向當前頁面的標準 URL（含語言前綴）。

### 其他 SEO 要點

- 每個語言版本 canonical 指向自身 ✅
- hreflang 自引用 + 雙向對稱 + x-default ✅
- 每個語言獨立 XML sitemap ✅（`/sitemap.xml` 索引 + `/sitemap-{acode}.xml`）
- robots.txt 含多語言 sitemap 引用 ✅
- Open Graph 社交分享標籤（og:title / og:description / og:url / og:locale / og:locale:alternate）✅
- 動態 `<html lang>` 屬性（zh-Hans / zh-Hant / en）✅
- 結構化數據本地化（貨幣、語言、地區）— 待實現

### 每語言獨立 XML Sitemap

| 路由 | 說明 |
|------|------|
| `/sitemap.xml` | Sitemap 索引，列出所有語言的 sitemap |
| `/sitemap-tc.xml` | 繁體中文 sitemap（默認語言，URL 無前綴） |
| `/sitemap-sc.xml` | 簡體中文 sitemap（URL 含 `/sc/` 前綴） |
| `/sitemap-en.xml` | 英文 sitemap（URL 含 `/en/` 前綴） |
| `/robots.txt` | 爬蟲規則 + 所有語言 sitemap 引用 |

每個 sitemap 包含：
- 首頁（priority=1.0, changefreq=daily）
- 所有欄目頁（priority=0.8, changefreq=weekly）
- 所有內容頁（priority=0.6, changefreq=monthly, 含 lastmod）

### Open Graph 標籤

在 `comm/head.html` 中使用 `{gboot:og}` 標籤自動生成：

```html
<meta property="og:title" content="頁面標題" />
<meta property="og:description" content="頁面描述" />
<meta property="og:url" content="https://example.com/en/product" />
<meta property="og:type" content="website" />  <!-- 或 article -->
<meta property="og:site_name" content="站點名稱" />
<meta property="og:locale" content="en_US" />
<meta property="og:locale:alternate" content="zh_CN" />
<meta property="og:locale:alternate" content="zh_HK" />
```

### 動態 HTML lang 屬性

在模板中使用 `{gboot:htmllang}` 標籤：

```html
<html lang="{gboot:htmllang}">
```

| acode | html lang | og:locale |
|-------|-----------|-----------|
| sc    | zh-Hans   | zh_CN     |
| tc    | zh-Hant   | zh_HK     |
| en    | en        | en_US     |

## 連結前綴自動重寫（post-render）

### 問題

非默認語言頁面上的導航連結（`[nav:link]`、`[sort:link]`、`[content:link]` 等）不帶語言前綴。用戶在 `/en/product` 點擊「About Us」會跳到 `/aboutus`（默認語言），而非 `/en/aboutus`。

### 方案

在 `postRender` 階段執行一次性正則替換，將所有 `href="/path"`、`action="/path"`、`data-action="/path"` 重寫為帶語言前綴的版本：

```
模板渲染完成
  ↓
提取並保護 <script> 區塊（避免重寫 JavaScript 中的 URL）
  ↓
提取並保護語言切換器區塊（<div class="lang-switch">）
  ↓
正則掃描所有 href/action/data-action="/..." 屬性
  ├─ 已有合法 acode 前綴 → 跳過
  ├─ 需要跳過的路徑（admin/static/api/favicon/#）→ 跳過
  └─ 重寫：href="/path" → href="/{acode}/path"
  ↓
還原語言切換器區塊
  ↓
還原 <script> 區塊
```

**設計要點**：
- 語言切換器連結已由 `{gboot:language}` 正確生成，需保護不被重寫
- `<script>` 區塊中的 URL 不應被重寫（避免破壞 JavaScript 邏輯）
- 默認語言不執行重寫（連結保持原樣）
- 跳過外部連結、協議相對連結（`//`）、錨點（`#`）、管理路徑
- **同時重寫 `action` 和 `data-action` 屬性**，確保表單提交和 AJAX 請求也帶語言前綴

### 重寫的屬性

| 屬性 | 用途 | 範例 |
|------|------|------|
| `href` | 導航連結 | `href="/aboutus"` → `href="/en/aboutus"` |
| `action` | 表單提交目標 | `action="/search"` → `action="/en/search"` |
| `data-action` | AJAX 請求目標 | `data-action="/comment/add?contentid=57"` → `data-action="/en/comment/add?contentid=57"` |

### 會員頁面重定向語言保持

所有會員相關的重定向（登入、登出、註冊、個人中心等）使用 `langPath(c, path)` 函數自動添加語言前綴：

| 場景 | 原路徑 | /sc/ 下重定向 | /en/ 下重定向 |
|------|--------|-------------|-------------|
| 已登入訪問登入頁 | `/ucenter` | `/sc/ucenter` | `/en/ucenter` |
| 未登入訪問個人中心 | `/login` | `/sc/login` | `/en/login` |
| 登出 | `/login` | `/sc/login` | `/en/login` |
| 註冊成功 | `/login` | `/sc/login` | `/en/login` |
| 找回密碼成功 | `/login` | `/sc/login` | `/en/login` |
| 修改資料成功 | `/umodify` | `/sc/umodify` | `/en/umodify` |
| 評論需登入 | `/login?backurl=...` | `/sc/login?backurl=%2Fsc%2F...` | `/en/login?backurl=%2Fen%2F...` |

**backurl 也帶語言前綴**：確保登入成功後返回正確語言的頁面，而非跳回默認語言。

### 留言區域自動隔離

留言提交時 `Acode` 欄位從 context 動態取得，確保不同語言的留言寫入對應區域：

```go
msg := model.Message{
    Acode: acodeplugin.GetAcode(c.Request.Context()),  // 動態取得，非硬編碼
    ...
}
```

## 業界方案對比

| 方案 | 範例 | 優點 | 缺點 | 本專案 |
|------|------|------|------|--------|
| 子目錄 (URL 前綴) | `/sc/`, `/en/` | SEO 權重共享，實施簡單 | 地理定位弱 | ✅ 採用 |
| 子域名 | `sc.example.com` | 介於兩者之間 | 權重不完全共享 | ✗ |
| ccTLD | `example.cn` | 地理定位最強 | 維護成本高 | ✗ |
| URL 參數 | `?lang=sc` | 最簡單 | SEO 不友好 | ✗ |

**Google 推薦**：子目錄方案是大多數情況下的首選，首次做國際化的首選方案。

## 已知限制與待改進

1. `validAcodes` 和 `defaultAcodeCache` 永不刷新 — 管理員新增區域後需重啟服務
2. `matchDomainToAcode` 每次請求查 DB 無緩存 — 高流量場景需加 TTL 緩存
3. `currentHomePath`、`language`、`buildHreflang`、`buildCanonical`、`buildOpenGraph` 每次調用查 DB — 應緩存到 ctx
4. 結構化數據本地化 — 待實現

## 安全修復清單

| 問題 | 修復方案 | 影響文件 |
|------|---------|---------|
| SQL 注入：`ctx.Filters` 字段名直接拼接 SQL | `IsSafeFieldName()` 白名單驗證 `^ext_[a-zA-Z0-9_]+$` | `providers.go`、`front.go` |
| XSS：QRCode provider `str` 未轉義 | `url.QueryEscape(str)` URL 編碼 | `providers.go` |
| XSS：`pathinfo` query key 未 HTML 轉義 | `html.EscapeString(key)` + `url.QueryEscape` | `Render.go` |
| Cookie `HttpOnly: false` | 改為 `HttpOnly: true`（JS 不需要讀取） | `IndexController.go`、`captcha.go` |
| 連結重寫跳過列表過寬 | 移除 `sort/`、`content/`、`tags`、`login` 等不應跳過的路徑 | `front.go` |
| 語言切換器正則不匹配巢狀 div | 改為 `</div>\s*</div>` 匹配外層閉合 | `front.go` |
| 留言 `Acode` 硬編碼 `"cn"` | 改為 `acodeplugin.GetAcode(c.Request.Context())` 動態取得 | `front.go` |
| 會員重定向丟失語言前綴 | 新增 `langPath(c, path)` 函數，應用於所有重定向和 tourl | `front.go`、`member.go`、`comment.go` |
| `backurl` 丟失語言前綴 | `langPath` 應用於 backurl 值，確保登入後返回正確語言頁面 | `front.go`、`comment.go` |
| 表單 `action`/`data-action` 未重寫 | 擴展 `linkRewriteRe` 正則同時匹配三種屬性 | `front.go` |
| 首頁變體重定向丟失語言 | `/index` 重定向改用 `langPath(c, "/")` | `front.go` |
| `getDefaultAcode` 未導出 | 改為 `GetDefaultAcode()` 供控制器層使用 | `middleware/acode.go` |
