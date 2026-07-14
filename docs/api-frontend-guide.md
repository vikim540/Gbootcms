# Gbootcms RESTful API 前端對接指南

> 本文件供前端開發者快速理解 API 介面規範，完成前後端分離對接。
> 生成日期：2026-07-14

---

## 一、基礎資訊

| 項目 | 值 |
|------|-----|
| Base URL | `http://localhost:8080/api/v1` |
| 認證方式 | JWT Bearer Token 或 API Key（二選一） |
| 請求格式 | `Content-Type: application/json` |
| 回應格式 | JSON |
| CORS | 已啟用，支援 `credentials: 'include'` |
| 字元編碼 | UTF-8 |

---

## 二、認證機制

### 2.1 公開端點（無需認證）

以下端點可直接調用，無需攜帶任何認證頭：

```
POST /auth/login
POST /auth/refresh
GET  /site
GET  /company
GET  /sorts
GET  /sorts/:scode
GET  /nav
GET  /contents
GET  /contents/:id
GET  /contents/:id/images
GET  /search
POST /messages        ← 留言提交公開
GET  /slides
GET  /links
GET  /tags
```

### 2.2 需認證端點

以下端點必須攜帶 JWT Token 或 API Key：

```
GET  /messages        ← 留言列表需認證
GET  /forms/:fcode/fields
GET  /forms/:fcode/data
```

### 2.3 認證頭格式

**方式一：JWT Token（推薦前端使用）**

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**方式二：API Key（推薦伺服器間通訊使用）**

```http
X-API-Key: your-api-key-here
```

或作為 query 參數：

```
GET /api/v1/messages?api_key=your-api-key-here
```

### 2.4 登入取得 Token

```javascript
// 登入
const res = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    username: 'admin',
    password: '123456'
  })
});
const data = await res.json();

if (data.code === 1) {
  // 儲存 token
  localStorage.setItem('api_token', data.data.token);
  localStorage.setItem('token_expires', Date.now() + data.data.expires_in * 1000);
  console.log('登入成功', data.data.user);
} else {
  console.error('登入失敗', data.msg);
}
```

### 2.5 攜帶 Token 調用

```javascript
// 封裝統一的請求函數
async function apiRequest(url, options = {}) {
  const token = localStorage.getItem('api_token');

  // Token 過期自動刷新
  if (token && Date.now() > parseInt(localStorage.getItem('token_expires'))) {
    await refreshToken();
  }

  const headers = {
    'Content-Type': 'application/json',
    ...options.headers
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${url}`, { ...options, headers });

  // 401 時跳轉登入
  if (res.status === 401) {
    localStorage.removeItem('api_token');
    window.location.href = '/login';
    return;
  }

  return res.json();
}

// 刷新 Token
async function refreshToken() {
  const res = await fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('api_token')}`
    }
  });
  const data = await res.json();
  if (data.code === 1) {
    localStorage.setItem('api_token', data.data.token);
    localStorage.setItem('token_expires', Date.now() + data.data.expires_in * 1000);
  }
}
```

---

## 三、統一回應格式

### 3.1 成功回應

```json
{
  "code": 1,
  "msg": "success",
  "data": { ... } | [ ... ] | null,
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 100
  }
}
```

### 3.2 失敗回應

```json
{
  "code": 0,
  "msg": "錯誤訊息"
}
```

> `data` 和 `meta` 在失敗時省略（`omitempty`）。

### 3.3 分頁回應

所有列表端點返回 `meta` 欄位：

```json
{
  "code": 1,
  "msg": "success",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "pagesize": 15,
    "total": 42
  }
}
```

---

## 四、HTTP 狀態碼

| 狀態碼 | 含義 | 場景 |
|--------|------|------|
| 200 | 成功 | 所有成功請求 |
| 400 | 請求參數錯誤 | 缺少必填欄位、ID 格式錯誤 |
| 401 | 認證失敗 | Token 無效或過期、缺少 API Key |
| 404 | 資源不存在 | 內容/欄目/表單未找到 |
| 429 | 請求過多 | 登入鎖定、留言頻率限制 |
| 500 | 伺服器錯誤 | JWT 密鑰未配置、資料庫錯誤 |

---

## 五、通用查詢參數

