package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// pbootToGoRouteMap PbootCMS 原版 URL 前缀 → Go 版实际路由前缀
// 设计思路：保持原版 PHP 模板和菜单的 URL 100% 不变，由该映射表
// 在 NoRoute 兜底时将原版 URL 重写到 Go 路由树上的实际路径，再用
// gin.HandleContext 重新分发，从而避免在 route.go 中堆叠数百个别名。
var pbootToGoRouteMap = map[string]string{
	// 全局配置（M156）
	"/admin/config":   "/admin/system/config",
	"/admin/label":    "/admin/content/label",
	"/admin/model":    "/admin/content/model",
	"/admin/extfield": "/admin/content/extField",

	// 基础内容（M110）
	"/admin/site":        "/admin/content/site",
	"/admin/company":     "/admin/content/company",
	"/admin/contentsort": "/admin/content/sort",

	// 文章内容（M130）
	"/admin/single": "/admin/content/single",
	// /admin/content/* 已经是 Go 路由原生路径，无需映射

	// 扩展内容（M157）
	"/admin/message": "/admin/content/message",
	"/admin/slide":   "/admin/content/slide",
	"/admin/link":    "/admin/content/link",
	"/admin/form":    "/admin/content/form",
	"/admin/tags":    "/admin/content/tags",
	"/admin/media":   "/admin/content/media",

	// 会员中心（M1001）
	"/admin/membergroup":   "/admin/member/group",
	"/admin/memberfield":   "/admin/member/field",
	"/admin/membercomment": "/admin/member/comment",
	// /admin/member 已经是 Go 路由原生路径

	// 系统管理（M101）
	"/admin/area":     "/admin/system/area",
	"/admin/menu":     "/admin/system/menu",
	"/admin/role":     "/admin/system/role",
	"/admin/user":     "/admin/system/user",
	"/admin/syslog":   "/admin/system/syslog",
	"/admin/type":     "/admin/system/type",
	"/admin/database": "/admin/system/database",
	"/admin/imageext": "/admin/system/imageExt",

	// 顶级菜单占位（/admin/Mxxx/index）→ 重定向到首页
	"/admin/m156":  "/admin/index/home",
	"/admin/m110":  "/admin/index/home",
	"/admin/m130":  "/admin/index/home",
	"/admin/m157":  "/admin/index/home",
	"/admin/m1001": "/admin/index/home",
	"/admin/m101":  "/admin/index/home",
}

// RewriteAdminPath 把 PbootCMS 原版 admin URL 重写到 Go 版实际路由路径
// 例: /admin/Config/index → /admin/system/config/index
// 不匹配时原样返回。
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

// PathRewrite 中间件保留供未来场景使用（如果将来想在所有匹配前重写）
func PathRewrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.URL.Path = RewriteAdminPath(c.Request.URL.Path)
		c.Next()
	}
}
