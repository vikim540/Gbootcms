package parser

import (
	"regexp"
	"strings"

	"gbootcms/apps/admin/model"
)

// externalLinkRe 匹配帶有外部 href 的 <a> 標籤
var externalLinkRe = regexp.MustCompile(`(?i)<a\s+([^>]*?)href=["']https?://([^"']+)["']([^>]*)>`)

// hasRelRe 檢查標籤中是否已有 rel 屬性
var hasRelRe = regexp.MustCompile(`(?i)\brel\s*=`)

// addNofollowToExternalLinks 為外部連結自動添加 rel="nofollow noopener noreferrer"
// 僅處理指向外部域名的 <a> 標籤，站內連結不受影響
func addNofollowToExternalLinks(htmlContent string) string {
	if htmlContent == "" {
		return htmlContent
	}

	// 檢查是否啟用 nofollow（預設啟用）
	if model.GetConfigValue("nofollow_external", "1") == "0" {
		return htmlContent
	}

	// 取得站點域名列表，用於判斷是否為外部連結
	siteDomains := getSiteDomainsForNofollow()

	return externalLinkRe.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := externalLinkRe.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		hrefDomain := strings.ToLower(submatches[2])

		// 去掉端口和路徑，只保留域名部分
		if idx := strings.Index(hrefDomain, "/"); idx > 0 {
			hrefDomain = hrefDomain[:idx]
		}
		if idx := strings.Index(hrefDomain, ":"); idx > 0 {
			hrefDomain = hrefDomain[:idx]
		}

		// 站內連結不處理
		for _, d := range siteDomains {
			if hrefDomain == strings.ToLower(d) {
				return match
			}
		}

		// 已有 rel 屬性，補充缺失的值
		if hasRelRe.MatchString(match) {
			return enhanceExistingRel(match)
		}

		// 無 rel 屬性，添加完整的 rel
		return strings.Replace(match, "<a ", `<a rel="nofollow noopener noreferrer" `, 1)
	})
}

// enhanceExistingRel 在已有的 rel 屬性中補充缺失的值
func enhanceExistingRel(tag string) string {
	relRe := regexp.MustCompile(`(?i)rel=["']([^"']*)["']`)
	match := relRe.FindStringSubmatch(tag)
	if match == nil {
		return tag
	}

	values := strings.Fields(match[1])
	has := map[string]bool{}
	for _, v := range values {
		has[strings.ToLower(v)] = true
	}

	needed := []string{"nofollow", "noopener", "noreferrer"}
	for _, n := range needed {
		if !has[n] {
			values = append(values, n)
		}
	}

	newRel := `rel="` + strings.Join(values, " ") + `"`
	return relRe.ReplaceAllString(tag, newRel)
}

// getSiteDomainsForNofollow 取得所有站點域名
func getSiteDomainsForNofollow() []string {
	var sites []model.Site
	model.DB.Find(&sites)
	var domains []string
	for _, s := range sites {
		if s.Domain != "" {
			d := strings.ToLower(s.Domain)
			d = strings.TrimPrefix(d, "https://")
			d = strings.TrimPrefix(d, "http://")
			if idx := strings.Index(d, "/"); idx > 0 {
				d = d[:idx]
			}
			domains = append(domains, d)
		}
	}
	return domains
}