| 參數 | 類型 | 預設 | 說明 |
|------|------|------|------|
| `page` | int | 1 | 頁碼 |
| `pagesize` | int | 15 | 每頁筆數（最大 100） |
| `scode` | string | - | 欄目編號 |
| `mcode` | string | - | 模型編號 |
| `keyword` | string | - | 搜尋關鍵字 |
| `field` | string | `title\|keywords\|description` | 搜尋欄位（`|` 分隔） |
| `fuzzy` | string | `1` | `1`=模糊匹配, `0`=精準匹配 |
| `order` | string | `date` | 排序：`date`/`visits`/`sorting` |
| `track` | string | - | `1`=累加瀏覽計數（僅 `GET /contents/:id`） |
| `gid` | string | - | 幻燈片/友情連結分組 ID |
| `status` | string | - | 留言狀態篩選 |

---

## 六、端點詳情

### 6.1 認證

#### POST /auth/login — 登入

```json
// 請求
{ "username": "admin", "password": "123456" }

// 回應
{
  "code": 1,
  "msg": "登入成功",
  "data": {
    "token": "eyJhbGci...",
    "expires_in": 259200,
    "user": { "id": 1, "username": "admin", "realname": "管理員" }
  }
}
```

> 連續 5 次密碼錯誤後鎖定 15 分鐘（返回 429）。

#### POST /auth/refresh — 刷新 Token

需攜帶有效 JWT Token，返回新 Token。

---

### 6.2 站點資訊

#### GET /site

```json
{
  "code": 1,
  "data": {
    "title": "網站標題",
    "subtitle": "副標題",
    "domain": "example.com",
    "logo": "/static/upload/logo.png",
    "keywords": "關鍵字",
    "description": "網站描述",
    "icp": "備案號",
    "theme": "default"
  }
}
```

#### GET /company

```json
{
  "code": 1,
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

### 6.3 欄目

#### GET /sorts?scode=&mcode=

返回欄目列表（平鋪，非樹狀）。

#### GET /sorts/:scode

按 `scode`、`filename` 或 `urlname` 查詢欄目詳情。

#### GET /nav?scode=

返回樹狀導航結構（含 `children` 遞迴子欄目）：

```json
{
  "code": 1,
  "data": [
    {
      "id": 1,
      "scode": "S10001",
      "name": "關於我們",
      "filename": "about",
      "sorting": 1,
      "children": [
        {
          "id": 2,
          "scode": "S10002",
          "name": "公司簡介",
          "filename": "intro",
          "sorting": 1,
          "children": []
        }
      ]
    }
  ]
}
```

> 空結果返回 `[]`（不是 `null`）。

---

### 6.4 內容

#### GET /contents?scode=&keyword=&order=&page=&pagesize=

內容列表，含分頁。每條內容包含 `ext` 擴展欄位（自定義欄位）。

#### GET /contents/:id?track=1

內容詳情。`track=1` 時累加瀏覽計數。

`prev` 和 `next` 欄位為物件或 `null`（無上/下一篇時）：

```json
{
  "code": 1,
  "data": {
    "id": 52,
    "title": "文章標題",
    "content": "<p>HTML 內容</p>",
    "ext": { "ext_author": "張三", "ext_price": "100" },
    "prev": { "id": 51, "title": "上一篇", "url": "/about-51.html" },
    "next": null
  }
}
```

#### GET /contents/:id/images

內容多圖列表（`pics` 欄位按逗號拆分）：

```json
{
  "code": 1,
  "data": {
    "id": 52,
    "title": "文章標題",
    "ico": "/static/upload/cover.jpg",
    "images": ["/static/upload/1.jpg", "/static/upload/2.jpg"]
  }
}
```

> 無圖時 `images` 返回 `[]`（不是 `null`）。

---

### 6.5 搜索

#### GET /search?keyword=&field=title|keywords|description&fuzzy=1&page=&pagesize=

優先使用 MeiliSearch 全文搜索，未配置時自動降級到 SQL LIKE。

```javascript
// 精準搜索標題
const res = await apiRequest('/search?keyword=公告&field=title&fuzzy=0');

// 模糊搜索標題+關鍵字
const res = await apiRequest('/search?keyword=產品&field=title|keywords&fuzzy=1');
```

---

### 6.6 留言

#### POST /messages — 提交留言（公開）

```json
// 請求
{
  "contacts": "張三",
  "mobile": "13800000000",
  "content": "留言內容"
}

