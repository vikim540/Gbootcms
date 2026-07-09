package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"gbootcms/apps/admin/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// cacheBodyWriter 包裝 ResponseWriter 以捕獲響應體
type cacheBodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *cacheBodyWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// HTMLCache 動態頁面緩存中間件（對齊 PHP View::cache 邏輯）
// 當 tpl_html_cache=1 時，快取前台 HTML 響應到 runtime/cache/ 目錄
// 帶 p（pathinfo）或 s（搜索）參數的請求不快取（對齊 PHP: !query_string('p,s')）
func HTMLCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台和靜態資源不快取
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		// 檢查快取開關（對齊 PHP: Config::get('tpl_html_cache')）
		if model.GetConfigValue("tpl_html_cache", "0") == "0" {
			c.Next()
			return
		}

		// 帶 p 或 s 參數的請求不快取（對齊 PHP: !query_string('p,s')）
		if c.Query("p") != "" || c.Query("s") != "" {
			c.Next()
			return
		}

		// 計算快取檔名（對齊 PHP: md5(get_http_url() . REQUEST_URI . lg . wap)）
		cacheKey := fmt.Sprintf("%s%s%s", c.Request.Host, c.Request.RequestURI, c.GetHeader("Cookie"))
		cacheFile := filepath.Join("runtime", "cache", fmt.Sprintf("%x.html", md5.Sum([]byte(cacheKey))))

		// 檢查快取是否存在且未過期
		if info, err := os.Stat(cacheFile); err == nil {
			cacheTime, _ := time.ParseDuration(model.GetConfigValue("tpl_html_cache_time", "900") + "s")
			if cacheTime <= 0 {
				cacheTime = 900 * time.Second
			}
			if time.Since(info.ModTime()) < cacheTime {
				// 快取命中，直接返回（對齊 PHP: Kernel.php 快取讀取邏輯）
				data, err := os.ReadFile(cacheFile)
				if err == nil {
					c.Header("Content-Type", "text/html; charset=utf-8")
					c.String(http.StatusOK, string(data))
					c.Abort()
					return
				}
			}
		}

		// 快取未命中，繼續處理請求並捕獲響應
		cw := &cacheBodyWriter{
			ResponseWriter: c.Writer,
			buf:            &bytes.Buffer{},
		}
		c.Writer = cw

		c.Next()

		// 請求處理完成，判斷是否為 HTML 響應
		contentType := cw.Header().Get("Content-Type")
		if strings.HasPrefix(contentType, "text/html") && cw.Status() == http.StatusOK {
			// 寫入快取檔案（對齊 PHP: file_put_contents($cacheFile, $content)）
			os.MkdirAll(filepath.Dir(cacheFile), 0755)
			os.WriteFile(cacheFile, cw.buf.Bytes(), 0644)
		}

		// 輸出實際響應
		cw.ResponseWriter.WriteHeader(cw.Status())
		cw.ResponseWriter.Write(cw.buf.Bytes())
	}
}
