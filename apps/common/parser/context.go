package parser

import (
	"context"
	"regexp"
	"strings"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/member"
)

type Context struct {
	Sort         *content.ContentSort
	Content      *content.Content
	ContentExt   map[string]interface{}   // 當前內容的擴展字段快取（詳情頁避免重複查詢）
	sortPathCache map[string]string        // 欄目路徑快取（scode → sortPath），避免重複查詢 ContentSort
	ifContext    map[string]interface{}   // {gboot:if} 條件上下文快取，避免每個 if 標籤重複建構 + 8 次 GetConfigValue
	Site         *content.Site
	Company      *content.Company
	Page         map[string]interface{}
	Member       *member.Member
	Gcode        int    // 當前訪客的會員等級編號（0=未登入或無等級）
	Ucode        string // 當前訪客的會員編號
	Keyword      string
	CurrentPage  int
	Filters      map[string]string // ext_ 篩選參數 (ext_type=基礎版 等)
	Ctx          context.Context   // 請求級 context，用於區域數據隔離
	CurrentPath  string            // 當前頁面路徑（已剝離 acode 前綴），用於語言切換保持當前頁
	HttpURL      string            // 動態請求 URL（http(s)://Host），對齊 PbootCMS get_http_url()
}

// safeFieldRe 擴展字段名白名單正則：只允許 ext_ 前綴 + 字母數字下底線
// 防止 SQL 注入：字段名直接拼接進 SQL 查詢，必須嚴格驗證
var safeFieldRe = regexp.MustCompile(`^ext_[a-zA-Z0-9_]+$`)

// 預編譯熱路徑正則（避免每次渲染重複編譯）
var (
	safeTableRe    = regexp.MustCompile(`^[\w]+$`)
	commentSubRe   = regexp.MustCompile(`(?s)\{gboot:commentsub(?:\s+([^}]+))?\}(.*?)\{/gboot:commentsub\}`)
	innerIfRe      = regexp.MustCompile(`(?s)\{gboot:if\(([^)]+)\)\}(.*?)(?:\{else\}(.*?))?\{/gboot:if\}`)
)

// IsSafeFieldName 驗證字段名是否安全（僅允許 ext_ 前綴的合法欄位名）
func IsSafeFieldName(name string) bool {
	return safeFieldRe.MatchString(name)
}

// addLeadingSlash 為本地路徑添加前導 /
func addLeadingSlash(path string) string {
	if path != "" && !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "http") {
		return "/" + path
	}
	return path
}

