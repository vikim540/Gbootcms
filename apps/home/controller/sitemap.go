package controller

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gbootcms/apps/admin/model"
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
	var areas []model.Area
	model.DB.WithContext(skipCtx).Find(&areas)

	// 構建域名基礎
	domain := fc.getDomain(c)

	// 查最新內容更新時間作為 lastmod
	var latestContent model.Content
	latestTime := time.Now()
	if err := model.DB.WithContext(skipCtx).Order("update_time DESC").First(&latestContent).Error; err == nil {
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
	var area model.Area
	if err := model.DB.WithContext(skipCtx).Where("acode = ?", acode).First(&area).Error; err != nil {
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
	var sorts []model.ContentSort
	model.DB.WithContext(ctx).Where("status = 1").Order("sorting ASC").Find(&sorts)
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
	var contents []model.Content
	model.DB.WithContext(ctx).Where("status = 1").Order("id ASC").Find(&contents)
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

// Robots 生成 robots.txt
func (fc *FrontController) Robots(c *gin.Context) {
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())

	var sb strings.Builder
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")
	sb.WriteString("Disallow: /admin/\n")
	sb.WriteString("Disallow: /api/\n")
	sb.WriteString("\n")

	// 添加所有語言的 sitemap 引用
	domain := fc.getDomain(c)
	var areas []model.Area
	model.DB.WithContext(skipCtx).Find(&areas)
	for _, a := range areas {
		sb.WriteString(fmt.Sprintf("Sitemap: %s/sitemap-%s.xml\n", domain, a.Acode))
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, sb.String())
}

// getDomain 獲取站點域名（含協議）
func (fc *FrontController) getDomain(c *gin.Context) string {
	// 優先使用請求的協議和 Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	// 如果配置了反向代理，檢查 X-Forwarded-Proto
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	domain := scheme + "://" + c.Request.Host

	// 如果後台配置了域名，使用配置的域名（更穩定）
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())
	var site model.Site
	if err := model.DB.WithContext(skipCtx).First(&site).Error; err == nil && site.Domain != "" {
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
func buildSortURL(s *model.ContentSort) string {
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
func buildSitemapContentURL(ctx context.Context, c *model.Content) string {
	if c.Outlink != "" {
		return "" // 外鏈不納入 sitemap
	}

	// 短路徑模式
	if model.GetConfigValue("url_rule_content_path", "0") == "1" {
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
		var s model.ContentSort
		if model.DB.WithContext(ctx).Where("scode = ?", c.Scode).First(&s).Error == nil {
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
