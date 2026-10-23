package controller

import (	"gbootcms/apps/admin/model/system"
	"gbootcms/apps/admin/model/content"
	"gbootcms/core/db"

	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gbootcms/apps/common"
	"gbootcms/core/acodeplugin"

	"github.com/gin-gonic/gin"
)

// ── XML 結構定義 ──

// sitemapURLset 對應 <urlset> 根元素
type sitemapURLset struct {
	XMLName xml.Name      `xml:"urlset"`
	XMLNS   string        `xml:"xmlns,attr"`
	URLs    []sitemapURL  `xml:"url"`
}

// sitemapURL 對應 <url> 元素
type sitemapURL struct {
	Loc        string  `xml:"loc"`
	Lastmod    string  `xml:"lastmod,omitempty"`
	Changefreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// sitemapIndex 對應 <sitemapindex> 根元素
type sitemapIndex struct {
	XMLName xml.Name        `xml:"sitemapindex"`
	XMLNS   string          `xml:"xmlns,attr"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

// sitemapEntry 對應 <sitemap> 元素
type sitemapEntry struct {
	Loc     string `xml:"loc"`
	Lastmod string `xml:"lastmod,omitempty"`
}

// Sitemap 處理 sitemap.xml 請求
// /sitemap.xml → sitemap 索引（列出所有語言的 sitemap）
// /sitemap-{acode}.xml → 指定語言的 sitemap
func (fc *FrontController) Sitemap(c *gin.Context) {
	// 優先從路由參數獲取 acode（/sitemap-:acode.xml）
	requestedAcode := c.Param("acode")

	if requestedAcode == "" {
		// 生成 sitemap 索引
		fc.sitemapIndexHandler(c)
	} else {
		// 生成指定語言的 sitemap
		fc.sitemapLangHandler(c, requestedAcode)
	}
}

// sitemapIndexHandler 生成 sitemap 索引文件
func (fc *FrontController) sitemapIndexHandler(c *gin.Context) {
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())

	// 查所有區域
	var areas []system.Area
	db.DB.WithContext(skipCtx).Find(&areas)

	// 構建域名基礎
	domain := fc.getDomain(c)

	// 查最新內容更新時間作為 lastmod
	var latestContent content.Content
	latestTime := time.Now()
	if err := db.DB.WithContext(skipCtx).Order("update_time DESC").First(&latestContent).Error; err == nil {
		if !latestContent.UpdateTime.IsZero() {
			latestTime = latestContent.UpdateTime
		}
	}

	index := sitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for _, a := range areas {
		index.Sitemaps = append(index.Sitemaps, sitemapEntry{
			Loc:     domain + "/sitemap-" + a.Acode + ".xml",
			Lastmod: latestTime.Format("2006-01-02"),
		})
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	xmlData, _ := xml.MarshalIndent(index, "", "  ")
	c.String(http.StatusOK, xml.Header+string(xmlData))
}

// SitemapLang 生成指定語言的 sitemap（供 NoRoute 處理器調用）
func (fc *FrontController) SitemapLang(c *gin.Context, acode string) {
	fc.sitemapLangHandler(c, acode)
}

// sitemapLangHandler 生成指定語言的 sitemap
func (fc *FrontController) sitemapLangHandler(c *gin.Context, acode string) {
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())

	// 驗證 acode 是否合法
	var area system.Area
	if err := db.DB.WithContext(skipCtx).Where("acode = ?", acode).First(&area).Error; err != nil {
		c.String(http.StatusNotFound, "404")
		return
	}

	// 設定當前請求的 acode context（供查詢使用）
	ctx := acodeplugin.WithAcode(c.Request.Context(), acode)

	// 構建域名基礎 + 語言前綴
	domain := fc.getDomain(c)
	// 判斷是否為默認語言（默認語言不加前綴）
	isDefault := area.IsDefault == "1"
	prefix := ""
	if !isDefault {
		prefix = "/" + acode
	}

	urlset := sitemapURLset{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	// 1. 首頁
	urlset.URLs = append(urlset.URLs, sitemapURL{
		Loc:        domain + prefix + "/",
		Changefreq: "daily",
		Priority:   1.0,
	})

	// 2. 所有欄目頁（status=1）
	var sorts []content.ContentSort
	db.DB.WithContext(ctx).Where("status = 1").Order("sorting ASC").Find(&sorts)
	for _, s := range sorts {
		sortURL := buildSortURL(&s)
		if sortURL != "" {
			urlset.URLs = append(urlset.URLs, sitemapURL{
				Loc:        domain + prefix + sortURL,
				Changefreq: "weekly",
				Priority:   0.8,
			})
		}
	}

	// 3. 所有內容頁（status=1）
	var contents []content.Content
	db.DB.WithContext(ctx).Where("status = 1").Order("id ASC").Find(&contents)
	for _, ct := range contents {
		contentURL := buildSitemapContentURL(ctx, &ct)
		if contentURL != "" {
			lastmod := ""
			if !ct.UpdateTime.IsZero() {
				lastmod = ct.UpdateTime.Format("2006-01-02")
			} else if !ct.CreateTime.IsZero() {
				lastmod = ct.CreateTime.Format("2006-01-02")
			}
			urlset.URLs = append(urlset.URLs, sitemapURL{
				Loc:        domain + prefix + contentURL,
				Lastmod:    lastmod,
				Changefreq: "monthly",
				Priority:   0.6,
			})
		}
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	xmlData, _ := xml.MarshalIndent(urlset, "", "  ")
	c.String(http.StatusOK, xml.Header+string(xmlData))
}

// Robots 生成 robots.txt（符合 RFC 9309 規範）
func (fc *FrontController) Robots(c *gin.Context) {
	// 0. 環境感知防收錄：若當前為測試站、預覽站或本地開發，強制封鎖爬蟲，避免污染主站搜尋引擎權重
	if common.IsStagingHost(c) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("X-Robots-Tag", "noindex, nofollow, noarchive")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
		return
	}

	skipCtx := acodeplugin.SkipAcode(c.Request.Context())

	// 1. 若後台自定義了 robots_content 則優先使用
	customRobots := strings.TrimSpace(db.GetConfigValue("robots_content", ""))
	if customRobots != "" {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=3600")
		c.String(http.StatusOK, customRobots)
		return
	}

	// 2. 留空時全自動動態生成
	var sb strings.Builder
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")
	sb.WriteString("Disallow: /admin/\n")
	sb.WriteString("Disallow: /api/\n")
	sb.WriteString("\n")

	// 添加所有語言的 sitemap 引用（符合 sitemaps.org 規範，嚴格僅聲明 XML 地圖）
	domain := fc.getDomain(c)
	var areas []system.Area
	db.DB.WithContext(skipCtx).Find(&areas)
	for _, a := range areas {
		sb.WriteString(fmt.Sprintf("Sitemap: %s/sitemap-%s.xml\n", domain, a.Acode))
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, sb.String())
}

// getDomain 獲取站點域名（含協議）
func (fc *FrontController) getDomain(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	// 若為測試站，絕不跨域名綁定正式生產域名，嚴格使用當前請求域名
	if common.IsStagingHost(c) {
		return scheme + "://" + c.Request.Host
	}

	domain := scheme + "://" + c.Request.Host

	// 如果後台配置了生產域名，使用配置的域名（更穩定）
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())
	var site content.Site
	if err := db.DB.WithContext(skipCtx).First(&site).Error; err == nil && site.Domain != "" {
		configuredDomain := site.Domain
		if !strings.HasPrefix(configuredDomain, "http://") && !strings.HasPrefix(configuredDomain, "https://") {
			configuredDomain = "https://" + configuredDomain
		}
		// 去除尾部斜線
		configuredDomain = strings.TrimSuffix(configuredDomain, "/")
		domain = configuredDomain
	}

	return domain
}

// buildSortURL 構建欄目 URL（不含域名和語言前綴）
func buildSortURL(s *content.ContentSort) string {
	if s.Filename != "" {
		return "/" + s.Filename
	}
	if s.URLName != "" {
		return "/" + s.URLName
	}
	if s.Scode != "" {
		return fmt.Sprintf("/sort/%s", s.Scode)
	}
	return ""
}

// buildSitemapContentURL 構建內容 URL（不含域名和語言前綴）
func buildSitemapContentURL(ctx context.Context, c *content.Content) string {
	if c.Outlink != "" {
		return "" // 外鏈不納入 sitemap
	}

	// 短路徑模式
	if db.GetConfigValue("url_rule_content_path", "0") == "1" {
		if c.Filename != "" {
			return "/" + c.Filename
		}
		if c.URLName != "" {
			return "/" + c.URLName
		}
		return fmt.Sprintf("/content/%d", c.ID)
	}

	// 帶欄目路徑模式（預設）
	var sortPath string
	if c.Scode != "" {
		var s content.ContentSort
		if db.DB.WithContext(ctx).Where("scode = ?", c.Scode).First(&s).Error == nil {
			if s.Filename != "" {
				sortPath = s.Filename
			} else if s.URLName != "" {
				sortPath = s.URLName
			}
		}
	}

	if c.Filename != "" {
		if sortPath != "" {
			return "/" + sortPath + "/" + c.Filename
		}
		return "/" + c.Filename
	}
	if c.URLName != "" {
		if sortPath != "" {
			return "/" + sortPath + "/" + c.URLName
		}
		return "/" + c.URLName
	}
	if sortPath != "" {
		return fmt.Sprintf("/%s/%d", sortPath, c.ID)
	}
	return fmt.Sprintf("/content/%d", c.ID)
}
