package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"gbootcms/apps/admin/model"

	"github.com/gin-gonic/gin"
)

// 301 重定向規則快取
var (
	redirectRules   []model.Redirect
	redirectRulesMu sync.RWMutex
	redirectLoaded  bool
)

func loadRedirectRules() {
	var rules []model.Redirect
	if model.DB != nil {
		model.DB.Where("status = 1").Order("sorting ASC, id ASC").Find(&rules)
	}
	redirectRulesMu.Lock()
	redirectRules = rules
	redirectLoaded = true
	redirectRulesMu.Unlock()
}

// RefreshRedirectRules 刷新重定向規則快取（管理員修改後呼叫）
func RefreshRedirectRules() {
	loadRedirectRules()
}

func getRedirectRules() []model.Redirect {
	redirectRulesMu.RLock()
	if !redirectLoaded {
		redirectRulesMu.RUnlock()
		loadRedirectRules()
		redirectRulesMu.RLock()
	}
	rules := redirectRules
	redirectRulesMu.RUnlock()
	return rules
}

// RedirectMiddleware 合併處理 HTTPS 跳轉、主域名跳轉與自定義 301 重定向規則（單次遍歷）
func RedirectMiddleware() gin.HandlerFunc {
	// 啟動後延遲載入 301 自定義規則
	go func() {
		time.Sleep(3 * time.Second)
		loadRedirectRules()
	}()

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 後台、API、靜態資源不參與前台跳轉
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/template") || strings.HasPrefix(path, "/api") {
			c.Next()
			return
		}

		// 1. HTTPS 強制跳轉
		if c.Request.TLS == nil && model.GetConfigValue("to_https", "0") != "0" {
			c.Redirect(http.StatusMovedPermanently, "https://"+c.Request.Host+c.Request.RequestURI)
			c.Abort()
			return
		}

		// 2. 主域名強制跳轉
		mainDomain := model.GetConfigValue("main_domain", "")
		toMainDomain := model.GetConfigValue("to_main_domain", "0")
		if mainDomain != "" && toMainDomain != "0" {
			host := c.Request.Host
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

		// 3. 數據庫 301 自定義路徑匹配
		rules := getRedirectRules()
		for _, rule := range rules {
			var matched bool
			if rule.MatchType == 2 {
				matched = strings.HasPrefix(path, rule.OldURL)
			} else {
				matched = path == rule.OldURL
			}

			if matched {
				target := rule.NewURL
				if rule.MatchType == 2 && len(path) > len(rule.OldURL) {
					target = strings.TrimSuffix(target, "/") + path[len(rule.OldURL):]
				}
				if c.Request.URL.RawQuery != "" {
					if strings.Contains(target, "?") {
						target += "&" + c.Request.URL.RawQuery
					} else {
						target += "?" + c.Request.URL.RawQuery
					}
				}
				c.Redirect(http.StatusMovedPermanently, target)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// 兼容舊別名
func SiteRedirect() gin.HandlerFunc {
	return RedirectMiddleware()
}

func Redirect301() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
