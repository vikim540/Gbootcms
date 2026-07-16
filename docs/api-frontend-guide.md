# Gbootcms API 前端對接指南

> **版本**：v1.0 | **更新日期**：2026-07-16 | **適用對象**：前端開發工程師

本指南面向需要與 Gbootcms CMS 進行前後端分離對接的前端工程師。涵蓋 API 認證流程、全部接口規格、後台配置操作步驟、以及前端部署規範。

---

## 整體架構

Gbootcms 提供一套 RESTful API，固定前綴 `/api/v1/`，與 PbootCMS 原版 `api.php` 路由和 `appid+timestamp+MD5` 簽名鑑權完全不相容。API 與前台模板渲染共用同一個 Go 服務進程，監聽同一端口（默認 8080）。

```
前端應用（Vue/React/Next.js）
    │
    ├── 公開接口 → 無需認證，直接調用
    │     GET  /api/v1/site          站點資訊
    │     GET  /api/v1/contents       內容列表
    │     POST /api/v1/messages       提交留言
    │     ...
    │
    └── 認證接口 → 需要 JWT Token 或 API Key
          GET  /api/v1/messages       留言列表
          GET  /api/v1/forms/:fcode/data  表單數據
          POST /api/v1/auth/refresh   刷新 Token
```

### 與前台模板的關係

Gbootcms 同時支援傳統服務端模板渲染（`{gboot:xxx}` 標籤）和 API 前後端分離兩種模式。兩者讀取的是同一個 SQLite 數據庫，API 自動套用與前台相同的區域（acode）數據隔離。如果你在前端使用 API 獲取數據，就不需要再解析 `{gboot:xxx}` 模板標籤。

---

## 認證體系

### 認證方式

API 支援兩種認證方式，可以根據場景選擇：

| 方式 | 適用場景 | 傳遞方法 | 過期時間 |
|------|---------|---------|---------|
| JWT Token | 用戶登入後的操作 | `Authorization: Bearer <token>` | 72 小時 |
| API Key | 服務端到服務端調用 | `X-API-Key: <key>` 或 `?api_key=<key>` | 永不過期 |

JWT Token 過期後可以通過 `/auth/refresh` 接口刷新（需攜帶即將過期的 Token）。API Key 在後台配置後長期有效，適合伺服器端定時拉取數據的場景。

### 公開接口與認證接口

大部分 GET 查詢接口是公開的，不需要任何認證。需要認證的接口僅限涉及用戶隱私數據的操作：

| 接口 | 認證要求 | 原因 |
|------|---------|------|
| `GET /messages` | 需要 | 留言列表含客戶聯繫方式 |
| `GET /forms/:fcode/fields` | 需要 | 表單字段定義屬於後台配置 |
| `GET /forms/:fcode/data` | 需要 | 表單提交數據含客戶隱私 |
| `POST /auth/refresh` | 需要 | 刷新 Token 必須驗證當前身份 |

其餘所有 GET 查詢接口和 `POST /messages`（留言提交）均為公開接口。

### 後台配置步驟

前端開始對接前，需要後台管理員完成以下配置：

1. 登入後台 `http://your-domain/admin`（默認帳號 `admin` / `123456`）
2. 進入 **系統 → 系統配置 → API 設置** 頁面
3. 配置以下項目：

| 配置項 | 說明 | 範例值 |
|--------|------|--------|
| `api_jwt_secret` | JWT 簽名密鑰，留空則 API 登入不可用 | `my-secret-key-2026` |
| `api_key` | API Key，留空則 API Key 認證不可用 | `ak_xxxxxxxxxxxx` |
| `api_cors_origins` | 允許的跨域來源，逗號分隔，`*` 表示允許全部 | `https://example.com,https://www.example.com` |

如果 `api_jwt_secret` 未配置，調用 `/auth/login` 會返回 500 錯誤。如果 `api_cors_origins` 為 `*`，生產環境存在安全風險，建議配置具體域名。

### JWT 登入流程

```
1. POST /api/v1/auth/login
   Body: {"username":"admin","password":"123456"}
   Response: {"code":1,"msg":"登入成功","data":{"token":"xxx","expires_in":259200,"user":{...}}}

2. 攜帶 Token 調用認證接口
   Header: Authorization: Bearer xxx

3. Token 即將過期時刷新
   POST /api/v1/auth/refresh
   Response: {"code":1,"msg":"刷新成功","data":{"token":"yyy","expires_in":259200}}
```

