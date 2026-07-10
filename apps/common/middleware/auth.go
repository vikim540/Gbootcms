package middleware

import (
	"net/http"
	"strings"
	"gbootcms/apps/common"

	"github.com/gin-gonic/gin"
)

// csrfExemptPaths 不需要 CSRF 校驗的完整路徑（上傳等無法注入 formcheck 的端點）
var csrfExemptPaths = map[string]bool{
	"/admin/index/upload":                true,
	"/admin/index/upload/watermark/1":    true,
	"/admin/index/upload/watermark/0":    true,
}

// csrfExemptControllers 不需要 CSRF 校驗的控制器前綴
var csrfExemptControllers = map[string]bool{}

func AdminAuth() gin.HandlerFunc {
	publicPaths := map[string]bool{
		"/admin/":                 true,
		"/admin/index/index":      true,
		"/admin/index/login":      true,
		"/admin/index/checkCode":  true,
		"/admin/index/checkcode":  true,
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if publicPaths[path] || path == "/admin" || path == "/admin/" {
			c.Next()
			return
		}

		uid := common.GetSessionInt(c, "admin_uid")
		if uid == 0 {
			if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "登入已過期，請重新登入"})
			} else {
				c.Redirect(http.StatusFound, "/admin/")
			}
			c.Abort()
			return
		}

		// CSRF 校驗：POST 請求檢查 formcheck token（對齊 PbootCMS PHP AdminController）
		if c.Request.Method == "POST" {
			if !csrfExemptPaths[path] {
				controller := extractControllerFromPath(path)
				if !csrfExemptControllers[controller] {
					if !common.VerifyFormcheck(c) {
						c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "表單驗證失敗，請重新整理頁面後再試"})
						c.Abort()
						return
					}
				}
			}
		}

		c.Set("admin_uid", uid)
		c.Set("admin_username", common.GetSession(c, "admin_username"))
		c.Set("admin_realname", common.GetSession(c, "admin_realname"))
		c.Set("admin_ucode", common.GetSession(c, "admin_ucode"))
		c.Set("admin_rcodes", common.GetSession(c, "admin_rcodes"))

		c.Next()
	}
}

// extractControllerFromPath 從 /admin/xxx/yyy 中提取控制器名稱 xxx（小寫）
func extractControllerFromPath(path string) string {
	lower := strings.ToLower(path)
	lower = strings.TrimPrefix(lower, "/admin/")
	parts := strings.SplitN(lower, "/", 2)
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "index"
}
