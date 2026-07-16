package middleware

import (
	"net/http"
	"gbootcms/apps/admin/model"
	"strings"

	"github.com/gin-gonic/gin"
)

// SiteStatus 關站檢查中間件（對齊 PHP HomeController.__construct 的 close_site 邏輯）
// close_site=1 時，前台請求返回 503 + 關站提示；後台/static/template 不受影響
func SiteStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不受關站影響（對齊 PHP: 後台不繼承 HomeController）
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		// 關站檢查（對齊 PHP: if (!!$close_site = Config::get('close_site'))）
		if model.GetConfigValue("close_site", "0") != "0" {
			closeSiteNote := model.GetConfigValue("close_site_note", "")
			if closeSiteNote == "" {
				closeSiteNote = "本站維護中，請稍後再訪問，帶來不便，敬請諒解！"
			}
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusServiceUnavailable, `<!DOCTYPE html>
<html lang="zh-TW">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>網站維護中</title></head>
<body style="text-align:center;padding:50px;font-family:sans-serif;">
<h2>%s</h2>
</body>
</html>`, closeSiteNote)
			c.Abort()
			return
		}

		c.Next()
	}
}

// isMobileUA 判斷是否為手機端用戶（對齊 PHP is_mobile() 邏輯）
func isMobileUA(ua string) bool {
	ua = strings.ToLower(ua)
	return strings.Contains(ua, "android") ||
		strings.Contains(ua, "iphone") ||
		strings.Contains(ua, "ipad") ||
		strings.Contains(ua, "windows phone")
}

// MobileSwitch 手機版切換中間件（對齊 PHP HomeController.__construct 的 open_wap 邏輯）
func MobileSwitch() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不處理
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		openWap := model.GetConfigValue("open_wap", "0")
		if openWap == "0" {
			c.Next()
			return
		}

		wapDomain := model.GetConfigValue("wap_domain", "")
		host := c.Request.Host
		// 去掉端口號
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}

		ua := c.Request.UserAgent()
		isMobile := isMobileUA(ua)

		if wapDomain != "" && wapDomain == host {
			// 已在手機域名，設置手機版模板標記
			c.Set("is_wap", true)
		} else if isMobile && wapDomain != "" && wapDomain != host {
			// 手機訪問但域名不一致，302 跳轉到手機域名（對齊 PHP: header Location 302）
			scheme := "http://"
			if c.Request.TLS != nil {
				scheme = "https://"
			}
			c.Redirect(http.StatusFound, scheme+wapDomain+c.Request.RequestURI)
			c.Abort()
			return
		} else if isMobile {
			// 手機訪問但未綁域名，設置手機版模板標記
			c.Set("is_wap", true)
		}

		c.Next()
	}
}

// SpiderLog 蜘蛛訪問記錄中間件（對齊 PHP SpiderController 邏輯）
// 識別爬蟲 UA，記錄到日誌文件（log/YYYYMMDD.log）
func SpiderLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不記錄
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		// 檢查是否開啟蜘蛛記錄（對齊 PHP: spiderlog !== '0'）
		if model.GetConfigValue("spiderlog", "0") != "0" {
			ua := strings.ToLower(c.Request.UserAgent())
			spider := identifySpider(ua)
			if spider != "" {
				// 加入佇列由 worker pool 批量寫入，不阻塞請求
				enqueueSpiderLog(spider, c.Request.URL.String(), c.ClientIP())
			}
		}

		c.Next()
	}
}

// identifySpider 識別爬蟲 UA（對齊 PHP SpiderController::getSpider，25 種蜘蛛）
func identifySpider(ua string) string {
	switch {
	case strings.Contains(ua, "googlebot"):
		return "Google"
	case strings.Contains(ua, "baiduspider"):
		return "Baidu"
	case strings.Contains(ua, "bingbot"):
		return "Bing"
	case strings.Contains(ua, "360spider"):
		return "360So"
	case strings.Contains(ua, "sogou"):
		return "Sogou"
	case strings.Contains(ua, "yandex"):
		return "Yandex"
	case strings.Contains(ua, "bytespider"):
		return "ByteSpider"
	case strings.Contains(ua, "applebot"):
		return "Apple"
	case strings.Contains(ua, "petalbot"):
		return "Petal"
	case strings.Contains(ua, "ahrefsbot"):
		return "Ahrefs"
	case strings.Contains(ua, "semrush"):
		return "Semrush"
	case strings.Contains(ua, "dotbot"):
		return "DotBot"
	case strings.Contains(ua, "mj12bot"):
		return "MJ12"
	case strings.Contains(ua, "amazonbot"):
		return "Amazon"
	case strings.Contains(ua, "yisouspider"):
		return "Yisou"
	case strings.Contains(ua, "spider"):
		return "other-spider"
	case strings.Contains(ua, "bot"):
		return "other-bot"
	}
	return ""
}