登入接口有安全防護：連續失敗 5 次（可配置 `lock_count`）會鎖定該 IP 900 秒（可配置 `lock_time`）。密碼使用雙 MD5 存儲，比對時使用常量時間比較防止時序攻擊。

### API Key 使用方式

API Key 適合不需要用戶登入的場景，例如定時任務拉取最新文章：

```bash
# 通過 Header
curl -H "X-API-Key: ak_xxxxxxxxxxxx" https://your-domain/api/v1/messages

# 通過 Query 參數
curl https://your-domain/api/v1/messages?api_key=ak_xxxxxxxxxxxx
```

API Key 認證後 `api_uid` 為 0，`api_username` 為 `"api_key"`，不具備任何用戶級權限。

---

## 統一響應格式

所有 API 接口返回統一的 JSON 結構：

```json
{
  "code": 1,
  "msg": "success",
  "data": {},
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 100
  }
}
```

| 字段 | 類型 | 說明 |
|------|------|------|
| `code` | int | `1` = 成功，`0` = 失敗 |
| `msg` | string | 狀態描述文字 |
| `data` | any | 響應數據，失敗時可能不存在 |
| `meta` | object | 分頁信息，僅分頁接口返回 |

失敗響應的 HTTP 狀態碼同時反映錯誤類型：

| HTTP 狀態碼 | 含義 |
|-------------|------|
| 200 | 請求成功（code=1）或業務邏輯失敗（code=0） |
| 400 | 請求參數錯誤 |
| 401 | 未認證或認證失效 |
| 404 | 資源不存在 |
| 429 | 請求過於頻繁（登入鎖定或留言限流） |
| 500 | 伺服器內部錯誤 |

### 分頁規範

所有列表接口支援分頁，通過 Query 參數控制：

| 參數 | 默認值 | 說明 |
|------|--------|------|
| `page` | 1 | 頁碼，從 1 開始 |
| `pagesize` | 15 | 每頁條數，最大 100 |

分頁信息在 `meta` 字段中返回：

```json
{
  "code": 1,
  "msg": "success",
  "data": [...],
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 42
  }
}
```

---

## 接口清單

### 認證接口

#### POST /auth/login — 管理員登入

公開接口。使用後台管理員帳號登入，獲取 JWT Token。

**請求體**：

```json
{
  "username": "admin",
  "password": "123456"
}
```

**成功響應**（200）：

```json
{
  "code": 1,
  "msg": "登入成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 259200,
    "user": {
      "id": 1,
      "username": "admin",
      "realname": "管理員"
    }
  }
}
```

**失敗響應**：

| 場景 | HTTP 狀態碼 | 響應 |
|------|-------------|------|
| 參數缺失 | 400 | `{"code":0,"msg":"請求參數錯誤"}` |
| 密碼錯誤 | 401 | `{"code":0,"msg":"用戶名或密碼錯誤"}` |
| IP 被鎖定 | 429 | `{"code":0,"msg":"登入嘗試過多，請 899 秒後再試"}` |
| JWT 未配置 | 500 | `{"code":0,"msg":"API 未正確配置，請聯繫管理員設定 api_jwt_secret"}` |

#### POST /auth/refresh — 刷新 Token

需要認證。使用當前有效的 Token 換取新 Token。

**成功響應**（200）：

```json
{
  "code": 1,
  "msg": "刷新成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 259200
  }
}
```

---

### 站點與公司

#### GET /site — 站點資訊

公開接口。返回站點基本配置。

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "title": "網站標題",
    "subtitle": "網站副標題",
    "domain": "https://example.com",
    "logo": "static/upload/202606/logo.png",
    "keywords": "網站關鍵詞",
    "description": "網站描述",
    "icp": "粵ICP備XXXXXXX號",
    "theme": "default"
  }
}
```

`logo` 等文件路徑為相對路徑，前端需自行拼接域名前綴。

#### GET /company — 公司資訊

公開接口。返回公司聯繫信息。

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "name": "公司名稱",
    "address": "公司地址",
    "phone": "電話",
    "mobile": "手機",
    "email": "郵箱",
    "qq": "QQ號",
    "wechat": "微信號"
  }
}
```

---

### 欄目管理

#### GET /sorts — 欄目列表

公開接口。返回欄目（分類）列表。

**Query 參數**：

