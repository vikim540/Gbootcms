package middleware

import (
	"net/http"
	"gbootcms/apps/admin/model"
	"strings"

	"github.com/gin-gonic/gin"
)

// SiteRedirect HTTPS 跳轉和主域名跳轉中間件
// 對齊 PHP HomeController.__construct() 的 to_https 和 to_main_domain 邏輯
func SiteRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不跳轉（對齊 PHP 原版：後台不繼承 HomeController）
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		// HTTPS 跳轉（對齊 PHP: if (!is_https() && !!$tohttps = Config::get('to_https'))）
		if c.Request.TLS == nil && model.GetConfigValue("to_https", "0") != "0" {
			c.Redirect(http.StatusMovedPermanently, "https://"+c.Request.Host+c.Request.RequestURI)
			c.Abort()
			return
		}

		// 主域名跳轉（對齊 PHP: if (!!$main_domain && !!$to_main_domain)）
		mainDomain := model.GetConfigValue("main_domain", "")
		toMainDomain := model.GetConfigValue("to_main_domain", "0")
		if mainDomain != "" && toMainDomain != "0" {
			host := c.Request.Host
			// 去掉端口號
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if !strings.EqualFold(host, mainDomain) {
				scheme := "http://"
				if c.Request.TLS != nil {
					scheme = "https://"
				}
				c.Redirect(http.StatusMovedPermanently, scheme+mainDomain+c.Request.RequestURI)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
