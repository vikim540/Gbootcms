# SEO 與 GEO 規範配套 (SEO & GEO)

Gbootcms 為現代搜尋引擎（Google、Bing）及 AI 檢索代理（SearchGPT、Perplexity、Claude）提供了成體系、全自動、零手動維護的底層支持。

---

## 1. 動態環境感知 Robots.txt

傳統系統往往需要維護靜態 `robots.txt`，一旦測試站忘記設置 `Disallow`，就會導致測試域名被搜尋引擎收錄並判定主站「重複內容（Duplicate Content）」。

Gbootcms 代碼自動根據請求的 `Host` 感知環境：

```go
// 核心邏輯展示
func RobotsHandler(c *gin.Context) {
    host := c.Request.Host
    isProduction := (host == config.SiteDomain)

    if !isProduction {
        // 測試環境 / 預發環境：強制禁止一切爬蟲，並附加 noindex HTTP 頭
        c.Header("X-Robots-Tag", "noindex, nofollow")
        c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
        return
    }

    // 正式生產環境：放行爬蟲，並輸出 Sitemap 索引位址
    c.String(http.StatusOK, fmt.Sprintf(
        "User-agent: *\nAllow: /\nDisallow: /admin/\nDisallow: /api/\n\nSitemap: https://%s/sitemap.xml\n", 
        host,
    ))
}
```

---

## 2. 結構化資料 JSON-LD (`Schema.org`)

在前台公共頭部加入 `{gboot:schema}` 標籤，系統自動產出標準結構化資料：

```html
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      "@id": "https://example.com/#website",
      "url": "https://example.com/",
      "name": "站點名稱"
    },
    {
      "@type": "Article",
      "@id": "https://example.com/news/15.html",
      "isPartOf": { "@id": "https://example.com/#website" },
      "headline": "文章標題",
      "datePublished": "2026-10-23T08:00:00+08:00",
      "mainEntityOfPage": {
        "@type": "WebPage",
        "@id": "https://example.com/news/15.html"
      }
    }
  ]
}
</script>
```

> **嚴格一致性約束**：JSON-LD 的 `mainEntityOfPage.@id` 必須與頁面的 `<link rel="canonical">` 100% 相同，避免規範衝突。

---

## 3. 大模型檢索索引 (`/llms.txt`)

針對現代 LLM 爬蟲，提供標準 `/llms.txt` 端點：
- 列出站點主題、核心欄目摘要、主要文章與文檔索引。
- 支援 `/llms-full.txt` 提供 Markdown 格式的完整站點知識庫輸出。

---

## 4. 營銷歸因 Cookie 潔淨度

`apps/common/attribution.go` 在記錄 `g_landing_page` 與來源管道時，自動排除非前端業務路徑：
- `/robots.txt`, `/sitemap*`, `/llms.txt`
- `/api/*`, `/admin/*`, `/static/*`, `/upload/*`, `/favicon.ico`
- 確保爬蟲訪問不會在伺服器端產生無效 Cookie 或污染日誌分析。
