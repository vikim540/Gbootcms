package common

import (
	"html"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// gbootIfRegex 匹配 {gboot:if(...)} 標籤（對齊 PbootCMS PHP 的 pboot:if 過濾）
var gbootIfRegex = regexp.MustCompile(`(?i)\{gboot:if`)

// gbootSqlRegex 匹配 {gboot:sql(...)} 標籤（對齊 PbootCMS PHP 的 pboot:sql 過濾）
var gbootSqlRegex = regexp.MustCompile(`(?i)\{gboot:sql`)

// hexBracketRegex 匹配十六進制括號 x3c/x3e（對齊 PbootCMS PHP 的 hex 括號過濾）
var hexBracketRegex = regexp.MustCompile(`(?i)(x3c)|(x3e)`)

// unsafeRedirectRegex 預編譯的開放重定向檢測正則（全域複用，避免每次調用重新編譯）
var unsafeRedirectRegex = regexp.MustCompile(`^(https?:)?//`)

// identifierRegex 識別符白名單正則（對齊 PbootCMS PHP checkKey: /^[\w\.\-]+$/）
var identifierRegex = regexp.MustCompile(`^[\w\.\-]+$`)

// varTypeRegex 對齊 PbootCMS PHP var 類型驗證: /^[\w\-\.]+$/
var varTypeRegex = regexp.MustCompile(`^[\w\-\.]+$`)

// columnNameRegex 欄位名驗證（必須字母開頭，對齊 PbootCMS MemberField: /^[a-zA-Z][\w]+$/）
var columnNameRegex = regexp.MustCompile(`^[a-zA-Z][\w]+$`)

// EscapeString 對齊 PbootCMS PHP escape_string()：
// 使用 htmlspecialchars(ENT_QUOTES, UTF-8) 等價的 Go html.EscapeString
// 用於 XSS 防護，轉義 < > & " '
func EscapeString(s string) string {
	if s == "" {
		return s
	}
	return html.EscapeString(s)
}

// FilterUserInput 對齊 PbootCMS PHP filter() 的安全過濾邏輯：
// 1. 過濾 {gboot:if} 標籤（防止模板注入）
// 2. 過濾 {gboot:sql} 標籤
// 3. 清除十六進制括號 x3c/x3e
// 4. HTML 轉義（XSS 防護）
func FilterUserInput(s string) string {
	if s == "" {
		return s
	}
	s = strings.TrimSpace(s)
	s = hexBracketRegex.ReplaceAllString(s, "")
	s = gbootIfRegex.ReplaceAllString(s, "")
	s = gbootSqlRegex.ReplaceAllString(s, "")
	return EscapeString(s)
}

// CheckIdentifier 驗證 SQL 識別符（表名、欄位名）是否安全
// 對齊 PbootCMS PHP Model::checkKey(): 只允許字母、數字、下劃線、點、橫線
// 返回 true 表示安全，false 表示含有非法字符
func CheckIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}
	return identifierRegex.MatchString(identifier)
}

// CheckVarType 對齊 PbootCMS PHP var 類型驗證: /^[\w\-\.]+$/
// 用於驗證用戶輸入的表名、欄位名等
func CheckVarType(s string) bool {
	if s == "" {
		return false
	}
	return varTypeRegex.MatchString(s)
}

// CheckColumnName 驗證欄位名必須以字母開頭
// 對齊 PbootCMS PHP MemberField: /^[a-zA-Z][\w]+$/
func CheckColumnName(s string) bool {
	if s == "" {
		return false
	}
	return columnNameRegex.MatchString(s)
}

// === Cookie 安全工具 ===
// 對標 Swoole 6 的安全回應標頭 + PHP 8.5 的 SameSite cookie 屬性

// IsHTTPS 判斷請求是否通過 HTTPS 連線
// 支援三種偵測方式：直接 TLS、反向代理 X-Forwarded-Proto、X-Forwarded-Ssl
func IsHTTPS(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		return true
	}
	if c.GetHeader("X-Forwarded-Ssl") == "on" {
		return true
	}
	return false
}

// SetSecureCookie 設定帶完整安全屬性的 Cookie
// Secure: 根據請求是否 HTTPS 動態判斷（HTTP 開發環境仍可正常使用）
// HttpOnly: true（防止 JavaScript 竊取，防 XSS）
// SameSite: Lax（允許頂層導航攜帶，防 CSRF）
func SetSecureCookie(c *gin.Context, name, value string, maxAge int, path string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   IsHTTPS(c),
		SameSite: http.SameSiteLaxMode,
	})
}

// IsSafeRedirectURL 驗證跳轉 URL 為相對路徑（防止開放重定向攻擊）
// 規則：
// 1. 必須以 / 開頭
// 2. 不能以 // 開頭（協議相對 URL，瀏覽器會解釋為外部域名）
// 3. 不能是 http:// 或 https:// 絕對 URL
func IsSafeRedirectURL(u string) bool {
	if u == "" {
		return false
	}
	if len(u) >= 2 && u[0] == '/' && u[1] == '/' {
		return false
	}
	if unsafeRedirectRegex.MatchString(u) {
		return false
	}
	return u[0] == '/'
}
