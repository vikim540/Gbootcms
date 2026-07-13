package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/core/acodeplugin"
)

// registerJSONLDProvider 註冊 JSON-LD 結構化資料標籤
// 支援類型：article, breadcrumb, organization, localbusiness, website, faq, product
// 用法：{gboot:jsonld type=article}
func registerJSONLDProvider(p *TagParser, ctx *Context) {
	p.Register("jsonld", func(tagName string, params map[string]string, inner string) string {
		ldType := params["type"]
		if ldType == "" {
			ldType = "article"
		}

		var data interface{}
		switch ldType {
		case "article":
			data = buildArticleJSONLD(ctx)
		case "breadcrumb":
			data = buildBreadcrumbJSONLD(ctx)
		case "organization":
			data = buildOrganizationJSONLD(ctx)
		case "localbusiness":
			data = buildLocalBusinessJSONLD(ctx)
		case "website":
			data = buildWebsiteJSONLD(ctx)
		case "faq":
			data = buildFAQJSONLD(ctx, params)
		case "product":
			data = buildProductJSONLD(ctx)
		default:
			return ""
		}

		if data == nil {
			return ""
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return ""
		}

		return fmt.Sprintf(`<script type="application/ld+json">%s</script>`, string(jsonBytes))
	})
}

// getBaseURL 取得站點基礎 URL
func getBaseURL(ctx *Context) string {
	if ctx.Site != nil && ctx.Site.Domain != "" {
		domain := ctx.Site.Domain
		if !strings.HasPrefix(domain, "http") {
			domain = "https://" + domain
		}
		return strings.TrimSuffix(domain, "/")
	}
	httpurl := model.GetConfigValue("httpurl", "")
	if httpurl != "" && httpurl != "/" {
		if !strings.HasPrefix(httpurl, "http") {
			httpurl = "https://" + httpurl
		}
		return strings.TrimSuffix(httpurl, "/")
	}
	return ""
}

// getLogoURL 取得 Logo 完整 URL
func getLogoURL(ctx *Context) string {
	if ctx.Site == nil || ctx.Site.Logo == "" {
		return ""
	}
	logo := ctx.Site.Logo
	if strings.HasPrefix(logo, "http") {
		return logo
	}
	base := getBaseURL(ctx)
	if base != "" {
		return base + "/" + strings.TrimPrefix(logo, "/")
	}
	return logo
}

// getContentURL 構建內容頁 URL
func getContentURL(ct *model.Content) string {
	if ct.Outlink != "" {
		return ct.Outlink
	}
	if ct.Filename != "" {
		return "/" + ct.Filename + ".html"
	}
	if ct.URLName != "" {
		return "/" + ct.URLName + ".html"
	}
	return fmt.Sprintf("/content/%d.html", ct.ID)
}

// getSortURL 構建欄目 URL
func getSortURL(s *model.ContentSort) string {
	if s.Filename != "" {
		return "/" + s.Filename
	}
	if s.URLName != "" {
		return "/" + s.URLName
	}
	return fmt.Sprintf("/sort/%s", s.Scode)
}

