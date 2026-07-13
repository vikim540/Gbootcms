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

// publicPermPaths 免權限檢查的路徑（對齊 PbootCMS PHP $public_path）
// 使用正規化後的 "controller/action" 格式（小寫）
var publicPermPaths = map[string]bool{
	"index/index":            true,
	"index/login":            true,
	"index/home":             true,
	"index/loginout":         true,
	"index/ucenter":          true,
	"index/area":             true,
	"index/clearcache":       true,
	"index/clearonlysyscache": true,
	// clearsession 不在公開路徑中（對齊 PbootCMS PHP $public_path）
	// 清理會話需要特定權限，防止普通用戶踢出其他管理員
	"index/upload":           true,
	"index/checkcode":        true,
}

// reverseRouteMap Go 路由前綴 → PbootCMS 原版前綴的反向映射
// 用於權限檢查時將 Go 路由正規化為與 ay_role_level.level 相同的格式
var reverseRouteMap = map[string]string{
	"/admin/system/config":      "/admin/config",
	"/admin/content/label":      "/admin/label",
	"/admin/content/model":      "/admin/model",
	"/admin/content/extfield":   "/admin/extfield",
	"/admin/content/site":       "/admin/site",
	"/admin/content/company":    "/admin/company",
	"/admin/content/sort":       "/admin/contentsort",
	"/admin/content/single":     "/admin/single",
	"/admin/content/message":    "/admin/message",
	"/admin/content/slide":      "/admin/slide",
	"/admin/content/link":       "/admin/link",
	"/admin/content/form":       "/admin/form",
	"/admin/content/tags":       "/admin/tags",
	"/admin/content/media":      "/admin/media",
	"/admin/member/group":       "/admin/membergroup",
	"/admin/member/field":       "/admin/memberfield",
	"/admin/member/comment":     "/admin/membercomment",
	"/admin/system/area":        "/admin/area",
	"/admin/system/menu":        "/admin/menu",
	"/admin/system/role":        "/admin/role",
	"/admin/system/user":        "/admin/user",
	"/admin/system/syslog":      "/admin/syslog",
	"/admin/system/database":    "/admin/database",
}

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
				c.JSON(http.StatusOK, gin.H{"code": 0, "data": "登入已過期，請重新登入", "msg": "登入已過期，請重新登入"})
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
						c.JSON(http.StatusOK, gin.H{"code": 0, "data": "表單驗證失敗，請重新整理頁面後再試", "msg": "表單驗證失敗，請重新整理頁面後再試"})
						c.Abort()
						return
					}
				}
			}
		}

		// URL 級權限攔截（對齊 PbootCMS PHP AdminController::checkLevel()）
		// 超級管理員(uid=1)跳過，其他用戶檢查當前 URL 是否在 session levels 中
		if uid != 1 {
			// 取得原始 URL（重寫前的 PbootCMS 格式），用於權限比對
			permPath := path
			if origPath := c.GetHeader("X-Original-Path"); origPath != "" {
				permPath = origPath
			}
			normalized := normalizeForPerm(permPath)

			if !publicPermPaths[normalized] {
				levelsRaw := common.GetSession(c, "levels")
				var levels []string
				if l, ok := levelsRaw.([]string); ok {
					levels = l
				}

				permitted := false
				// 構建已授權的正規化 URL 集合
				levelSet := make(map[string]bool)
				for _, lvl := range levels {
					levelSet[normalizeForPerm(lvl)] = true
				}

				if levelSet[normalized] {
					permitted = true
				} else if !permitted {
					// 非標準操作（list/detail/mark/clean/refresh/backup/restore 等）
					// 如果用戶有該控制器的 index 瀏覽權限，則允許存取
					// 這對齊 PbootCMS PHP 的行為：非 CRUD 操作附屬於瀏覽權限
					parts := strings.SplitN(normalized, "/", 2)
					if len(parts) == 2 {
						action := parts[1]
						standardActions := map[string]bool{
							"add": true, "mod": true, "del": true, "index": true,
						}
						if !standardActions[action] {
							if levelSet[parts[0]+"/index"] {
								permitted = true
							}
						}
					}
				}

				if !permitted {
					if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
						c.JSON(http.StatusOK, gin.H{"code": 0, "data": "您的帳號權限不足，您無法執行該操作", "msg": "您的帳號權限不足，您無法執行該操作"})
					} else {
						c.Header("Content-Type", "text/html; charset=utf-8")
						c.String(http.StatusForbidden, "您的帳號權限不足，您無法執行該操作")
					}
					c.Abort()
					return
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

// normalizeForPerm 將 URL 正規化為 "controller/action" 格式（小寫）用於權限比對
// 處理 PbootCMS 原版 URL 和 Go 路由兩種格式
// 例：
//   /admin/Role/mod/rcode/R102     → role/mod
//   /admin/system/role/mod/rcode/R102 → role/mod（反向映射 system/role → role）
//   /admin/ContentSort/index       → contentsort/index
//   /admin/content/sort/index      → contentsort/index（反向映射 content/sort → contentsort）
func normalizeForPerm(path string) string {
	lower := strings.ToLower(path)
	lower = strings.TrimPrefix(lower, "/admin/")

	// 反向映射 Go 路由前綴到 PbootCMS 格式
	for goRoute, pbootRoute := range reverseRouteMap {
		goPrefix := strings.TrimPrefix(goRoute, "/admin/")
		// 確保匹配在段邊界（後面跟 / 或剛好是完整匹配）
		if strings.HasPrefix(lower, goPrefix+"/") {
			pbootPrefix := strings.TrimPrefix(pbootRoute, "/admin/")
			lower = pbootPrefix + lower[len(goPrefix):]
			break
		}
		if lower == goPrefix {
			lower = strings.TrimPrefix(pbootRoute, "/admin/")
			break
		}
	}

	// 取前兩段（controller/action）
	parts := strings.SplitN(lower, "/", 3)
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return lower
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
