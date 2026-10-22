package middleware

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// Tracking 商業廣告與獲客歸因中介軟體
// 全天候捕捉 Google Ads / Meta / 搜索引擎等商業投放 UTM 標籤與 GCLID，
// 自動在訪客端持久化 30 天 Cookie，保證跨頁瀏覽後表單轉化鏈路 100% 歸因不丟單。
func Tracking() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 忽略後台、API與靜態資源
		if strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/admin") ||
			strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/template") {
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