// buildSortChainForJSONLD 從當前欄目向上遞歸到根欄目
func buildSortChainForJSONLD(ctx *Context, scode string) []model.ContentSort {
	var chain []model.ContentSort
	current := scode
	for current != "" && current != "0" {
		var s model.ContentSort
		if err := model.DB.WithContext(ctx.Ctx).Where("scode = ?", current).First(&s).Error; err != nil {
			break
		}
		chain = append(chain, s)
		current = s.Pcode
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

// formatDateISO 格式化為 ISO 8601 日期
func formatDateISO(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// buildArticleJSONLD 文章結構化資料
func buildArticleJSONLD(ctx *Context) interface{} {
	if ctx.Content == nil {
		return nil
	}

	ct := ctx.Content
	base := getBaseURL(ctx)
	url := getContentURL(ct)
	if base != "" && !strings.HasPrefix(url, "http") {
		url = base + url
	}

	// 取得首圖
	image := ""
	if ct.Ico != "" {
		image = ct.Ico
		if !strings.HasPrefix(image, "http") && base != "" {
			image = base + "/" + strings.TrimPrefix(image, "/")
		}
	} else if ct.Pics != "" {
		pics := strings.Split(ct.Pics, ",")
		if len(pics) > 0 {
			image = pics[0]
			if !strings.HasPrefix(image, "http") && base != "" {
				image = base + "/" + strings.TrimPrefix(image, "/")
			}
		}
	}

	author := ct.Author
	if author == "" {
		if ctx.Site != nil {
			author = ctx.Site.Title
		}
	}

	article := map[string]interface{}{
		"@context":         "https://schema.org",
		"@type":            "Article",
		"headline":         ct.Title,
		"description":      ct.Description,
		"datePublished":    formatDateISO(ct.Date),
		"dateModified":     formatDateISO(ct.UpdateTime),
		"author":           map[string]interface{}{"@type": "Person", "name": author},
		"publisher":        buildPublisherRef(ctx),
		"mainEntityOfPage": map[string]interface{}{"@type": "WebPage", "@id": url},
	}
	if image != "" {
		article["image"] = image
	}
	if ct.Keywords != "" {
		article["keywords"] = ct.Keywords
	}
	if ct.Source != "" {
		article["articleSection"] = ct.Source
	}
	return article
}

// buildPublisherRef 構建發布者引用（用於 Article 的 publisher 欄位）
func buildPublisherRef(ctx *Context) interface{} {
	if ctx.Site == nil {
		return nil
	}
	pub := map[string]interface{}{
		"@type": "Organization",
		"name":  ctx.Site.Title,
	}
	logo := getLogoURL(ctx)
	if logo != "" {
		pub["logo"] = map[string]interface{}{
			"@type": "ImageObject",
			"url":   logo,
		}
	}
	return pub
}

// buildBreadcrumbJSONLD 麵包屑結構化資料
func buildBreadcrumbJSONLD(ctx *Context) interface{} {
	if ctx.Sort == nil {
		return nil
	}

	base := getBaseURL(ctx)
	chain := buildSortChainForJSONLD(ctx, ctx.Sort.Scode)

	items := []map[string]interface{}{
		{
			"@type": "ListItem",
			"position": 1,
			"name":     resolveHomeName(ctx),
			"item":     base + "/",
		},
	}

	for i, s := range chain {
		sortURL := getSortURL(&s)
		if base != "" {
			sortURL = base + sortURL
		}
		items = append(items, map[string]interface{}{
			"@type":    "ListItem",
			"position": i + 2,
			"name":     s.Name,
			"item":     sortURL,
		})
	}

	// 如果有內容，加上內容標題
	if ctx.Content != nil && ctx.Content.Title != "" {
		contentURL := getContentURL(ctx.Content)
		if base != "" {
			contentURL = base + contentURL
		}
		items = append(items, map[string]interface{}{
			"@type":    "ListItem",
			"position": len(items) + 1,
			"name":     ctx.Content.Title,
			"item":     contentURL,
		})
	}

	return map[string]interface{}{
		"@context":        "https://schema.org",
		"@type":           "BreadcrumbList",
		"itemListElement": items,
	}
}

// resolveHomeName 取得當前語言的首頁名稱
func resolveHomeName(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)
	switch acode {
	case "sc":
		return "首页"
	case "en":
		return "Home"
	default:
		return "首頁"
	}
}

// buildOrganizationJSONLD 組織結構化資料
func buildOrganizationJSONLD(ctx *Context) interface{} {
	if ctx.Site == nil {
		return nil
	}

	org := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     ctx.Site.Title,
		"url":      getBaseURL(ctx) + "/",
	}

	logo := getLogoURL(ctx)
	if logo != "" {
		org["logo"] = logo
	}

	if ctx.Site.Description != "" {
		org["description"] = ctx.Site.Description
	}

	if ctx.Company != nil {
		if ctx.Company.Email != "" {
			org["email"] = ctx.Company.Email
		}
		if ctx.Company.Phone != "" {
			org["telephone"] = ctx.Company.Phone
		}
		if ctx.Company.Fax != "" {
			org["faxNumber"] = ctx.Company.Fax
		}
		if ctx.Company.Address != "" {
			org["address"] = map[string]interface{}{
				"@type":         "PostalAddress",
				"streetAddress": ctx.Company.Address,
			}
			if ctx.Company.Postcode != "" {
				org["address"].(map[string]interface{})["postalCode"] = ctx.Company.Postcode
			}
		}
		// 社交連結可以從 Other 欄位或配置中取得
		if ctx.Company.Other != "" {
			// other 欄位可能包含 JSON 或逗號分隔的 URL
			urls := strings.Split(ctx.Company.Other, ",")
			var sameAs []string
			for _, u := range urls {
				u = strings.TrimSpace(u)
				if strings.HasPrefix(u, "http") {
					sameAs = append(sameAs, u)
				}
			}
			if len(sameAs) > 0 {
				org["sameAs"] = sameAs
			}
		}
	}

	return org
}

// buildLocalBusinessJSONLD 本地商戶結構化資料
func buildLocalBusinessJSONLD(ctx *Context) interface{} {
	if ctx.Site == nil || ctx.Company == nil {
		return nil
	}

	biz := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "LocalBusiness",
		"name":     ctx.Company.Name,
		"url":      getBaseURL(ctx) + "/",
	}

	if ctx.Company.Address != "" {
		addr := map[string]interface{}{
			"@type":         "PostalAddress",
			"streetAddress": ctx.Company.Address,
		}
		if ctx.Company.Postcode != "" {
			addr["postalCode"] = ctx.Company.Postcode
		}
		biz["address"] = addr
	}
	if ctx.Company.Phone != "" {
		biz["telephone"] = ctx.Company.Phone
	}
	if ctx.Company.Mobile != "" {
		biz["telephone"] = ctx.Company.Mobile
	}
	if ctx.Company.Email != "" {
		biz["email"] = ctx.Company.Email
	}
	if ctx.Company.Fax != "" {
		biz["faxNumber"] = ctx.Company.Fax
	}
	if ctx.Company.Legal != "" {
		biz["founder"] = map[string]interface{}{
			"@type": "Person",
			"name":  ctx.Company.Legal,
		}
	}
	if ctx.Company.Blicense != "" {
		biz["identifier"] = ctx.Company.Blicense
	}

	logo := getLogoURL(ctx)
	if logo != "" {
		biz["logo"] = logo
		biz["image"] = logo
	}

	if ctx.Site.Description != "" {
		biz["description"] = ctx.Site.Description
	}

	// 營業時間（可從配置讀取）
	openingHours := model.GetConfigValue("opening_hours", "")
	if openingHours != "" {
		biz["openingHours"] = openingHours
	}

	// 地理座標（可從配置讀取）
	geoLat := model.GetConfigValue("geo_latitude", "")
	geoLng := model.GetConfigValue("geo_longitude", "")
	if geoLat != "" && geoLng != "" {
		biz["geo"] = map[string]interface{}{
			"@type":     "GeoCoordinates",
			"latitude":  geoLat,
			"longitude": geoLng,
		}
	}

	return biz
}

