package common

import (
	"os"
	"strings"

	"gbootcms/apps/admin/model"
	"github.com/gin-gonic/gin"
)

// IsStagingHost 判斷當前請求的主機是否為測試站、預覽站或本地開發環境
// 規則：
//  1. 若 Host 包含 .eu.org、.test、.local、localhost、127.0.0.1、或包含端口號（:8080 等），直接判定為非生產環境
//  2. 檢查環境變量 ENV / APP_ENV
//  3. 若後台配置了生產域名（如 hkcmereye.com），而當前請求 Host 與之不一致，直接判定為測試站
func IsStagingHost(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	host := strings.ToLower(c.Request.Host)
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// 本地開發與常見測試/預覽後綴
	if host == "localhost" || host == "127.0.0.1" || strings.HasSuffix(host, ".eu.org") ||
		strings.HasSuffix(host, ".test") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".nip.io") {
		return true
	}

	// 檢查環境變量
	env := strings.ToLower(os.Getenv("ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("APP_ENV"))
	}
	if env == "staging" || env == "dev" || env == "test" || env == "development" {
		return true
	}

	// 檢查後台配置的生產域名
	prodDomain := GetProductionDomain(c)
	if prodDomain != "" && host != prodDomain {
		return true
	}

	return false
}

// GetProductionDomain 取得配置的生產域名（純主機名，不含協議和路徑）
func GetProductionDomain(c *gin.Context) string {
	domain := model.GetConfigValue("httpurl", "")
	if domain == "" {
		var site model.Site
		ctx := c.Request.Context()
		if model.DB.WithContext(ctx).First(&site).Error == nil && site.Domain != "" {
			domain = site.Domain
		}
	}
	if domain == "" {
		return ""
	}
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimRight(domain, "/")
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	return strings.ToLower(domain)
}

// GetRequestBaseURL 獲取當前請求應用的基礎 URL
// 關鍵：如果當前為測試站（如 gocms.icun.eu.org），嚴格使用當前測試主機，絕不跨域名綁定生產域名！
func GetRequestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	if IsStagingHost(c) {
		return scheme + "://" + c.Request.Host
	}

	prod := GetProductionDomain(c)
	if prod != "" {
		return "https://" + prod
	}
	return scheme + "://" + c.Request.Host
}