| 參數 | 說明 | 範例 |
|------|------|------|
| `scode` | 欄目編碼，篩選指定欄目及其直接子欄目 | `scode=1` |
| `mcode` | 內容模型編碼 | `mcode=1` |
| `status` | 狀態，默認 `1`（啟用），`-1` 為全部 | `status=1` |

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "scode": "1",
      "pcode": "0",
      "name": "關於我們",
      "filename": "about",
      "urlname": "",
      "mcode": "1",
      "listtpl": "list.html",
      "contenttpl": "content.html",
      "status": 1,
      "sorting": 1
    }
  ]
}
```

#### GET /sorts/:scode — 欄目詳情

公開接口。支持按 `scode`、`filename` 或 `urlname` 查詢。

```bash
GET /api/v1/sorts/1          # 按 scode
GET /api/v1/sorts/about      # 按 filename
GET /api/v1/sorts/news       # 按 urlname
```

#### GET /nav — 導航樹

公開接口。返回樹狀結構的導航菜單，前端可直接用於渲染多級導航。

**Query 參數**：

| 參數 | 說明 |
|------|------|
| `scode` | 指定根欄目，只返回該欄目及其子欄目 |

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "scode": "1",
      "name": "產品中心",
      "filename": "product",
      "sorting": 1,
      "children": [
        {
          "id": 2,
          "scode": "2",
          "name": "產品A",
          "filename": "product-a",
          "sorting": 1,
          "children": []
        }
      ]
    }
  ]
}
```

---

### 內容管理

#### GET /contents — 內容列表

公開接口。返回已發佈的文章/產品列表，支援多維度篩選。

**Query 參數**：

| 參數 | 說明 | 範例 |
|------|------|------|
| `scode` | 欄目編碼，自動包含所有子孫欄目（遞迴 CTE） | `scode=2` |
| `mcode` | 內容模型編碼 | `mcode=2` |
| `keyword` | 搜索關鍵字（標題、關鍵詞、描述） | `keyword=手機` |
| `istop` | 置頂篩選 | `istop=1` |
| `isrecommend` | 推薦篩選 | `isrecommend=1` |
| `order` | 排序方式：`date`（默認）、`visits`、`sorting` | `order=visits` |
| `page` | 頁碼 | `page=1` |
| `pagesize` | 每頁條數（最大 100） | `pagesize=20` |

`scode` 參數使用遞迴 CTE 查詢，會自動包含指定欄目的所有子欄目和孫子欄目。例如 `scode=1` 會返回欄目 1 及其全部子孫欄目下的內容，與前台 `{{gboot:list}}` 標籤的行為一致。

**響應**：

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 52,
      "title": "文章標題",
      "subtitle": "副標題",
      "date": "2026-07-13 16:52:30",
      "ico": "static/upload/202607/image.png",
      "description": "文章摘要",
      "keywords": "關鍵詞",
      "visits": 128,
      "likes": 5,
      "scode": "2",
      "istop": 0,
      "isrecommend": 1,
      "isheadline": 0,
      "url": "/article/52.html",
      "create_time": "2026-07-07 15:34:08",
      "update_time": "2026-07-13 16:52:30",
      "ext": {
        "ext_type": "基礎版",
        "ext_price": "99"
      }
    }
  ],
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 42
  }
}
```

`ext` 字段包含自定義擴展字段，字段名以 `ext_` 為前綴。不同欄目的擴展字段不同，前端應動態渲染。

`url` 字段為內容的相對路徑，規則為：外鏈 → `filename.html` → `urlname` → `/content/{id}.html`。

#### GET /contents/:id — 內容詳情

公開接口。返回單篇內容的完整信息。

**Query 參數**：

| 參數 | 說明 |
|------|------|
| `track` | `1` = 累加訪問量（默認不計數，避免 API 輪詢污染統計） |

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "id": 52,
    "title": "文章標題",
    "subtitle": "副標題",
    "titlecolor": "",
    "author": "作者",
    "source": "來源",
    "date": "2026-07-13 16:52:30",
    "ico": "static/upload/202607/image.png",
    "pics": "img1.jpg,img2.jpg",
    "content": "<p>正文 HTML</p>",
    "tags": "標籤1,標籤2",
    "keywords": "關鍵詞",
    "description": "摘要",
    "visits": 128,
    "likes": 5,
    "scode": "2",
    "istop": 0,
    "isrecommend": 1,
    "isheadline": 0,
    "url": "/article/52.html",
    "create_time": "2026-07-07 15:34:08",
    "update_time": "2026-07-13 16:52:30",
    "ext": {
      "ext_type": "基礎版"
    },
    "sort": {
      "id": 2,
      "scode": "2",
      "name": "欄目名稱"
    },
    "prev": {
      "id": 51,
      "title": "上一篇",
      "url": "/article/51.html"
    },
    "next": null
  }
}
```

