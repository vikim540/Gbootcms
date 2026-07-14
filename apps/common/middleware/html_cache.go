package middleware

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"

	"github.com/gin-gonic/gin"
)

// --- 記憶體緩存層（永遠開啟，無需配置） ---

type memCacheEntry struct {
	content  []byte
	expireAt int64
}

var (
	memCache   sync.Map // key: cacheKey(string), value: memCacheEntry
	memCacheMu sync.RWMutex
)

// ClearHTMLCache 清除所有 HTML 緩存（記憶體 + 檔案）
// 後台發布/編輯/刪除內容時呼叫
func ClearHTMLCache() {
	// 清除記憶體緩存
	memCache.Range(func(key, value interface{}) bool {
		memCache.Delete(key)
		return true
	})

	// 清除檔案緩存
	cacheDir := filepath.Join("runtime", "cache")
	entries, err := os.ReadDir(cacheDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".html") {
				os.Remove(filepath.Join(cacheDir, entry.Name()))
			}
		}
	}
}

// cacheBodyWriter 包裝 ResponseWriter 以捕獲響應體
type cacheBodyWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *cacheBodyWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// HTMLCache 動態頁面緩存中間件
// 記憶體緩存永遠開啟（TTL 由 tpl_html_cache_time 控制，預設 900 秒）
// 檔案緩存由 tpl_html_cache 配置項控制（跨重啟持久化）
// 帶 p（pathinfo）或 s（搜索）參數的請求不快取
// 已登入會員不快取（避免個人化內容被快取導致資訊洩露）
func HTMLCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 後台、API、靜態資源不快取
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		// 帶 p 或 s 參數的請求不快取
		if c.Query("p") != "" || c.Query("s") != "" {
			c.Next()
			return
		}

		// 已登入會員不快取（安全：避免個人化內容洩露給其他用戶）
		if uid := common.GetSessionInt(c, "pboot_uid"); uid > 0 {
			c.Next()
			return
		}

		// 快取 TTL
		cacheTTL := 900 * time.Second
		if v := model.GetConfigValue("tpl_html_cache_time", "900"); v != "" {
			if sec, err := time.ParseDuration(v + "s"); err == nil && sec > 0 {
				cacheTTL = sec
			}
		}

		// 計算快取鍵
		cacheKey := fmt.Sprintf("%s%s", c.Request.Host, c.Request.RequestURI)
		cacheKeyHash := fmt.Sprintf("%x", md5.Sum([]byte(cacheKey)))
		now := time.Now().Unix()

		// --- 第一層：記憶體緩存（永遠開啟，亞毫秒級） ---
		if entry, ok := memCache.Load(cacheKeyHash); ok {
			e := entry.(memCacheEntry)
			if now < e.expireAt {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.Header("X-Cache", "HIT-MEM")
				c.Data(http.StatusOK, "text/html; charset=utf-8", e.content)
				c.Abort()
				return
			}
			// 過期，刪除
			memCache.Delete(cacheKeyHash)
		}

		// --- 第二層：檔案緩存（由 tpl_html_cache 控制開關） ---
		fileCacheEnabled := model.GetConfigValue("tpl_html_cache", "0") == "1"
		cacheFile := filepath.Join("runtime", "cache", cacheKeyHash+".html")

		if fileCacheEnabled {
			if info, err := os.Stat(cacheFile); err == nil {
				if time.Since(info.ModTime()) < cacheTTL {
					data, err := os.ReadFile(cacheFile)
					if err == nil {
						// 同時寫入記憶體緩存（下次命中走記憶體，無磁碟 I/O）
						memCache.Store(cacheKeyHash, memCacheEntry{
							content:  data,
							expireAt: now + int64(cacheTTL.Seconds()),
						})
						c.Header("Content-Type", "text/html; charset=utf-8")
						c.Header("X-Cache", "HIT-FILE")
						c.Data(http.StatusOK, "text/html; charset=utf-8", data)
						c.Abort()
						return
					}
				}
			}
		}

		// --- 快取未命中，繼續處理請求並捕獲響應 ---
		cw := &cacheBodyWriter{
			ResponseWriter: c.Writer,
			buf:            &bytes.Buffer{},
		}
		c.Writer = cw

		c.Next()

		// 請求處理完成，判斷是否為 HTML 響應
		contentType := cw.Header().Get("Content-Type")
		if strings.HasPrefix(contentType, "text/html") && cw.Status() == http.StatusOK {
			body := cw.buf.Bytes()

			// 寫入記憶體緩存（永遠執行）
			memCache.Store(cacheKeyHash, memCacheEntry{
				content:  body,
				expireAt: now + int64(cacheTTL.Seconds()),
			})

			// 寫入檔案緩存（僅在開啟時）
			if fileCacheEnabled {
				os.MkdirAll(filepath.Dir(cacheFile), 0755)
				os.WriteFile(cacheFile, body, 0644)
			}
		}

		// 輸出實際響應
		cw.ResponseWriter.WriteHeader(cw.Status())
		cw.ResponseWriter.Write(cw.buf.Bytes())
	}
}
