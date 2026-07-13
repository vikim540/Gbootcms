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

// loadRedirectRules 從數據庫載入啟用的重定向規則
func loadRedirectRules() {
	var rules []model.Redirect
	model.DB.Where("status = 1").Order("sorting ASC, id ASC").Find(&rules)
	redirectRulesMu.Lock()
	redirectRules = rules
	redirectLoaded = true
	redirectRulesMu.Unlock()
}

// RefreshRedirectRules 刷新重定向規則快取（管理員修改後呼叫）
func RefreshRedirectRules() {
	loadRedirectRules()
}

// getRedirectRules 取得快取的重定向規則（首次存取時延遲載入）
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

// Redirect301 301 重定向中間件
// 檢查請求路徑是否匹配重定向規則，命中則返回 301 永久重定向
func Redirect301() gin.HandlerFunc {
	// 啟動後 5 秒載入規則（等待 DB 初始化完成）
	go func() {
		time.Sleep(5 * time.Second)
		loadRedirectRules()
	}()

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 跳過後台、API、靜態資源
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") ||
			strings.HasPrefix(path, "/template") || strings.HasPrefix(path, "/api") {
			c.Next()
			return
		}

		rules := getRedirectRules()
		for _, rule := range rules {
			var matched bool
			if rule.MatchType == 2 {
				// 前綴匹配
				matched = strings.HasPrefix(path, rule.OldURL)
			} else {
				// 精確匹配
				matched = path == rule.OldURL
			}

			if matched {
				target := rule.NewURL
				// 前綴匹配時，將剩餘路徑附加到目標 URL
				if rule.MatchType == 2 && len(path) > len(rule.OldURL) {
					target = strings.TrimSuffix(target, "/") + path[len(rule.OldURL):]
				}
				// 保留 query string
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