`prev` 和 `next` 在不存在時為 `null`。`content` 字段為富文本 HTML，前端渲染時注意 XSS 防護。

`pics` 字段為逗號分隔的多圖路徑，前端需 `split(',')` 後逐個渲染。

#### GET /contents/:id/images — 內容圖片

公開接口。返回內容的圖片列表，從 `pics` 字段拆分而來。

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "id": 52,
    "title": "文章標題",
    "ico": "static/upload/202607/image.png",
    "images": [
      "img1.jpg",
      "img2.jpg",
      "img3.jpg"
    ]
  }
}
```

---

### 搜索

#### GET /search — 內容搜索

公開接口。支援模糊搜索和精準搜索。

**Query 參數**：

| 參數 | 說明 | 範例 |
|------|------|------|
| `keyword` | 搜索關鍵字（必填） | `keyword=手機` |
| `field` | 搜索字段，`|` 分隔，默認 `title\|keywords\|description` | `field=title` |
| `fuzzy` | `1` = 模糊匹配（默認），`0` = 精準匹配 | `fuzzy=0` |
| `page` | 頁碼 | `page=1` |
| `pagesize` | 每頁條數 | `pagesize=20` |

搜索字段白名單僅允許 `title`、`keywords`、`description`，其他值會被忽略。如果後台配置了 MeiliSearch 搜索引擎，API 會自動使用 MeiliSearch 進行全文搜索；MeiliSearch 不可用時自動降級到 SQL LIKE 查詢，前端無需處理差異。

---

### 留言

#### POST /messages — 提交留言

公開接口。提交一條留言到後台留言管理。

**安全限制**：同一 IP 60 秒內最多提交 3 次留言，超限返回 429。

**請求體**：

```json
{
  "contacts": "張先生",
  "mobile": "13800138000",
  "content": "我想了解一下你們的產品"
}
```

| 字段 | 必填 | 說明 |
|------|------|------|
| `contacts` | 是 | 聯繫人姓名 |
| `mobile` | 否 | 聯繫電話 |
| `content` | 是 | 留言內容 |

所有文字輸入經過 XSS 過濾（`FilterUserInput`）和敏感詞替換（`FilterSensitiveWords`）。服務端自動採集訪客 IP、操作系統、瀏覽器 UA，如果請求攜帶了有效的 JWT Token 還會記錄會員 UID。

**成功響應**（200）：

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "id": 156
  }
}
```

**限流響應**（429）：

```json
{
  "code": 0,
  "msg": "提交過於頻繁，請稍後再試"
}
```

#### GET /messages — 留言列表

**需要認證**。返回後台留言列表，用於管理端展示。

**Query 參數**：

| 參數 | 說明 |
|------|------|
| `status` | `0` = 待處理，`1` = 已回覆 |
| `page` | 頁碼 |
| `pagesize` | 每頁條數 |

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 156,
      "contacts": "張先生",
      "mobile": "13800138000",
      "content": "我想了解一下產品",
      "user_ip": "192.168.1.1",
      "user_os": "Windows 11",
      "browser": "Chrome 126",
      "uid": 0,
      "status": 0,
      "recontent": "",
      "create_time": "2026-07-16 10:30:00"
    }
  ],
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 8
  }
}
```

---

### 自定義表單

#### GET /forms/:fcode/fields — 表單字段定義

**需要認證**。返回指定表單的字段定義，前端可用於動態渲染表單。

```bash
GET /api/v1/forms/1/fields
```

```json
{
  "code": 1,
  "msg": "success",
  "data": {
    "form": {
      "id": 1,
      "fcode": "1",
      "name": "報名表單",
      "table_name": "ay_diy_form_1"
    },
    "fields": [
      {
        "id": 1,
        "name": "name",
        "description": "姓名",
        "type": 1,
        "sorting": 1
      },
      {
        "id": 2,
        "name": "phone",
        "description": "電話",
        "type": 1,
        "sorting": 2
      }
    ]
  }
}
```

#### GET /forms/:fcode/data — 表單數據列表

**需要認證**。返回指定表單的提交數據。

**Query 參數**：

| 參數 | 說明 |
|------|------|
| `page` | 頁碼 |
| `pagesize` | 每頁條數 |

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 35,
      "name": "李先生",
      "phone": "13800138000",
      "create_time": "2026-07-15 14:20:00"
    }
  ],
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 35
  }
}
```

