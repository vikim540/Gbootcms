package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 商業級安全響應頭中間件
// 對齊 OWASP Secure Headers Project 建議
// 僅作用於前台頁面（text/html），API 和靜態資源由各自邏輯處理
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止點擊劫持：頁面不允許被嵌入 iframe
		// 後台頁面建議用 SAMEORIGIN，前台用 DENY 更安全
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// 防止 MIME 類型嗅探：瀏覽器必須使用 Content-Type 聲明的類型
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS 防護（現代瀏覽器已內建，但保留對舊瀏覽器的兼容）
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer 策略：跨域請求僅發送 origin，保護敏感 URL 參數
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 權限策略：禁用不需要的瀏覽器 API（麥克風、攝像頭、地理位置等）
		c.Header("Permissions-Policy", "microphone=(), camera=(), geolocation=(), payment=()")

		// 內容安全策略（CSP）：寬鬆模式，兼容 Layui jQuery 和內聯樣式
		// default-src 限制所有資源來源；script-src 允許內聯和 eval（Layui 依賴）
		// style-src 允許內聯樣式；img-src 允許 data: URI（Layui 圖標）
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")

		c.Next()
	}
}
