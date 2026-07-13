package api

import (
	"net/http"
	"strings"
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
func getJWTSecret() []byte {
	secret := model.GetConfigValue("api_jwt_secret", "gbootcms-default-jwt-secret-2026")
	return []byte(secret)
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

// APIAuth API 認證中間件
// 支援兩種認證方式：
// 1. JWT Bearer Token（Authorization: Bearer <token>）
// 2. API Key（X-API-Key: <key> 或 query ?api_key=<key>）
func APIAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 公開接口不需要認證
		path := c.Request.URL.Path
		if isPublicAPIPath(path) {
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

		// 嘗試 API Key 認證
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}
		if apiKey != "" {
			configuredKey := model.GetConfigValue("api_key", "")
			if configuredKey != "" && apiKey == configuredKey {
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

// isPublicAPIPath 判斷是否為公開 API 路徑（不需要認證）
func isPublicAPIPath(path string) bool {
	publicPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/site",
		"/api/v1/search",
		"/api/v1/sorts",
		"/api/v1/contents",
		"/api/v1/slides",
		"/api/v1/links",
		"/api/v1/tags",
	}
	for _, p := range publicPaths {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}
