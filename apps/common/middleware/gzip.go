package middleware

import (
	"bytes"
	"compress/gzip"
	"gbootcms/apps/admin/model"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

// compressBodyWriter 包裝 ResponseWriter 以捕獲響應體
type compressBodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *compressBodyWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// Compress 頁面壓縮中間件（Brotli 優先，Gzip 回退，對齊 PHP Controller::gzip 邏輯）
// 當 gzip 配置開啟時，根據瀏覽器 Accept-Encoding 選擇最佳壓縮算法：
// - Brotli：壓縮率比 Gzip 高 15-25%，特別適合 CSS/JS/HTML
// - Gzip：通用回退方案
func Compress() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不壓縮
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		// 檢查壓縮配置（對齊 PHP: Config::get('gzip')）
		if model.GetConfigValue("gzip", "0") == "0" {
			c.Next()
			return
		}

		acceptEncoding := c.GetHeader("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") && !strings.Contains(acceptEncoding, "br") {
			c.Next()
			return
		}

		cw := &compressBodyWriter{
			ResponseWriter: c.Writer,
			buf:            &bytes.Buffer{},
		}
		c.Writer = cw

		c.Next()

		// 請求處理完成後，判斷 Content-Type 決定是否壓縮
		contentType := cw.Header().Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/") &&
			!strings.Contains(contentType, "javascript") &&
			!strings.Contains(contentType, "json") &&
			!strings.Contains(contentType, "xml") {
			// 非文本類型，原樣輸出
			cw.ResponseWriter.WriteHeader(cw.Status())
			cw.ResponseWriter.Write(cw.buf.Bytes())
			return
		}

		originalBody := cw.buf.Bytes()
		statusCode := cw.Status()

		// Brotli 優先（壓縮率更高，特別適合 CSS/JS）
		if strings.Contains(acceptEncoding, "br") {
			var compressed bytes.Buffer
			writer := brotli.NewWriterLevel(&compressed, brotli.BestCompression)
			writer.Write(originalBody)
			writer.Close()

			cw.Header().Set("Content-Encoding", "br")
			cw.Header().Set("Vary", "Accept-Encoding")
			cw.Header().Del("Content-Length")
			cw.ResponseWriter.WriteHeader(statusCode)
			cw.ResponseWriter.Write(compressed.Bytes())
			return
		}

		// Gzip 回退
		if strings.Contains(acceptEncoding, "gzip") {
			var compressed bytes.Buffer
			writer, _ := gzip.NewWriterLevel(&compressed, 6)
			writer.Write(originalBody)
			writer.Close()

			cw.Header().Set("Content-Encoding", "gzip")
			cw.Header().Set("Vary", "Accept-Encoding")
			cw.Header().Del("Content-Length")
			cw.ResponseWriter.WriteHeader(statusCode)
			cw.ResponseWriter.Write(compressed.Bytes())
			return
		}

		// 不壓縮，原樣輸出
		cw.ResponseWriter.WriteHeader(statusCode)
		cw.ResponseWriter.Write(originalBody)
	}
}
