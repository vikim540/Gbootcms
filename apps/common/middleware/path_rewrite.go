package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// pbootToGoRouteMap PbootCMS 原版 URL 前綴 → Go 版實際路由前綴
// 設計思路：保持原版 PHP 模板和選單的 URL 100% 不變，由該映射表
// 在 NoRoute 兜底時將原版 URL 重寫到 Go 路由樹上的實際路徑，再用
// gin.HandleContext 重新分發，從而避免在 route.go 中堆疊數百個別名。
var pbootToGoRouteMap = map[string]string{
	// 全局配置（M156）
	"/admin/config":   "/admin/system/config",
	"/admin/label":    "/admin/content/label",
	"/admin/model":    "/admin/content/model",
	"/admin/extfield": "/admin/content/extField",

	// 基礎內容（M110）
	"/admin/site":        "/admin/content/site",
	"/admin/company":     "/admin/content/company",
	"/admin/contentsort": "/admin/content/sort",

	// 文章內容（M130）
	"/admin/single": "/admin/content/single",
	// /admin/content/* 已經是 Go 路由原生路徑，無需映射

	// 擴展內容（M157）
	"/admin/message": "/admin/content/message",
	"/admin/slide":   "/admin/content/slide",
	"/admin/link":    "/admin/content/link",
	"/admin/form":    "/admin/content/form",
	"/admin/tags":    "/admin/content/tags",
	"/admin/media":   "/admin/content/media",

	// 會員中心（M1001）
	"/admin/membergroup":   "/admin/member/group",
	"/admin/memberfield":   "/admin/member/field",
	"/admin/membercomment": "/admin/member/comment",
	// /admin/member 已經是 Go 路由原生路徑

	// 系統管理（M101）
	"/admin/area":     "/admin/system/area",
	"/admin/menu":     "/admin/system/menu",
	"/admin/role":     "/admin/system/role",
	"/admin/user":     "/admin/system/user",
	"/admin/syslog":   "/admin/system/syslog",
	"/admin/database": "/admin/system/database",

	// 頂級選單佔位（/admin/Mxxx/index）→ 重定向到首頁
	"/admin/m156":  "/admin/index/home",
	"/admin/m110":  "/admin/index/home",
	"/admin/m130":  "/admin/index/home",
	"/admin/m157":  "/admin/index/home",
	"/admin/m1001": "/admin/index/home",
	"/admin/m101":  "/admin/index/home",
}

// RewriteAdminPath 把 PbootCMS 原版 admin URL 重寫到 Go 版實際路由路徑
// 例: /admin/Config/index → /admin/system/config/index
// 不匹配時原樣返回。
func RewriteAdminPath(path string) string {
	lower := strings.ToLower(path)
	if !strings.HasPrefix(lower, "/admin/") {
		return path
	}

	parts := strings.SplitN(lower, "/", 4)
	if len(parts) < 3 {
		return path
	}
	prefix := "/admin/" + parts[2]

	target, ok := pbootToGoRouteMap[prefix]
	if !ok {
		return path
	}

	remaining := ""
	if len(parts) >= 4 {
		remaining = "/" + parts[3]
	}
	return target + remaining
}

// PathRewrite 中間件保留供未來場景使用（如果將來想在所有匹配前重寫）
func PathRewrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.URL.Path = RewriteAdminPath(c.Request.URL.Path)
		c.Next()
	}
}
