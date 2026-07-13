# Gbootcms 功能升級文檔

> 本文件記錄基於 AnQiCMS 對比分析後實施的功能升級，按優先級 P0/P1 分階段完成。
> 生成日期：2026-07-13

---

## 目錄

- [P0 已實現功能](#p0-已實現功能)
  - [1. MeiliSearch 全文搜索](#1-meilisearch-全文搜索)
  - [2. 301 重定向管理](#2-301-重定向管理)
  - [3. 文檔回收站（軟刪除）](#3-文檔回收站軟刪除)
  - [4. RESTful API 系統](#4-restful-api-系統)
- [P1 已實現功能](#p1-已實現功能)
  - [5. JSON-LD 結構化資料標籤系統](#5-json-ld-結構化資料標籤系統)
  - [6. 外鏈 Nofollow 自動處理](#6-外鏈-nofollow-自動處理)
  - [7. 定時發布增強](#7-定時發布增強)
  - [8. Cloudflare R2 雲存儲](#8-cloudflare-r2-雲存儲)
  - [9. 敏感詞過濾系統](#9-敏感詞過濾系統)
  - [10. LLMs.txt 自動生成](#10-llmstxt-自動生成)
- [未實現項目及原因](#未實現項目及原因)

---

## P0 已實現功能

### 1. MeiliSearch 全文搜索

**目標**：以 MeiliSearch 搜尋引擎替代 SQL LIKE 查詢，提升搜索效能與相關性排序。

**實現方案**：
- 整合 MeiliSearch Go SDK v0.36.3
- 即時同步：內容增刪改時自動同步到 MeiliSearch 索引
- 背景同步：每 5 分鐘全量同步最近更新的內容
- 降級機制：MeiliSearch 未配置或不可用時，自動降級為 SQL LIKE 查詢

**配置項**（`ay_config` 表）：

| 配置鍵 | 預設值 | 說明 |
|--------|--------|------|
| `meilisearch_url` | (空) | MeiliSearch 伺服器地址，如 `http://localhost:7700` |
| `meilisearch_key` | (空) | MeiliSearch API 金鑰 |

**使用方式**：
1. 安裝 MeiliSearch（`docker run -p 7700:7700 getmeili/meilisearch`）
2. 在後台系統配置中填入 `meilisearch_url` 和 `meilisearch_key`
3. 重啟程式，索引自動建立並同步

**相關文件**：
- `apps/api/meilisearch.go` — MeiliSearch 整合核心
- `apps/api/resources.go` — API 搜索端點（含降級邏輯）
- `apps/admin/controller/content/ContentController.go` — 內容 CRUD 同步

**索引結構**：
- 索引名稱：`gbootcms_contents`
- 可搜尋欄位：`title`, `keywords`, `description`, `content`
- 可過濾欄位：`acode`, `scode`, `status`
- 可排序欄位：`date`, `visits`

---

### 2. 301 重定向管理

**目標**：提供應用層 URL 重定向管理，補充 Nginx 靜態規則，支援後台可視化配置。

**實現方案**：
- 新建 `ay_301_redirect` 表（不改動原版表結構）
- 中介軟體攔截請求，快取重定向規則
- 支援精確匹配（MatchType=1）和前綴匹配（MatchType=2）
- 自動保留查詢字串
- 跳過 `/admin`、`/static`、`/template`、`/api` 路徑

**資料表結構**：

```sql
CREATE TABLE ay_301_redirect (
    id INTEGER PRIMARY KEY,
    old_url TEXT NOT NULL,      -- 原始 URL
    new_url TEXT NOT NULL,      -- 目標 URL
    match_type INTEGER DEFAULT 1, -- 1=精確匹配, 2=前綴匹配
    status INTEGER DEFAULT 1,   -- 1=啟用, 0=停用
    sorting INTEGER DEFAULT 0,
    create_user TEXT,
    update_user TEXT,
    create_time DATETIME,
    update_time DATETIME
);
```

**後台路由**：
- 列表：`GET /admin/content/redirect/index`
- 新增：`GET/POST /admin/content/redirect/add`
- 修改：`ANY /admin/content/redirect/mod/*action`
- 刪除：`POST /admin/content/redirect/del`

**相關文件**：
- `apps/admin/model/content/RedirectModel.go`
- `apps/admin/controller/content/RedirectController.go`
- `apps/common/middleware/redirect_301.go`
- `apps/admin/view/content/redirect.html`

**與 Nginx 的關係**：Nginx 適合處理大量靜態規則（效能更好），應用層重定向適合需要動態管理的場景。兩者互補使用。

---

### 3. 文檔回收站（軟刪除）

**目標**：內容刪除時不直接從資料庫移除，改為標記為「已刪除」（status=-1），支援復原。

**實現方案**：
- 使用 `status=-1` 標記已刪除內容（不改動表結構）
- 前台查詢 `status=1`，自動排除回收站內容
- 支援批量復原和永久刪除

**後台路由**：

| 路由 | 方法 | 說明 |
|------|------|------|
| `/admin/content/trash` | GET | 回收站列表 |
| `/admin/content/restore` | POST | 批量復原 |
| `/admin/content/permanentDel` | POST | 永久刪除 |

**狀態值對照**：

| status 值 | 含義 |
|-----------|------|
| 1 | 已發布 |
| 0 | 草稿/待審 |
| -1 | 回收站（已刪除） |

**相關文件**：
- `apps/admin/service/content/ContentService.go` — `DeleteContent`（軟刪除）、`ListTrashedContents`、`RestoreContent`、`PermanentDeleteContent`
- `apps/admin/controller/content/ContentController.go` — `Trash`、`Restore`、`PermanentDel`
- `apps/admin/view/content/trash.html`

**MeiliSearch 同步**：軟刪除時從索引移除，復原時重新加入索引。

---

### 4. RESTful API 系統

**目標**：提供標準 RESTful API，支援前端分離、小程式、APP 開發。

**認證方式**：
- **JWT Bearer Token**：`Authorization: Bearer <token>`，登入後取得
- **API Key**：`X-API-Key: <key>`，適合伺服器間通訊

**API 端點**：

### 認證

| 端點 | 方法 | 認證 | 說明 |
|------|------|------|------|
| `/api/v1/auth/login` | POST | 公開 | 管理員登入，返回 JWT |
| `/api/v1/auth/refresh` | POST | JWT | 刷新 Token |

### 資源

| 端點 | 方法 | 認證 | 說明 |
|------|------|------|------|
| `/api/v1/site` | GET | 公開 | 站點資訊 |
| `/api/v1/sorts` | GET | 公開 | 欄目列表 |
| `/api/v1/sorts/:scode` | GET | 公開 | 欄目詳情 |
| `/api/v1/contents` | GET | 公開 | 內容列表（支援分頁、篩選） |
| `/api/v1/contents/:id` | GET | 公開 | 內容詳情（含上/下一篇） |
| `/api/v1/search` | GET | 公開 | 全文搜索（MeiliSearch 優先） |
| `/api/v1/messages` | POST | 公開 | 提交留言 |
| `/api/v1/slides` | GET | 公開 | 幻燈片列表 |
| `/api/v1/links` | GET | 公開 | 友情連結列表 |
| `/api/v1/tags` | GET | 公開 | 標籤列表 |

**查詢參數**：
- `acode` — 語言代碼（預設使用系統預設語言）
- `page` — 頁頁碼（預設 1）
- `pagesize` — 每頁筆數（預設 15）
- `scode` — 欄目編號
- `mcode` — 模型編號
- `keyword` — 搜尋關鍵字
- `order` — 排序方式（date/visits/sorting）

**回應格式**：
```json
{
    "code": 1,
    "data": [...],
    "meta": {
        "page": 1,
        "pagesize": 15,
        "total": 100
    }
}
```

**相關文件**：
- `apps/api/middleware.go` — JWT + API Key 認證中介軟體
- `apps/api/auth.go` — 登入、Token 刷新
- `apps/api/resources.go` — 資源端點
- `apps/route/route.go` — `SetupAPIRoutes()`

---

## P1 已實現功能

### 5. JSON-LD 結構化資料標籤系統

**目標**：通過前台模板標籤輸出 JSON-LD 結構化資料，提升 SEO 效果。

**支援類型**：

| 類型 | 標籤語法 | 適用頁面 | 說明 |
|------|----------|----------|------|
| Article | `{gboot:jsonld type=article}` | 內容詳情頁 | 文章結構化資料 |
| BreadcrumbList | `{gboot:jsonld type=breadcrumb}` | 列表頁/詳情頁 | 麵包屑導航 |
| Organization | `{gboot:jsonld type=organization}` | 首頁/關於頁 | 組織資訊 |
| LocalBusiness | `{gboot:jsonld type=localbusiness}` | 首頁/聯繫頁 | 本地商戶（含地址、電話、營業時間） |
| WebSite | `{gboot:jsonld type=website}` | 首頁 | 網站資訊（含搜索功能） |
| FAQPage | `{gboot:jsonld type=faq}` | FAQ 頁面 | 常見問題（問答對） |
| Product | `{gboot:jsonld type=product}` | 產品詳情頁 | 產品資訊（含價格、品牌） |

**FAQ 標籤用法**：

方式一：標籤參數傳入
```
{gboot:jsonld type=faq q1="問題1" a1="答案1" q2="問題2" a2="答案2"}
```

方式二：擴展字段自動讀取（適合內容詳情頁）
- 建立 ext_faq_q_1, ext_faq_a_1, ext_faq_q_2, ext_faq_a_2 等擴展字段
- 標籤自動讀取並生成 FAQPage 結構化資料

**Product 擴展字段**：
- `ext_price` — 價格
- `ext_brand` — 品牌
- `ext_sku` — 庫存編號
- `ext_rating` — 評分

**配置項**（用於 LocalBusiness）：

| 配置鍵 | 說明 |
|--------|------|
| `opening_hours` | 營業時間，如 `Mo-Fr 09:00-18:00` |
| `geo_latitude` | 緯度 |
| `geo_longitude` | 經度 |
| `price_currency` | 貨幣代碼（預設 CNY） |

**相關文件**：
- `apps/common/parser/jsonld_provider.go` — JSON-LD 標籤實現
- `apps/common/parser/providers.go` — 標籤註冊

---

### 6. 外鏈 Nofollow 自動處理

**目標**：自動為文章內容中的外部連結添加 `rel="nofollow noopener noreferrer"` 屬性。

**實現方案**：
- 在模板標籤 `{gboot:content}` 輸出時自動處理
- 僅處理指向外部域名的 `<a>` 標籤
- 站內連結不受影響
- 已有 `rel` 屬性的標籤會補充缺失的值
- 可通過配置啟用/停用

**配置項**：

| 配置鍵 | 預設值 | 說明 |
|--------|--------|------|
| `nofollow_external` | `1` | 1=啟用, 0=停用 |

**處理邏輯**：
1. 正則匹配所有 `<a href="http://...">` 或 `<a href="https://...">` 標籤
2. 提取 href 中的域名
3. 與站點域名列表比對，站內連結跳過
4. 站外連結添加 `rel="nofollow noopener noreferrer"`
5. 已有 rel 屬性的補充缺失值

**相關文件**：
- `apps/common/parser/nofollow.go` — Nofollow 處理邏輯
- `apps/common/parser/providers.go` — 整合到 `getContentField`

---

### 7. 定時發布增強

**目標**：自動發布到期的定時內容，無需手動操作。

**實現方案**：
- 背景排程器每 60 秒檢查一次（帶 0-10 秒隨機延遲）
- 查詢條件：`status=0 AND date <= 當前時間 AND date > 零值`
- 自動更新為 `status=1`（已發布）
- 記錄發布日誌

**使用方式**：
1. 在後台新增內容時，設定未來日期
2. 將狀態設為「草稿」（status=0）
3. 到達設定時間後，排程器自動發布

**相關文件**：
- `apps/common/scheduler.go` — 排程器實現
- `main.go` — `common.InitScheduler()` 初始化

---

### 8. Cloudflare R2 雲存儲

**目標**：支援將上傳檔案同步到 Cloudflare R2，實現雲端存儲和 CDN 加速。

**實現方案**：
- 存儲抽象接口（`Storage` interface），支援本地和 R2 兩種實現
- 使用 minio-go/v7 SDK（S3 相容協議）
- 快取層：記憶體快取檔案存在性和 URL（TTL 可配置）
- 降級機制：R2 不可用時自動降級為本地存儲
- 上傳成功後返回 R2 公開 URL，失敗則返回本地路徑

**配置項**：

| 配置鍵 | 預設值 | 說明 |
|--------|--------|------|
| `r2_enabled` | `0` | 1=啟用 R2, 0=停用（使用本地） |
| `r2_endpoint` | (空) | R2 S3 端點，如 `<account_id>.r2.cloudflarestorage.com` |
| `r2_access_key` | (空) | R2 Access Key ID |
| `r2_secret_key` | (空) | R2 Secret Access Key |
| `r2_bucket` | (空) | R2 Bucket 名稱 |
| `r2_public_url` | (空) | R2 自訂域名或公開 URL（如 `https://cdn.example.com`） |
| `r2_cache_ttl` | `300` | 快取 TTL（秒） |

**快取處理**：
- 檔案存在性檢查結果快取在記憶體中（避免重複 API 呼叫）
- 快取 TTL 預設 300 秒（5 分鐘）
- 後台「清除快取」操作同時清除存儲快取
- 上傳/刪除操作自動更新快取

**使用方式**：
1. 在 Cloudflare 儀表板建立 R2 bucket
2. 建立 R2 API Token（取得 Access Key 和 Secret Key）
3. （可選）為 bucket 綁定自訂域名
4. 在後台系統配置中填入 R2 相關配置
5. 設定 `r2_enabled` 為 `1`
6. 重啟程式

**相關文件**：
- `apps/common/storage/storage.go` — 存儲接口、工廠、快取管理
- `apps/common/storage/local.go` — 本地存儲實現
- `apps/common/storage/r2.go` — R2 存儲實現
- `apps/admin/controller/IndexController.go` — 上傳處理整合

---

### 9. 敏感詞過濾系統

**目標**：對用戶提交的內容（留言、評論）進行敏感詞過濾，替換為 `***`。

**實現方案**：
- 敏感詞列表存儲在 `ay_config` 表的 `sensitive_words` 欄位
- 每行一個敏感詞
- 記憶體快取，啟動時載入
- 不區分大小寫匹配
- 應用於留言和評論的提交

**配置項**：

| 配置鍵 | 預設值 | 說明 |
|--------|--------|------|
| `sensitive_words` | (空) | 敏感詞列表，每行一個詞 |

**配置方式**：
在後台系統配置中，找到 `sensitive_words` 欄位，每行輸入一個敏感詞：
```
賭博
色情
暴力
```

**應用範圍**：
- 留言提交（`contacts` 和 `content` 欄位）
- 評論提交（`comment` 欄位）

**相關文件**：
- `apps/common/sensitive.go` — 敏感詞過濾核心
- `apps/home/controller/front.go` — 留言整合
- `apps/home/controller/comment.go` — 評論整合

---

### 10. LLMs.txt 自動生成

**目標**：生成 `/llms.txt` 文件，為 LLM 爬蟲（如 GPT、Claude）提供站點內容概覽。

**實現方案**：
- 路由：`GET /llms.txt`
- 自動生成站點資訊、主要欄目、最新內容（20 條）、聯繫資訊
- 遵循 [llmstxt.org](https://llmstxt.org/) 規範
- 快取控制：`Cache-Control: public, max-age=3600`

**文件格式**：
```
# 站點標題

> 站點描述

## 站點資訊
- 站點副標題: ...
- 站點關鍵字: ...
- 站點地址: https://example.com/

## 主要欄目
- [欄目名稱](URL): 欄目描述

## 最新內容
- [文章標題](URL): 文章摘要

## 聯繫資訊
- 公司名稱: ...
- 地址: ...
- 電話: ...
```

**相關文件**：
- `apps/home/controller/llms.go` — LLMs.txt 生成控制器
- `main.go` — 路由註冊

---

## 未實現項目及原因

| 編號 | 項目 | 原因 |
|------|------|------|
| P0 #1 | Docker 部署 | 產品完成後的錦上添花，非核心功能 |
| P1 #8 | 導入導出 | 暫無此方面需求 |
| P2 #13 | AI 整合 | 大修改，暫不處理 |
| P2 #14 | 多站點 | 與多語言邏輯不同，暫不處理 |
| P2 #15 | 後台界面升級 | 保留原 Layui 界面 |
| P2 #16 | GraphQL | 對 SQLite 意義不大，RESTful API 已足夠 |
| P2 #17 | 翻譯整合 | 已有多語言版本，翻譯是前端任務 |
| P2 #18 | 商城模組 | 大模組開發，暫不處理 |

---

## 新增依賴

| 依賴 | 版本 | 用途 |
|------|------|------|
| `github.com/golang-jwt/jwt/v5` | latest | JWT 認證 |
| `github.com/meilisearch/meilisearch-go` | latest | MeiliSearch 全文搜索 |
| `github.com/minio/minio-go/v7` | v7.2.1 | Cloudflare R2（S3 相容）存儲 |

---

## 新增文件清單

### P0 功能

| 文件 | 說明 |
|------|------|
| `apps/admin/model/content/RedirectModel.go` | 301 重定向資料模型 |
| `apps/admin/controller/content/RedirectController.go` | 301 重定向控制器 |
| `apps/common/middleware/redirect_301.go` | 301 重定向中介軟體 |
| `apps/admin/view/content/redirect.html` | 301 重定向管理頁面 |
| `apps/admin/view/content/trash.html` | 回收站頁面 |
| `apps/api/middleware.go` | API 認證中介軟體 |
| `apps/api/auth.go` | API 登入/Token |
| `apps/api/resources.go` | API 資源端點 |
| `apps/api/meilisearch.go` | MeiliSearch 整合 |

### P1 功能

| 文件 | 說明 |
|------|------|
| `apps/common/parser/jsonld_provider.go` | JSON-LD 標籤系統 |
| `apps/common/parser/nofollow.go` | 外鏈 Nofollow 處理 |
| `apps/common/scheduler.go` | 定時發布排程器 |
| `apps/common/sensitive.go` | 敏感詞過濾 |
| `apps/common/storage/storage.go` | 存儲抽象接口 |
| `apps/common/storage/local.go` | 本地存儲實現 |
| `apps/common/storage/r2.go` | R2 雲存儲實現 |
| `apps/home/controller/llms.go` | LLMs.txt 生成 |

---

## 修改文件清單

| 文件 | 修改內容 |
|------|----------|
| `main.go` | 新增排程器、敏感詞、存儲初始化；LLMs.txt 路由；URL 規範化跳過 `/llms` |
| `apps/route/route.go` | 新增回收站、301 重定向、API 路由 |
| `apps/admin/seed/seed.go` | 新增配置種子（MeiliSearch、R2、Nofollow、敏感詞、JSON-LD） |
| `apps/admin/controller/IndexController.go` | 上傳整合 R2 存儲；清除快取整合存儲快取 |
| `apps/admin/controller/content/ContentController.go` | 回收站方法；MeiliSearch 同步 |
| `apps/admin/service/content/ContentService.go` | 軟刪除、回收站查詢、復原、永久刪除 |
| `apps/admin/model/content/migrate.go` | AutoMigrate 新增 Redirect 表 |
| `apps/admin/model/db.go` | 新增 Redirect 類型別名 |
| `apps/common/parser/providers.go` | 註冊 JSON-LD 標籤；內容輸出整合 Nofollow |
| `apps/home/controller/front.go` | 留言整合敏感詞過濾 |
| `apps/home/controller/comment.go` | 評論整合敏感詞過濾 |
| `go.mod` | 新增 JWT、MeiliSearch、minio-go 依賴 |

---

## 新增配置項一覽

| 配置鍵 | 預設值 | 所屬功能 | 說明 |
|--------|--------|----------|------|
| `meilisearch_url` | (空) | 全文搜索 | MeiliSearch 伺服器地址 |
| `meilisearch_key` | (空) | 全文搜索 | MeiliSearch API 金鑰 |
| `r2_enabled` | `0` | 雲存儲 | 啟用 R2 |
| `r2_endpoint` | (空) | 雲存儲 | R2 S3 端點 |
| `r2_access_key` | (空) | 雲存儲 | R2 Access Key |
| `r2_secret_key` | (空) | 雲存儲 | R2 Secret Key |
| `r2_bucket` | (空) | 雲存儲 | R2 Bucket 名稱 |
| `r2_public_url` | (空) | 雲存儲 | R2 公開 URL |
| `r2_cache_ttl` | `300` | 雲存儲 | 快取 TTL（秒） |
| `nofollow_external` | `1` | SEO | 外鏈 Nofollow |
| `sensitive_words` | (空) | 安全 | 敏感詞列表 |
| `opening_hours` | (空) | JSON-LD | 營業時間 |
| `geo_latitude` | (空) | JSON-LD | 緯度 |
| `geo_longitude` | (空) | JSON-LD | 經度 |
| `price_currency` | `CNY` | JSON-LD | 貨幣代碼 |

---

## 新增資料表

| 表名 | 說明 |
|------|------|
| `ay_301_redirect` | 301 重定向規則表 |

> 注意：原版 PbootCMS 表結構未做任何修改，新增表使用 `ay_` 前綴保持一致性。