// buildWebsiteJSONLD 網站結構化資料
func buildWebsiteJSONLD(ctx *Context) interface{} {
	if ctx.Site == nil {
		return nil
	}

	site := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     ctx.Site.Title,
		"url":      getBaseURL(ctx) + "/",
	}

	if ctx.Site.Subtitle != "" {
		site["alternateName"] = ctx.Site.Subtitle
	}
	if ctx.Site.Description != "" {
		site["description"] = ctx.Site.Description
	}

	// 搜索功能
	searchURL := getBaseURL(ctx) + "/search?keyword={search_term_string}"
	site["potentialAction"] = map[string]interface{}{
		"@type":       "SearchAction",
		"target":      searchURL,
		"query-input": "required name=search_term_string",
	}

	return site
}

// buildFAQJSONLD FAQ 結構化資料
// 從擴展字段讀取問答對：ext_faq_q_1/ext_faq_a_1, ext_faq_q_2/ext_faq_a_2, ...
// 或從標籤參數讀取：{gboot:jsonld type=faq q1="問題1" a1="答案1" q2="問題2" a2="答案2"}
func buildFAQJSONLD(ctx *Context, params map[string]string) interface{} {
	var entities []map[string]interface{}

	// 方式1：從標籤參數讀取 q1/a1, q2/a2, ...
	for i := 1; i <= 50; i++ {
		q := params[fmt.Sprintf("q%d", i)]
		a := params[fmt.Sprintf("a%d", i)]
		if q == "" || a == "" {
			break
		}
		entities = append(entities, map[string]interface{}{
			"@type":          "Question",
			"name":           q,
			"acceptedAnswer": map[string]interface{}{"@type": "Answer", "text": a},
		})
	}

	// 方式2：從擴展字段讀取
	if len(entities) == 0 && ctx.ContentExt != nil {
		for i := 1; i <= 50; i++ {
			qKey := fmt.Sprintf("ext_faq_q_%d", i)
			aKey := fmt.Sprintf("ext_faq_a_%d", i)
			q, qok := ctx.ContentExt[qKey].(string)
			a, aok := ctx.ContentExt[aKey].(string)
			if !qok || !aok || q == "" || a == "" {
				break
			}
			entities = append(entities, map[string]interface{}{
				"@type":          "Question",
				"name":           q,
				"acceptedAnswer": map[string]interface{}{"@type": "Answer", "text": a},
			})
		}
	}

	if len(entities) == 0 {
		return nil
	}

	return map[string]interface{}{
		"@context":           "https://schema.org",
		"@type":              "FAQPage",
		"mainEntity":         entities,
	}
}

