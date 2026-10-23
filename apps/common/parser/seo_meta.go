package parser

import (
	"context"
	"fmt"
	"html"
	"strings"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/system"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
)

// acodeToHreflang 將 acode 映射為標準 hreflang 代碼（ISO 639-1 + ISO 3166-1）
func acodeToHreflang(acode string) string {
	switch acode {
	case "sc":
		return "zh-Hans"
	case "tc":
		return "zh-Hant"
	case "en":
		return "en"
	default:
		return acode
	}
}

// getEffectiveDomain 獲取當前頁面生效的域名（包含協議）
func getEffectiveDomain(ctx *Context) string {
	// 如果是測試環境（如 gocms.icun.eu.org、localhost 等），強制使用當前請求的動態 Host，絕不跨域名污染生產站
	if ctx != nil && ctx.HttpURL != "" {
		host := strings.ToLower(ctx.HttpURL)
		if strings.Contains(host, ".eu.org") || strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") ||
			strings.Contains(host, ".test") || strings.Contains(host, ".local") || strings.Contains(host, ".nip.io") {
			return strings.TrimRight(ctx.HttpURL, "/")
		}
	}

	// 生產環境：優先使用站點配置的正式域名
	if ctx != nil && ctx.Site != nil && ctx.Site.Domain != "" {
		domain := ctx.Site.Domain
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		return strings.TrimRight(domain, "/")
	}

	if ctx != nil && ctx.HttpURL != "" {
		return strings.TrimRight(ctx.HttpURL, "/")
	}
	return "http://localhost:8080"
}

// buildDetailURL 構建內容頁語言相對路徑
func buildDetailURL(acode, defaultAcode string, c *content.Content) string {
	var path string
	if c.Filename != "" {
		path = "/" + c.Filename + ".html"
	} else if c.URLName != "" {
		path = "/" + c.URLName + ".html"
	} else {
		var sortPath string
		if c.Scode != "" {
			var s content.ContentSort
			if db.DB.WithContext(acodeplugin.SkipAcode(context.Background())).Where("scode = ? AND acode = ?", c.Scode, acode).First(&s).Error == nil {
				if s.Filename != "" {
					sortPath = s.Filename
				} else if s.URLName != "" {
					sortPath = s.URLName
				}
			}
		}
		if sortPath != "" {
			path = fmt.Sprintf("/%s/%d", sortPath, c.ID)
		} else {
			path = fmt.Sprintf("/content/%d", c.ID)
		}
	}

	if acode == defaultAcode {
		return path
	}
	return "/" + acode + path
}

// buildSortURLWithLang 構建欄目頁語言相對路徑
func buildSortURLWithLang(acode, defaultAcode string, s *content.ContentSort) string {
	var path string
	if s.Filename != "" {
		path = "/" + s.Filename
	} else if s.URLName != "" {
		path = "/" + s.URLName
	} else if s.Scode != "" {
		path = fmt.Sprintf("/sort/%s", s.Scode)
	} else {
		path = "/"
	}

	if acode == defaultAcode {
		return path
	}
	return "/" + acode + path
}