// 回應
{ "code": 1, "msg": "success", "data": { "id": 43 } }
```

安全機制：
- XSS 過濾：`<script>` 等標籤自動轉義
- 敏感詞替換：含敏感詞時自動替換為 `***`
- 速率限制：同一 IP 60 秒內最多 3 次（返回 429）
- 自動採集：IP、作業系統、瀏覽器 UA

#### GET /messages?page=&pagesize=&status= — 留言列表（需認證）

```json
{
  "code": 1,
  "data": [
    {
      "id": 43,
      "contacts": "張三",
      "mobile": "13800000000",
      "content": "留言內容",
      "status": 0,
      "create_time": "2026-07-14 18:00:00",
      "user_ip": "127.0.0.1",
      "user_os": "Windows",
      "user_bs": "Chrome"
    }
  ],
  "meta": { "page": 1, "pagesize": 15, "total": 1 }
}
```

---

### 6.7 自定義表單（需認證）

#### GET /forms/:fcode/fields — 表單欄位定義

#### GET /forms/:fcode/data?page=&pagesize= — 表單數據列表

---

### 6.8 其他資源

| 端點 | 說明 | 篩選 |
|------|------|------|
| GET /slides | 幻燈片 | `gid` 分組 |
| GET /links | 友情連結 | `gid` 分組 |
| GET /tags | 標籤列表 | - |

---

## 七、前端整合建議

### 7.1 API 請求封裝

建議使用 axios 統一封裝：

```javascript
import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  withCredentials: true  // CORS credentials
});

// 請求攔截：自動注入 Token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('api_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 回應攔截：統一處理錯誤
api.interceptors.response.use(
  response => response.data,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('api_token');
      window.location.href = '/login';
    }
    if (error.response?.status === 429) {
      alert(error.response.data.msg || '請求過於頻繁');
    }
    return Promise.reject(error);
  }
);

export default api;
```

### 7.2 使用範例

```javascript
// 獲取首頁數據
const [site, nav, slides] = await Promise.all([
  api.get('/site'),
  api.get('/nav'),
  api.get('/slides?gid=1')
]);

// 獲取內容列表
const list = await api.get('/contents', {
  params: { scode: 'S10001', page: 1, pagesize: 10, order: 'date' }
});

// 獲取內容詳情（含瀏覽計數）
const detail = await api.get(`/contents/${id}`, { params: { track: 1 } });

// 提交留言
await api.post('/messages', {
  contacts: '張三',
  mobile: '13800000000',
  content: '好網站'
});
```

### 7.3 TypeScript 型別定義

```typescript
interface ApiResponse<T = any> {
  code: 0 | 1;
  msg: string;
  data: T;
  meta?: {
    page: number;
    pagesize: number;
    total: number;
  };
}

interface Content {
  id: number;
  title: string;
  subtitle: string;
  date: string;
  ico: string;
  description: string;
  content: string;
  pics: string;
  visits: number;
  scode: string;
  istop: string;
  isrecommend: string;
  url: string;
  ext: Record<string, any>;
  prev: { id: number; title: string; url: string } | null;
  next: { id: number; title: string; url: string } | null;
}

interface NavItem {
  id: number;
  scode: string;
  name: string;
  filename: string;
  urlname: string;
  sorting: number;
  children: NavItem[];
}
```

---

## 八、安全注意事項

1. **Token 儲存**：建議使用 `localStorage`，不建議存 cookie（CSRF 風險）
2. **Token 過期**：72 小時，過期前可用 `/auth/refresh` 刷新
3. **HTTPS**：生產環境必須使用 HTTPS
4. **CORS**：後台配置 `api_cors_origins` 限制允許域名（生產環境不要用 `*`）
5. **留言防刷**：同一 IP 60 秒內最多 3 條留言
6. **登入鎖定**：連續 5 次密碼錯誤鎖定 15 分鐘
7. **API Key**：伺服器間通訊用，不要暴露在前端代碼中

---

## 九、常見問題

### Q1: 列表為空時 data 返回什麼？

返回 `[]`（空陣列），不是 `null`。前端可以安全使用 `.map()`。

### Q2: prev/next 為 null 時怎麼處理？

```javascript
if (detail.prev) {
  // 有上一篇
  showPrevButton(detail.prev.url, detail.prev.title);
}
```

### Q3: 如何取得自定義欄位（ext_）？

內容詳情和列表的 `ext` 欄位包含所有自定義欄位，鍵名為 `ext_欄位名`：

```json
{ "ext": { "ext_author": "張三", "ext_price": "100" } }
```

### Q4: 搜索降級是什麼意思？

MeiliSearch 未配置或搜索失敗時，自動降級到 SQL `LIKE` 查詢，前端無需處理。

### Q5: 多語言如何切換？

API 透過 GORM AcodePlugin 自動隔離多語言資料。前端無需傳遞 `acode` 參數，後端根據請求 context 自動判斷語言。
