package middleware

import (
	"bytes"
	"compress/gzip"
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
	"golang.org/x/sync/singleflight"
)

// --- 記憶體緩存層（永遠開啟，無需配置） ---

type memCacheEntry struct {
	content     []byte
	gzipContent []byte // 預壓縮 gzip 內容（level 6），快取命中時直接服務，跳過壓縮 CPU 開銷
	expireAt    int64
	staleUntil  int64
}

var (
	memCache sync.Map // key: cacheKey(string), value: memCacheEntry
	sfGroup  singleflight.Group
)

// gzipBytes 將數據壓縮為 gzip（level 6，平衡速度與壓縮率）
func gzipBytes(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, 6)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// serveCacheEntry 服務快取條目：優先返回預壓縮 gzip（避免壓縮 CPU 開銷），否則返回原始內容
// 返回 true 表示已服務（呼叫者應 return），false 表示繼續走壓縮流程
func serveCacheEntry(c *gin.Context, entry memCacheEntry, cacheHeader string) bool {
	acceptEncoding := c.GetHeader("Accept-Encoding")
	if strings.Contains(acceptEncoding, "gzip") && len(entry.gzipContent) > 0 {
		// 預壓縮命中：直接返回 gzip 內容，Compress 中間件看到 flag 後跳過壓縮
		c.Set("pre-compressed", true)
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("X-Cache", cacheHeader+"-GZ")
		c.Data(http.StatusOK, "text/html; charset=utf-8", entry.gzipContent)
		c.Abort()
		return true
	}
	// 客戶端不支援 gzip，返回原始內容（Compress 會用 Brotli/gzip 壓縮）
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("X-Cache", cacheHeader)
	c.Data(http.StatusOK, "text/html; charset=utf-8", entry.content)
	c.Abort()
	return true
}