// buildProductJSONLD 產品結構化資料
// 從內容和擴展字段讀取產品信息：ext_price（價格）, ext_brand（品牌）, ext_sku（庫存編號）等
func buildProductJSONLD(ctx *Context) interface{} {
	if ctx.Content == nil {
		return nil
	}

	ct := ctx.Content
	base := getBaseURL(ctx)
	url := getContentURL(ct)
	if base != "" {
		url = base + url
	}

	product := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Product",
		"name":        ct.Title,
		"description": ct.Description,
		"url":         url,
	}

	// 產品圖片
	image := ""
	if ct.Ico != "" {
		image = ct.Ico
	} else if ct.Pics != "" {
		pics := strings.Split(ct.Pics, ",")
		if len(pics) > 0 {
			image = pics[0]
		}
	}
	if image != "" {
		if !strings.HasPrefix(image, "http") && base != "" {
			image = base + "/" + strings.TrimPrefix(image, "/")
		}
		product["image"] = image
	}

	// 從擴展字段讀取產品屬性
	if ctx.ContentExt != nil {
		if price, ok := ctx.ContentExt["ext_price"].(string); ok && price != "" {
			product["offers"] = map[string]interface{}{
				"@type":         "Offer",
				"price":         price,
				"priceCurrency": model.GetConfigValue("price_currency", "CNY"),
				"availability":  "https://schema.org/InStock",
			}
		}
		if brand, ok := ctx.ContentExt["ext_brand"].(string); ok && brand != "" {
			product["brand"] = map[string]interface{}{
				"@type": "Brand",
				"name":  brand,
			}
		}
		if sku, ok := ctx.ContentExt["ext_sku"].(string); ok && sku != "" {
			product["sku"] = sku
		}
		if rating, ok := ctx.ContentExt["ext_rating"].(string); ok && rating != "" {
			product["aggregateRating"] = map[string]interface{}{
				"@type":       "AggregateRating",
				"ratingValue": rating,
				"reviewCount": "1",
			}
		}
	}

	return product
}
