package middleware

import (
	"net/http"
	"gbootcms/apps/common"

	"github.com/gin-gonic/gin"
)

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
				c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "登录已过期，请重新登录"})
			} else {
				c.Redirect(http.StatusFound, "/admin/")
			}
			c.Abort()
			return
		}

		c.Set("admin_uid", uid)
		c.Set("admin_username", common.GetSession(c, "admin_username"))
		c.Set("admin_realname", common.GetSession(c, "admin_realname"))
		c.Set("admin_ucode", common.GetSession(c, "admin_ucode"))
		c.Set("admin_rcodes", common.GetSession(c, "admin_rcodes"))

		c.Next()
	}
}
