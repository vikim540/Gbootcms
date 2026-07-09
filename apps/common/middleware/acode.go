package middleware

import (
	"context"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/core/acodeplugin"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// defaultAcodeCache 緩存默認區域，避免每次請求都查 DB
var (
	defaultAcodeCache     string
	defaultAcodeCacheOnce sync.Once
)

// GetDefaultAcode 從 DB 查詢默認區域（is_default=1），緩存結果
// 供控制器層判斷是否需要為重定向 URL 添加語言前綴
func GetDefaultAcode() string {
	defaultAcodeCacheOnce.Do(func() {
		skipCtx := acodeplugin.SkipAcode(context.Background())
		var area model.Area
		if err := model.DB.WithContext(skipCtx).Where("is_default = '1'").First(&area).Error; err == nil {
			defaultAcodeCache = area.Acode
		}
		if defaultAcodeCache == "" {
			defaultAcodeCache = "sc" // 回退默認值
		}
	})
	return defaultAcodeCache
}

// InjectAcode 從 URL 前綴、session（後台）或域名匹配（前台）提取當前 acode，
// 注入到 request context，供 GORM AcodePlugin 自動過濾。
//
// 優先級：URL 前綴（/sc /tc /en） > session（後台切換） > 域名匹配 > 默認區域
//
// 此函數可直接在中間件或 NoRoute 處理器中調用。
func InjectAcode(c *gin.Context) {
	acode := ""

	// 1. URL 前綴解析（由 main.go 的 URL 規範化中間件設置到 request context）
	if urlAcode, ok := c.Request.Context().Value(urlAcodeKey{}).(string); ok && urlAcode != "" {
		acode = urlAcode
	}

	// 2. 後台 session
	if acode == "" {
		acode = common.GetSessionString(c, "acode")
	}

	// 3. 前台域名匹配
	if acode == "" {
		acode = matchDomainToAcode(c.Request.Context(), c.Request.Host)
	}

	// 4. 默認區域（從 DB 查詢 is_default=1 的區域）
	if acode == "" {
		acode = GetDefaultAcode()
	}

	// 注入到 request context
	ctx := acodeplugin.WithAcode(c.Request.Context(), acode)
	c.Request = c.Request.WithContext(ctx)
}

// urlAcodeKey 用於在 request context 中傳遞 URL 前綴解析出的 acode
type urlAcodeKey struct{}

// SetURLAcode 將 URL 前綴解析出的 acode 存入 request context
// 供 main.go 的 URL 規範化中間件調用
func SetURLAcode(ctx context.Context, acode string) context.Context {
	return context.WithValue(ctx, urlAcodeKey{}, acode)
}

// AcodeMiddleware 區域隔離中間件（Gin 中間件形式）
//
// 註冊位置：r.Use(middleware.AcodeMiddleware()) 全局註冊
// 同時適用於後台（session）和前台（域名匹配）
func AcodeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		InjectAcode(c)
		c.Next()
	}
}

// matchDomainToAcode 依域名匹配區域（對齊 PHP: core/basic/Kernel.php get_area()）
//
// 邏輯：
//  1. 去除端口號，提取純域名
//  2. 查詢 ay_area 中 domain 完全匹配的區域
//  3. 若無匹配，返回空字串（調用方使用默認區域）
func matchDomainToAcode(ctx context.Context, host string) string {
	// 去除端口號
	domain := host
	if idx := strings.LastIndex(domain, ":"); idx > 0 {
		domain = domain[:idx]
	}
	domain = strings.TrimSpace(domain)
	if domain == "" || domain == "127.0.0.1" || domain == "localhost" {
		return ""
	}

	// 查詢匹配的區域（使用 SkipAcode 跨區查詢，ay_area 本身有 acode 欄位）
	skipCtx := acodeplugin.SkipAcode(ctx)
	var area model.Area
	if err := model.DB.WithContext(skipCtx).Where("domain = ? AND domain != ''", domain).First(&area).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ""
		}
		return ""
	}
	return area.Acode
}