表單數據的 `data` 字段為動態結構，具體字段取決於表單字段定義。表名經過白名單驗證（`CheckVarType`），防止 SQL 注入。

---

### 其他資源

#### GET /slides — 幻燈片列表

公開接口。

**Query 參數**：`gid`（分組 ID）

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "gid": 1,
      "title": "首頁Banner",
      "pic": "static/upload/202606/banner1.jpg",
      "link": "https://example.com",
      "sorting": 1
    }
  ]
}
```

#### GET /links — 友情連結列表

公開接口。

**Query 參數**：`gid`（分組 ID）

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "gid": 1,
      "name": "合作夥伴",
      "link": "https://partner.com",
      "logo": "static/upload/202606/partner.png",
      "sorting": 1
    }
  ]
}
```

#### GET /tags — 標籤列表

公開接口。返回所有內鏈標籤。

```json
{
  "code": 1,
  "msg": "success",
  "data": [
    {
      "id": 5,
      "name": "錘子",
      "link": "https://www.smartisan.com/",
      "sorting": 0
    }
  ]
}
```

---

## 前端部署規範

### 靜態資源路徑

API 返回的所有文件路徑（`logo`、`ico`、`pics`、`pic` 等）均為相對路徑，以 `static/` 開頭。前端有兩種處理方式：

**方案 A：反向代理（推薦生產環境）**

在 Nginx 或 CDN 中將 `/static/` 路徑代理到 Gbootcms 伺服器：

```nginx
location /static/ {
    proxy_pass http://127.0.0.1:8080;
}
location /api/ {
    proxy_pass http://127.0.0.1:8080;
}
```

前端直接使用 API 返回的相對路徑：`<img :src="'/static/' + item.ico" />`

**方案 B：拼接域名前綴**

從 `/site` 接口獲取 `domain` 配置，拼接完整 URL：

```javascript
const domain = siteData.domain || ''
const imageUrl = domain + '/' + item.ico
```

注意 `domain` 配置可能為空，此時應使用當前請求的 origin 作為前綴。

### 跨域配置

開發環境下，前端 dev server（如 `localhost:3000`）調用 API（如 `localhost:8080`）會遇到跨域問題。API 已內建 CORS 支援，處理方式取決於後台配置：

| `api_cors_origins` 配置值 | 行為 |
|---------------------------|------|
| `*`（默認） | 允許任意來源，開發環境方便使用 |
| `https://example.com,https://www.example.com` | 僅允許列出的來源 |

CORS 預檢請求（OPTIONS）返回 204 No Content，`Access-Control-Max-Age` 設為 86400 秒（24 小時），瀏覽器會快取預檢結果。

開發環境推薦配置 `*`，生產環境必須配置具體域名清單。

### Token 管理

前端應在記憶體中保存 JWT Token（不建議存入 localStorage，防 XSS 攔截），並在每次請求時附加到 Header：

```javascript
// Axios 攔截器範例
apiClient.interceptors.request.use(config => {
  const token = getAuthToken() // 從記憶體/Pinia/Vuex 獲取
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 401 自動刷新
apiClient.interceptors.response.use(
  response => response,
  async error => {
    if (error.response?.status === 401) {
      try {
        const { data } = await apiClient.post('/api/v1/auth/refresh')
        setAuthToken(data.data.token)
        error.config.headers.Authorization = `Bearer ${data.data.token}`
        return apiClient(error.config) // 重試原請求
      } catch {
        clearAuthToken()
        router.push('/login')
      }
    }
    return Promise.reject(error)
  }
)
```

Token 有效期 72 小時（259200 秒），`expires_in` 字段已告知前端確切過期時間。建議在過期前 1 小時主動刷新。

### 內容 URL 規則

API 返回的 `url` 字段遵循 PbootCMS 的 URL 生成規則，前端可直接使用：

| 優先級 | 條件 | URL 格式 | 範例 |
|--------|------|---------|------|
| 1 | 有外鏈 | 直接返回外鏈 URL | `https://example.com` |
| 2 | 有 filename | `/{filename}.html` | `/about.html` |
| 3 | 有 urlname | `/{urlname}` | `/news` |
| 4 | 默認 | `/content/{id}.html` | `/content/52.html` |

