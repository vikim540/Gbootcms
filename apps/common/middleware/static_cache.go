package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticCache 靜態資源緩存中間件
// 為 /static 和 /template 開頭的靜態資源添加長期緩存頭
// 靜態資源（CSS/JS/圖片/字體）變更頻率低，長期緩存可大幅提升性能
// 使用 Cache-Control: public, max-age=31536000, immutable
// immutable 指示瀏覽器在緩存有效期內無需發送驗證請求
func StaticCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 僅對靜態資源路徑生效
		if strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/template/") {
			// 靜態資源長期緩存（1年）
			// public: 可被 CDN/代理緩存
			// max-age=31536000: 瀏覽器緩存 1 年
			// immutable: 緩存有效期內無需重新驗證
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}

		c.Next()
	}
}
