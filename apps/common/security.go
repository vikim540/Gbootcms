package common

import (
	"html"
	"regexp"
	"strings"
)

// gbootIfRegex 匹配 {gboot:if(...)} 標籤（對齊 PbootCMS PHP 的 pboot:if 過濾）
var gbootIfRegex = regexp.MustCompile(`(?i)\{gboot:if`)

// gbootSqlRegex 匹配 {gboot:sql(...)} 標籤（對齊 PbootCMS PHP 的 pboot:sql 過濾）
var gbootSqlRegex = regexp.MustCompile(`(?i)\{gboot:sql`)

// hexBracketRegex 匹配十六進制括號 x3c/x3e（對齊 PbootCMS PHP 的 hex 括號過濾）
var hexBracketRegex = regexp.MustCompile(`(?i)(x3c)|(x3e)`)

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
