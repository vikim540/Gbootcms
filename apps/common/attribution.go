package common

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// LeadAttribution 商業獲客營銷歸因模型
type LeadAttribution struct {
	UtmSource   string `json:"utm_source"`
	UtmMedium   string `json:"utm_medium"`
	UtmCampaign string `json:"utm_campaign"`
	UtmTerm     string `json:"utm_term"`
	UtmContent  string `json:"utm_content"`
	Gclid       string `json:"gclid"`
	LandingPage string `json:"landing_page"`
	Referrer    string `json:"referrer"`
}

// GetLeadAttribution 提取全鏈路商業營銷歸因（優先順序：POST Form > URL Query > 追蹤 Cookie > HTTP Header）
func GetLeadAttribution(c *gin.Context) LeadAttribution {
	getVal := func(keys ...string) string {
		for _, k := range keys {
			// 1. POST body
			if v := strings.TrimSpace(c.PostForm(k)); v != "" {
				return FilterUserInput(v)
			}
			// 2. URL Query
			if v := strings.TrimSpace(c.Query(k)); v != "" {
				return FilterUserInput(v)
			}
			// 3. Cookie (前綴 g_ 或直接名稱)
			if v, err := c.Cookie("g_" + k); err == nil && strings.TrimSpace(v) != "" {
				if unescaped, err := url.QueryUnescape(v); err == nil {
					return FilterUserInput(unescaped)
				}
				return FilterUserInput(v)
			}
			if v, err := c.Cookie(k); err == nil && strings.TrimSpace(v) != "" {
				if unescaped, err := url.QueryUnescape(v); err == nil {
					return FilterUserInput(unescaped)
				}
				return FilterUserInput(v)
			}
		}
		return ""
	}

	landing := getVal("landing_page")
	if landing == "" {
		landing = c.Request.RequestURI
	}

	ref := getVal("referrer", "referer")
	if ref == "" {
		ref = c.Request.Referer()
	}

	return LeadAttribution{
		UtmSource:   getVal("utm_source"),
		UtmMedium:   getVal("utm_medium"),
		UtmCampaign: getVal("utm_campaign"),
		UtmTerm:     getVal("utm_term"),
		UtmContent:  getVal("utm_content"),
		Gclid:       getVal("gclid"),
		LandingPage: landing,
		Referrer:    ref,
	}
}

// TrackingMiddleware 商業廣告與獲客歸因中間件
// 全天候捕捉 Google Ads / Meta / 搜索引擎等商業投放 UTM 標籤與 GCLID，
// 自動在訪客端持久化 30 天 Cookie，保證跨頁瀏覽後表單轉化鏈路 100% 歸因不丟單。
func TrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 忽略後台、API、靜態資源、系統與爬蟲端點（防止 g_landing_page 等 Cookie 污染系統文件）
		if strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/admin") ||
			strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/template") ||
			strings.HasPrefix(path, "/upload") ||
			path == "/robots.txt" ||
			strings.HasPrefix(path, "/sitemap") ||
			path == "/llms.txt" ||
			path == "/llms-full.txt" ||
			path == "/favicon.ico" {
			c.Next()
			return
		}

		q := c.Request.URL.Query()
		utmParams := []string{"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "gclid"}

		for _, param := range utmParams {
			if val := strings.TrimSpace(q.Get(param)); val != "" {
				// 存儲 30 天 Cookie (2592000 秒)
				c.SetCookie("g_"+param, url.QueryEscape(val), 2592000, "/", "", false, false)
			}
		}

		// 著陸頁首跳記錄（首次進入頁面時寫入）
		if _, err := c.Cookie("g_landing_page"); err != nil && c.Request.Method == "GET" {
			landingURL := c.Request.RequestURI
			if landingURL != "" {
				c.SetCookie("g_landing_page", url.QueryEscape(landingURL), 2592000, "/", "", false, false)
			}
		}

		// 外來來源頁 Referrer 記錄（若無記錄且非本站域名）
		if _, err := c.Cookie("g_referrer"); err != nil {
			if ref := c.Request.Referer(); ref != "" {
				if parsed, err := url.Parse(ref); err == nil && parsed.Host != "" && parsed.Host != c.Request.Host {
					c.SetCookie("g_referrer", url.QueryEscape(ref), 2592000, "/", "", false, false)
				}
			}
		}

		c.Next()
	}
}
