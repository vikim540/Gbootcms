package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gbootcms/apps/admin/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims JWT 聲明
type JWTClaims struct {
	UID      int    `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// getJWTSecret 從配置取得 JWT 密鑰
// 配置項 api_jwt_secret 為空時返回 nil，呼叫方應在啟動時拒絕 API 服務
func getJWTSecret() []byte {
	secret := model.GetConfigValue("api_jwt_secret", "")
	return []byte(secret)
}

// IsJWTConfigured 檢查 JWT 密鑰是否已配置
func IsJWTConfigured() bool {
	return model.GetConfigValue("api_jwt_secret", "") != ""
}

// GenerateToken 生成 JWT Token
func GenerateToken(uid int, username string) (string, error) {
	claims := JWTClaims{
		UID:      uid,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gbootcms",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ParseToken 解析 JWT Token
func ParseToken(tokenStr string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// --- API 登入鎖定（記憶體 TTL，不使用檔案） ---

type loginAttemptEntry struct {
	count int
	time  int64
}

var (
	loginAttempts   sync.Map // key: IP, value: loginAttemptEntry
	loginAttemptsMu sync.Mutex
)

// checkLoginLock 檢查 IP 是否已被鎖定，返回剩餘鎖定秒數（0=未鎖定）
func checkLoginLock(c *gin.Context) int {
	lockCount := 5
	lockTime := 900
	if v := model.GetConfigValue("lock_count", "5"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%d", &lockCount); err != nil || parsed == 0 {
			lockCount = 5
		}
	}
	if v := model.GetConfigValue("lock_time", "900"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%d", &lockTime); err != nil || parsed == 0 {
			lockTime = 900
		}
	}

	ip := c.ClientIP()
	now := time.Now().Unix()

	val, ok := loginAttempts.Load(ip)
	if !ok {
		return 0
	}
	entry, ok := val.(loginAttemptEntry)
	if !ok {
		return 0
	}

	if entry.count >= lockCount {
		elapsed := now - entry.time
		remain := lockTime - int(elapsed)
		if remain > 0 {
			return remain
		}
		// 鎖定已過期，清除記錄
		loginAttempts.Delete(ip)
	}
	return 0
}

// recordLoginFailure 記錄登入失敗
func recordLoginFailure(c *gin.Context) {
	lockCount := 5
	lockTime := 900
	if v := model.GetConfigValue("lock_count", "5"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%d", &lockCount); err != nil || parsed == 0 {
			lockCount = 5
		}
	}
	if v := model.GetConfigValue("lock_time", "900"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%d", &lockTime); err != nil || parsed == 0 {
			lockTime = 900
		}
	}

	ip := c.ClientIP()
	now := time.Now().Unix()

	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	val, ok := loginAttempts.Load(ip)
	if ok {
		entry := val.(loginAttemptEntry)
		if entry.count < lockCount && now-entry.time < int64(lockTime) {
			entry.count++
			entry.time = now
		} else {
			// 過期或已達上限，重新計數
			entry = loginAttemptEntry{count: 1, time: now}
		}
		loginAttempts.Store(ip, entry)
	} else {
		loginAttempts.Store(ip, loginAttemptEntry{count: 1, time: now})
	}
}

// clearLoginFailure 登入成功時清除失敗記錄
func clearLoginFailure(c *gin.Context) {
	loginAttempts.Delete(c.ClientIP())
}

// APIAuth API 認證中間件
// 支援兩種認證方式：
// 1. JWT Bearer Token（Authorization: Bearer <token>）
// 2. API Key（X-API-Key: <key> 或 query ?api_key=<key>）
func APIAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 公開接口不需要認證
		path := c.Request.URL.Path
		if isPublicAPIPath(path, c.Request.Method) {
			c.Next()
			return
		}

		// 嘗試 JWT 認證
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := ParseToken(tokenStr)
			if err == nil {
				c.Set("api_uid", claims.UID)
				c.Set("api_username", claims.Username)
				c.Next()
				return
			}
		}

		// 嘗試 API Key 認證（常量時間比較，防時序攻擊）
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}
		if apiKey != "" {
			configuredKey := model.GetConfigValue("api_key", "")
			if configuredKey != "" && subtle.ConstantTimeCompare([]byte(apiKey), []byte(configuredKey)) == 1 {
				c.Set("api_uid", 0)
				c.Set("api_username", "api_key")
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 0,
			"msg":  "認證失敗，請提供有效的 JWT Token 或 API Key",
		})
		c.Abort()
	}
}

// CORS 跨域中間件
// 允許的域名從配置項 api_cors_origins 讀取（逗號分隔），為空時默認允許全部
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowedOrigins := model.GetConfigValue("api_cors_origins", "*")
		allowed := false
		if allowedOrigins == "*" {
			allowed = true
		} else {
			for _, o := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(o) == origin {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// isPublicAPIPath 判斷是否為公開 API 路徑（不需要認證）
// method 參數用於區分同一路徑的不同操作（如 POST /messages 公開，GET /messages 需認證）
func isPublicAPIPath(path, method string) bool {
	// 認證接口永遠公開
	if path == "/api/v1/auth/login" || path == "/api/v1/auth/refresh" {
		return true
	}

	// 留言提交公開，留言列表需認證
	if path == "/api/v1/messages" && method == "POST" {
		return true
	}

	// 以下路徑及其子路徑公開（GET 資源查詢）
	publicGETPaths := []string{
		"/api/v1/site",
		"/api/v1/company",
		"/api/v1/search",
		"/api/v1/sorts",
		"/api/v1/contents",
		"/api/v1/nav",
		"/api/v1/slides",
		"/api/v1/links",
		"/api/v1/tags",
	}
	if method == "GET" {
		for _, p := range publicGETPaths {
			if path == p || strings.HasPrefix(path, p+"/") {
				return true
			}
		}
	}
	return false
}
