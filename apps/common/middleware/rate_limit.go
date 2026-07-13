package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ipRateEntry 單個 IP 的速率計數條目
type ipRateEntry struct {
	count    int
	lastTime time.Time
}

// ipRateLimiter 通用的 IP 速率限制器（內存實現，適合單機部署）
// 使用滑動窗口算法：在指定時間窗口內，每個 IP 最多允許 maxRequests 次請求
type ipRateLimiter struct {
	mu          sync.Mutex
	entries     map[string]*ipRateEntry
	window      time.Duration // 時間窗口
	maxRequests int           // 窗口內最大請求數
}

var (
	// apiRateLimiter API 端點專用限速器：每 IP 每分鐘最多 30 次
	apiLimiter     = newIPRateLimiter(time.Minute, 30)
	// voteRateLimiter 投票端點專用限速器：每 IP 每分鐘最多 10 次（更嚴格）
	voteLimiter     = newIPRateLimiter(time.Minute, 10)
	// 清理計時器（避免內存無限增長）
	cleanupOnce    sync.Once
)

// newIPRateLimiter 建立新的 IP 速率限制器
func newIPRateLimiter(window time.Duration, maxRequests int) *ipRateLimiter {
	rl := &ipRateLimiter{
		entries:     make(map[string]*ipRateEntry),
		window:      window,
		maxRequests: maxRequests,
	}
	return rl
}

// Allow 檢查 IP 是否允許通過，允許則計數 +1
func (rl *ipRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[ip]
	if !exists || now.Sub(entry.lastTime) > rl.window {
		// 新 IP 或窗口已過期，重置計數
		rl.entries[ip] = &ipRateEntry{count: 1, lastTime: now}
		return true
	}
	if entry.count >= rl.maxRequests {
		return false
	}
	entry.count++
	entry.lastTime = now
	return true
}

// cleanup 清理過期條目，防止內存洩漏
func (rl *ipRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, entry := range rl.entries {
		if now.Sub(entry.lastTime) > rl.window*2 {
			delete(rl.entries, ip)
		}
	}
}

// startCleanup 啟動後台定期清理（只啟動一次）
func startCleanup() {
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				apiLimiter.cleanup()
				voteLimiter.cleanup()
			}
		}()
	})
}

// APIRateLimit API 端點通用速率限制中間件
// 每 IP 每分鐘最多 30 次請求
func APIRateLimit() gin.HandlerFunc {
	startCleanup()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !apiLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": 0,
				"msg":  "請求過於頻繁，請稍後再試",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// VoteRateLimit 投票端點嚴格速率限制中間件
// 每 IP 每分鐘最多 10 次請求（防止刷票）
func VoteRateLimit() gin.HandlerFunc {
	startCleanup()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !voteLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": 0,
				"msg":  "操作過於頻繁，請稍後再試",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