如果前端使用 SPA 路由，需要將這些 URL 映射到對應的頁面組件。

### 擴展字段處理

內容列表和詳情的 `ext` 字段為動態鍵值對，字段名以 `ext_` 前綴。不同欄目配置不同的擴展字段，前端應動態渲染：

```javascript
// 動態渲染擴展字段
Object.entries(item.ext || {}).forEach(([key, value]) => {
  if (key.startsWith('ext_') && value !== null) {
    const label = fieldLabels[key] || key.replace('ext_', '')
    renderField(label, value)
  }
})
```

後台配置擴展字段的路徑：**內容 → 擴展字段管理**。每個擴展字段綁定到特定的內容模型（mcode），前端可通過 `/forms/:fcode/fields` 接口獲取字段定義。

### 區域（acode）隔離

GbootCMS 支援多區域數據隔離。API 自動根據請求來源判斷當前區域，返回對應區域的數據。前端不需要手動傳遞 `acode` 參數。如果需要切換區域，使用不同的域名或路徑前綴訪問 API 即可。

### 訪問量統計

`GET /contents/:id` 默認不計入訪問量。如果需要統計 API 調用的訪問量，添加 `?track=1` 參數：

```bash
GET /api/v1/contents/52?track=1
```

建議只在真實用戶瀏覽時傳遞 `track=1`，API 輪詢或數據同步場景不要傳遞，避免污染統計數據。

### 錯誤處理規範

前端應統一處理 API 錯誤，建議按 HTTP 狀態碼分類：

| 狀態碼 | 處理方式 |
|--------|---------|
| 400 | 顯示 `msg` 字段內容，提示用戶修正參數 |
| 401 | 清除 Token，跳轉登入頁 |
| 404 | 顯示「資源不存在」提示 |
| 429 | 顯示 `msg` 中的冷卻時間提示，倒計時後允許重試 |
| 500 | 顯示「伺服器錯誤，請稍後重試」，記錄到日誌 |

對於網絡錯誤（無響應），建議實施指數退避重試（最多 3 次）。

---

## 接口速查表

| 方法 | 路徑 | 認證 | 說明 |
|------|------|------|------|
| POST | `/auth/login` | 公開 | 管理員登入，返回 JWT Token |
| POST | `/auth/refresh` | JWT | 刷新 Token |
| GET | `/site` | 公開 | 站點資訊 |
| GET | `/company` | 公開 | 公司資訊 |
| GET | `/sorts` | 公開 | 欄目列表 |
| GET | `/sorts/:scode` | 公開 | 欄目詳情 |
| GET | `/nav` | 公開 | 導航樹 |
| GET | `/contents` | 公開 | 內容列表（支援 scode/keyword/istop 篩選） |
| GET | `/contents/:id` | 公開 | 內容詳情（`?track=1` 計訪問量） |
| GET | `/contents/:id/images` | 公開 | 內容圖片列表 |
| GET | `/search` | 公開 | 搜索內容 |
| POST | `/messages` | 公開 | 提交留言（限流 3次/60秒） |
| GET | `/messages` | 認證 | 留言列表 |
| GET | `/forms/:fcode/fields` | 認證 | 表單字段定義 |
| GET | `/forms/:fcode/data` | 認證 | 表單數據列表 |
| GET | `/slides` | 公開 | 幻燈片列表 |
| GET | `/links` | 公開 | 友情連結列表 |
| GET | `/tags` | 公開 | 標籤列表 |

---

## cURL 調用範例

```bash
# 獲取站點資訊
curl https://your-domain/api/v1/site

# 獲取產品欄目下的內容（含子欄目）
curl "https://your-domain/api/v1/contents?scode=2&pagesize=10"

# 搜索關鍵字
curl "https://your-domain/api/v1/search?keyword=手機&fuzzy=1"

# 提交留言
curl -X POST https://your-domain/api/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"contacts":"張先生","mobile":"13800138000","content":"想了解產品"}'

# 管理員登入
curl -X POST https://your-domain/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'

# 使用 Token 查看留言列表
curl -H "Authorization: Bearer eyJhbG..." https://your-domain/api/v1/messages

# 使用 API Key 查看留言列表
curl -H "X-API-Key: ak_xxx" https://your-domain/api/v1/messages
```