// findTranslatedContent 在目標語言中查找與當前文章對應的翻譯版本（精準匹配，杜絕 404）
func findTranslatedContent(reqCtx context.Context, cur *content.Content, targetAcode string) *content.Content {
	if cur == nil || targetAcode == "" {
		return nil
	}
	skipCtx := acodeplugin.SkipAcode(reqCtx)

	// 1. 若當前文章有關聯 ID (rel_id > 0)
	if cur.RelID > 0 {
		var target content.Content
		if err := db.DB.WithContext(skipCtx).Where("(rel_id = ? OR id = ?) AND acode = ? AND status = 1", cur.RelID, cur.RelID, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	} else {
		// 當前文章 rel_id == 0，嘗試查是否有其他文章將 rel_id 指向當前文章 ID
		var target content.Content
		if err := db.DB.WithContext(skipCtx).Where("rel_id = ? AND acode = ? AND status = 1", cur.ID, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}

	// 2. 自定義 URL 別名（urlname / filename）精準匹配
	if cur.URLName != "" {
		var target content.Content
		if err := db.DB.WithContext(skipCtx).Where("urlname = ? AND acode = ? AND status = 1", cur.URLName, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}
	if cur.Filename != "" {
		var target content.Content
		if err := db.DB.WithContext(skipCtx).Where("filename = ? AND acode = ? AND status = 1", cur.Filename, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}

	return nil
}

// findTranslatedSort 在目標語言中查找對應欄目
func findTranslatedSort(reqCtx context.Context, cur *content.ContentSort, targetAcode string) *content.ContentSort {
	if cur == nil || targetAcode == "" {
		return nil
	}
	skipCtx := acodeplugin.SkipAcode(reqCtx)

	var target content.ContentSort
	if cur.Scode != "" {
		if err := db.DB.WithContext(skipCtx).Where("scode = ? AND acode = ? AND status = 1", cur.Scode, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}
	if cur.Filename != "" {
		if err := db.DB.WithContext(skipCtx).Where("filename = ? AND acode = ? AND status = 1", cur.Filename, targetAcode).First(&target).Error; err == nil {
			return &target
		}
	}
	return nil
}

// buildHreflang 生成 hreflang 標籤組（自引用 + 雙向對稱 + x-default）
func buildHreflang(ctx *Context) string {
	areas := system.GetCachedAreas()
	if len(areas) <= 1 {
		return ""
	}

	domain := getEffectiveDomain(ctx)

	// 找默認區域
	defaultAcode := ""
	for _, a := range areas {
		if a.IsDefault == "1" {
			defaultAcode = a.Acode
			break
		}
	}
	if defaultAcode == "" && len(areas) > 0 {
		defaultAcode = areas[0].Acode
	}

	var sb strings.Builder
	var defaultHref string

	// 場景 1：內容詳情頁 (ctx.Content != nil)
	if ctx.Content != nil {
		curContent := ctx.Content
		curAcode := curContent.Acode

		for _, a := range areas {
			hreflangCode := acodeToHreflang(a.Acode)

			// 若是當前語言，使用當前文章自己的真實 URL
			if a.Acode == curAcode {
				link := buildDetailURL(a.Acode, defaultAcode, curContent)
				href := domain + link
				sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
				sb.WriteString("\n")
				if a.Acode == defaultAcode {
					defaultHref = href
				}
				continue
			}

			// 跨語言尋找真正對應的翻譯文章（避免 404 與偽實現）
			targetContent := findTranslatedContent(ctx.Ctx, curContent, a.Acode)
			if targetContent == nil {
				// 該語言無對應翻譯文章，依 Google 規範嚴格不輸出該語言 hreflang，避免 404 報錯
				continue
			}

			link := buildDetailURL(a.Acode, defaultAcode, targetContent)
			href := domain + link
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
			sb.WriteString("\n")
			if a.Acode == defaultAcode {
				defaultHref = href
			}
		}

		if defaultHref != "" {
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="x-default" href="%s" />`, defaultHref))
		}
		return sb.String()
	}

	// 場景 2：欄目列表頁 (ctx.Sort != nil && ctx.Content == nil)
	if ctx.Sort != nil {
		curSort := ctx.Sort
		curAcode := curSort.Acode

		for _, a := range areas {
			hreflangCode := acodeToHreflang(a.Acode)
			if a.Acode == curAcode {
				link := buildSortURLWithLang(a.Acode, defaultAcode, curSort)
				href := domain + link
				sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
				sb.WriteString("\n")
				if a.Acode == defaultAcode {
					defaultHref = href
				}
				continue
			}

			targetSort := findTranslatedSort(ctx.Ctx, curSort, a.Acode)
			if targetSort == nil {
				continue
			}
			link := buildSortURLWithLang(a.Acode, defaultAcode, targetSort)
			href := domain + link
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
			sb.WriteString("\n")
			if a.Acode == defaultAcode {
				defaultHref = href
			}
		}

		if defaultHref != "" {
			sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="x-default" href="%s" />`, defaultHref))
		}
		return sb.String()
	}

	// 場景 3：首頁 / 靜態頁
	currentPath := ctx.CurrentPath
	if currentPath == "" || currentPath == "/" {
		currentPath = "/"
	} else if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	for _, a := range areas {
		link := currentPath
		if a.Acode != defaultAcode {
			if currentPath == "/" {
				link = "/" + a.Acode + "/"
			} else {
				link = "/" + a.Acode + currentPath
			}
		}
		href := domain + link
		hreflangCode := acodeToHreflang(a.Acode)
		sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s" />`, hreflangCode, href))
		sb.WriteString("\n")
		if a.Acode == defaultAcode {
			defaultHref = href
		}
	}

	if defaultHref != "" {
		sb.WriteString(fmt.Sprintf(`<link rel="alternate" hreflang="x-default" href="%s" />`, defaultHref))
	}

	return sb.String()
}

// GetCanonicalURL 獲取當前頁面的標準 Canonical URL（共享函數，保證與 JSON-LD @id 100% 一致）
func GetCanonicalURL(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)

	currentPath := ctx.CurrentPath
	if currentPath == "" {
		currentPath = "/"
	}
	if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	// 默認區域不需要前綴
	areas := system.GetCachedAreas()
	isDefault := false
	for _, a := range areas {
		if a.Acode == acode && a.IsDefault == "1" {
			isDefault = true
			break
		}
	}

	canonicalPath := currentPath
	if !isDefault && acode != "" && !strings.HasPrefix(canonicalPath, "/"+acode+"/") && canonicalPath != "/"+acode {
		canonicalPath = "/" + acode + canonicalPath
	}

	domain := getEffectiveDomain(ctx)
	return domain + canonicalPath
}

// buildCanonical 生成 canonical 標籤，指向當前頁面的標準 URL
func buildCanonical(ctx *Context) string {
	canonicalURL := GetCanonicalURL(ctx)
	return fmt.Sprintf(`<link rel="canonical" href="%s" />`, canonicalURL)
}

// acodeToLocale 將 acode 轉換為 Open Graph locale 格式（zh_CN / zh_HK / en_US）
func acodeToLocale(acode string) string {
	switch acode {
	case "sc":
		return "zh_CN"
	case "tc":
		return "zh_HK"
	case "en":
		return "en_US"
	default:
		return acode
	}
}

// buildOpenGraph 生成 Open Graph meta 標籤（社交分享用）
func buildOpenGraph(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)

	// 站點域名（環境感知：測試站使用當前 Host，生產站使用配置域名）
	domain := getEffectiveDomain(ctx)
	siteName := ""
	if ctx.Site != nil {
		siteName = ctx.Site.Title
	}

	// 當前頁面 URL（與 canonical 相同）
	currentPath := ctx.CurrentPath
	if currentPath == "" {
		currentPath = "/"
	}
	if !strings.HasPrefix(currentPath, "/") {
		currentPath = "/" + currentPath
	}

	link := currentPath
	if acode != "" {
		areas := system.GetCachedAreas()
		isDefault := false
		for _, a := range areas {
			if a.Acode == acode && a.IsDefault == "1" {
				isDefault = true
				break
			}
		}
		if !isDefault {
			link = "/" + acode + currentPath
		}
	}

	canonicalURL := link
	if domain != "" {
		canonicalURL = domain + link
	}

	// 頁面標題和描述
	title := resolvePageTitle(ctx)
	description := resolvePageDescription(ctx)

	// 頁面類型
	ogType := "website"
	if ctx.Content != nil {
		ogType = "article"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<meta property="og:title" content="%s" />`, html.EscapeString(title)))
	sb.WriteString(fmt.Sprintf(`<meta property="og:description" content="%s" />`, html.EscapeString(description)))
	sb.WriteString(fmt.Sprintf(`<meta property="og:url" content="%s" />`, canonicalURL))
	sb.WriteString(fmt.Sprintf(`<meta property="og:type" content="%s" />`, ogType))
	if siteName != "" {
		sb.WriteString(fmt.Sprintf(`<meta property="og:site_name" content="%s" />`, html.EscapeString(siteName)))
	}
	sb.WriteString(fmt.Sprintf(`<meta property="og:locale" content="%s" />`, acodeToLocale(acode)))

	// og:locale:alternate — 其他語言
	for _, a := range system.GetCachedAreas() {
		if a.Acode != acode {
			sb.WriteString(fmt.Sprintf(`<meta property="og:locale:alternate" content="%s" />`, acodeToLocale(a.Acode)))
		}
	}

	return sb.String()
}

// resolvePageTitle 根據頁面類型讀取對應的 *_title 配置，解析巢狀標籤後返回最終標題
func resolvePageTitle(ctx *Context) string {
	var tmpl string
	if ctx.Content != nil && ctx.Sort != nil {
		// 內容頁（有欄目和內容）
		tmpl = db.GetConfigValue("content_title", "")
	} else if ctx.Sort != nil {
		// 列表/欄目頁
		tmpl = db.GetConfigValue("list_title", "")
	} else if ctx.Content != nil {
		// 單頁（有內容但無欄目）
		tmpl = db.GetConfigValue("about_title", "")
	} else {
		// 首頁
		tmpl = db.GetConfigValue("index_title", "")
	}

	if tmpl == "" {
		// 配置為空時使用預設值（對齊 PHP 各方法的預設格式）
		if ctx.Content != nil && ctx.Sort != nil {
			tmpl = "{content:title}-{sort:name}-{gboot:sitetitle}-{gboot:sitesubtitle}"
		} else if ctx.Sort != nil {
			tmpl = "{sort:name}-{gboot:sitetitle}-{gboot:sitesubtitle}"
		} else if ctx.Content != nil {
			tmpl = "{content:title}-{gboot:sitetitle}-{gboot:sitesubtitle}"
		} else {
			tmpl = "{gboot:sitetitle}-{gboot:sitesubtitle}"
		}
	}

	return resolveTitleTags(tmpl, ctx)
}

// resolvePageKeywords 根據頁面類型返回關鍵詞
func resolvePageKeywords(ctx *Context) string {
	if ctx.Content != nil && ctx.Content.Keywords != "" {
		return ctx.Content.Keywords
	}
	if ctx.Sort != nil && ctx.Sort.Keywords != "" {
		return ctx.Sort.Keywords
	}
	if ctx.Site != nil {
		return ctx.Site.Keywords
	}
	return ""
}

// resolvePageDescription 根據頁面類型返回描述
func resolvePageDescription(ctx *Context) string {
	if ctx.Content != nil && ctx.Content.Description != "" {
		return ctx.Content.Description
	}
	if ctx.Sort != nil && ctx.Sort.Description != "" {
		return ctx.Sort.Description
	}
	if ctx.Site != nil {
		return ctx.Site.Description
	}
	return ""
}

// resolveTitleTags 解析標題模板中的巢狀標籤
// 支援：{gboot:sitetitle} {gboot:sitesubtitle} {content:title} {sort:name} {sort:title} 等
func resolveTitleTags(tmpl string, ctx *Context) string {
	result := tmpl

	// 站點標籤
	if ctx.Site != nil {
		result = strings.ReplaceAll(result, "{gboot:sitetitle}", ctx.Site.Title)
		result = strings.ReplaceAll(result, "{gboot:sitesubtitle}", ctx.Site.Subtitle)
		result = strings.ReplaceAll(result, "{gboot:sitekeywords}", ctx.Site.Keywords)
		result = strings.ReplaceAll(result, "{gboot:sitedescription}", ctx.Site.Description)
		result = strings.ReplaceAll(result, "{pboot:sitetitle}", ctx.Site.Title)
		result = strings.ReplaceAll(result, "{pboot:sitesubtitle}", ctx.Site.Subtitle)
		result = strings.ReplaceAll(result, "{pboot:sitekeywords}", ctx.Site.Keywords)
		result = strings.ReplaceAll(result, "{pboot:sitedescription}", ctx.Site.Description)
	}

	// 內容標籤
	if ctx.Content != nil {
		result = strings.ReplaceAll(result, "{content:title}", ctx.Content.Title)
		result = strings.ReplaceAll(result, "{content:keywords}", ctx.Content.Keywords)
		result = strings.ReplaceAll(result, "{content:description}", ctx.Content.Description)
	}

	// 欄目標籤
	if ctx.Sort != nil {
		sortName := ctx.Sort.Name
		sortTitle := ctx.Sort.Title
		if sortTitle == "" {
			sortTitle = sortName
		}
		result = strings.ReplaceAll(result, "{sort:name}", sortName)
		result = strings.ReplaceAll(result, "{sort:title}", sortTitle)
		result = strings.ReplaceAll(result, "{sort:keywords}", ctx.Sort.Keywords)
		result = strings.ReplaceAll(result, "{sort:description}", ctx.Sort.Description)
	}

	return result
}

