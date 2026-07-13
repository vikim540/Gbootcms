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
// 注意：只對前綴（控制器名）做大小寫歸一化，保留路徑參數的原始大小寫
// （如 rcode=R102 不會被轉成 r102）。
// 未在映射表中的前綴（如 /admin/content、/admin/member 等原生 Go 路由）
// 也會將控制器部分轉為小寫，以匹配 Gin 的大小寫敏感路由。
func RewriteAdminPath(path string) string {
	lower := strings.ToLower(path)
	if !strings.HasPrefix(lower, "/admin/") {
		return path
	}

	// 用小寫版本匹配前綴（控制器名大小寫不敏感）
	lowerParts := strings.SplitN(lower, "/", 4)
	if len(lowerParts) < 3 {
		return path
	}
	prefix := "/admin/" + lowerParts[2]

	target, ok := pbootToGoRouteMap[prefix]
	if !ok {
		// 未在映射表中：可能是原生 Go 路由（如 /admin/content/index）
		// 將控制器部分轉為小寫以匹配 Gin 路由，保留其餘路徑參數的原始大小寫
		origParts := strings.SplitN(path, "/", 4)
		remaining := ""
		if len(origParts) >= 4 {
			remaining = "/" + origParts[3]
		}
		result := "/admin/" + lowerParts[2] + remaining
		if result == path {
			return path // 已經是小寫，無需重寫
		}
		return result
	}

	// 用原始路徑提取剩餘部分（保留參數大小寫）
	origParts := strings.SplitN(path, "/", 4)
	remaining := ""
	if len(origParts) >= 4 {
		remaining = "/" + origParts[3]
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