// ClearHTMLCache 清除所有 HTML 緩存（記憶體 + 檔案）
func ClearHTMLCache() {
	memCache.Range(func(key, value interface{}) bool {
		memCache.Delete(key)
		return true
	})

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

// renderResult 是 singleflight 的返回類型
type renderResult struct {
	status      int
	contentType string
	body        []byte
	gzipBody    []byte // 預壓縮 gzip 內容
	cacheSource string
}

// HTMLCache 動態頁面緩存中間件
// 記憶體緩存永遠開啟（TTL 由 tpl_html_cache_time 控制，預設 900 秒）
// 使用 singleflight 防止快取擊穿 + stale-while-revalidate 避免阻塞
// 快取預壓縮：存入快取時計算 gzip，命中時直接服務預壓縮內容，跳過壓縮 CPU 開銷
// 檔案緩存由 tpl_html_cache 配置項控制（跨重啟持久化）
// 帶 p（pathinfo）或 s（搜索）參數的請求不快取
// 已登入會員不快取（避免個人化內容被快取導致資訊洩露）
func HTMLCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/static") || strings.HasPrefix(path, "/template") {
			c.Next()
			return
		}

		if c.Query("p") != "" || c.Query("s") != "" {
			c.Next()
			return
		}

		if uid := common.GetSessionInt(c, "pboot_uid"); uid > 0 {
			c.Next()
			return
		}

		cacheTTL := 900 * time.Second
		if v := model.GetConfigValue("tpl_html_cache_time", "900"); v != "" {
			if sec, err := time.ParseDuration(v + "s"); err == nil && sec > 0 {
				cacheTTL = sec
			}
		}

		staleGrace := int64(cacheTTL.Seconds()) / 2
		if staleGrace < 60 {
			staleGrace = 60
		}

		cacheKey := fmt.Sprintf("%s%s", c.Request.Host, c.Request.RequestURI)
		cacheKeyHash := fmt.Sprintf("%x", md5.Sum([]byte(cacheKey)))
		now := time.Now().Unix()

		// --- 第一層：記憶體緩存 ---
		if entry, ok := memCache.Load(cacheKeyHash); ok {
			e := entry.(memCacheEntry)
			if now < e.expireAt {
				serveCacheEntry(c, e, "HIT-MEM")
				return
			}
			if now < e.staleUntil {
				serveCacheEntry(c, e, "HIT-STALE")
				return
			}
			memCache.Delete(cacheKeyHash)
		}

		// --- 第二層：檔案緩存 ---
		fileCacheEnabled := model.GetConfigValue("tpl_html_cache", "0") == "1"
		cacheFile := filepath.Join("runtime", "cache", cacheKeyHash+".html")

		if fileCacheEnabled {
			if info, err := os.Stat(cacheFile); err == nil {
				if time.Since(info.ModTime()) < cacheTTL {
					data, err := os.ReadFile(cacheFile)
					if err == nil {
						e := memCacheEntry{
							content:     data,
							gzipContent: gzipBytes(data),
							expireAt:    now + int64(cacheTTL.Seconds()),
							staleUntil:  now + int64(cacheTTL.Seconds()) + staleGrace,
						}
						memCache.Store(cacheKeyHash, e)
						serveCacheEntry(c, e, "HIT-FILE")
						return
					}
				}
			}
		}

		// --- 快取未命中：singleflight 防止快取擊穿 ---
		result, _, shared := sfGroup.Do(cacheKeyHash, func() (interface{}, error) {
			// 雙重檢查
			if entry, ok := memCache.Load(cacheKeyHash); ok {
				e := entry.(memCacheEntry)
				if time.Now().Unix() < e.expireAt {
					return &renderResult{
						status:      http.StatusOK,
						contentType: "text/html; charset=utf-8",
						body:        e.content,
						gzipBody:    e.gzipContent,
						cacheSource: "MEM",
					}, nil
				}
			}

			// 實際渲染
			cw := &cacheBodyWriter{
				ResponseWriter: c.Writer,
				buf:            &bytes.Buffer{},
			}
			c.Writer = cw
			c.Next()
			c.Writer = cw.ResponseWriter

			contentType := cw.Header().Get("Content-Type")
			body := cw.buf.Bytes()
			nowInner := time.Now().Unix()

			var gzipBody []byte
			if strings.HasPrefix(contentType, "text/html") && cw.Status() == http.StatusOK {
				gzipBody = gzipBytes(body)
				memCache.Store(cacheKeyHash, memCacheEntry{
					content:     body,
					gzipContent: gzipBody,
					expireAt:    nowInner + int64(cacheTTL.Seconds()),
					staleUntil:  nowInner + int64(cacheTTL.Seconds()) + staleGrace,
				})

				if fileCacheEnabled {
					os.MkdirAll(filepath.Dir(cacheFile), 0755)
					os.WriteFile(cacheFile, body, 0644)
				}
			}

			return &renderResult{
				status:      cw.Status(),
				contentType: contentType,
				body:        body,
				gzipBody:    gzipBody,
				cacheSource: "FRESH",
			}, nil
		})

		if result == nil {
			return
		}

		rr := result.(*renderResult)

		// 優先服務預壓縮 gzip（避免 1000 並發同時壓縮的 CPU 風暴）
		acceptEncoding := c.GetHeader("Accept-Encoding")
		if strings.Contains(acceptEncoding, "gzip") && len(rr.gzipBody) > 0 {
			c.Set("pre-compressed", true)
			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")
			if rr.contentType != "" {
				c.Header("Content-Type", rr.contentType)
			}
			if shared {
				c.Header("X-Cache", "HIT-SF-GZ")
			} else if rr.cacheSource == "MEM" {
				c.Header("X-Cache", "HIT-MEM-GZ")
			} else {
				c.Header("X-Cache", "MISS-GZ")
			}
			c.Data(rr.status, rr.contentType, rr.gzipBody)
			return
		}

		// 降級：客戶端不支援 gzip，返回原始內容由 Compress 處理
		if rr.contentType != "" {
			c.Header("Content-Type", rr.contentType)
		}
		if shared && rr.cacheSource == "FRESH" {
			c.Header("X-Cache", "HIT-SF")
		} else if rr.cacheSource == "MEM" {
			c.Header("X-Cache", "HIT-MEM")
		} else if shared {
			c.Header("X-Cache", "HIT-SF")
		} else {
			c.Header("X-Cache", "MISS")
		}
		c.Data(rr.status, rr.contentType, rr.body)
	}
}
