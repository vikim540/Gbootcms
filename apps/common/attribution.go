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
