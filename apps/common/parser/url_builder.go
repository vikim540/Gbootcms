package parser

import (
	"net/url"
	"strconv"
	"strings"

	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/system"
	"gbootcms/core/acodeplugin"
	"gbootcms/core/db"
)

// preloadSortPaths 批量預載入欄目路徑到 ctx.sortPathCache
// 避免 contentURL 在列表循環中逐個查 DB（N+1 → 1 次查詢）
func preloadSortPaths(ctx *Context, contents []content.Content) {
	if len(contents) == 0 {
		return
	}
	if ctx.sortPathCache == nil {
		ctx.sortPathCache = make(map[string]string)
	}
	// 收集未快取的 scode
	var needed []string
	seen := map[string]bool{}
	for _, c := range contents {
		if c.Scode != "" && !seen[c.Scode] {
			seen[c.Scode] = true
			if _, ok := ctx.sortPathCache[c.Scode]; !ok {
				needed = append(needed, c.Scode)
			}
		}
	}
	if len(needed) == 0 {
		return
	}
	// 單次批量查詢
	var sorts []content.ContentSort
	db.DB.WithContext(ctx.Ctx).
		Where("scode IN ?", needed).
		Find(&sorts)
	for _, s := range sorts {
		if s.Filename != "" {
			ctx.sortPathCache[s.Scode] = s.Filename
		} else if s.URLName != "" {
			ctx.sortPathCache[s.Scode] = s.URLName
		} else {
			ctx.sortPathCache[s.Scode] = ""
		}
	}
}

// contentURL 生成內容鏈接（Google SEO 標準：無 .html 副檔名）
// URL 規則：
//   1. 多段 slug（含 /，如 test/a/b）→ /test/a/b（自定義完整路徑）
//   2. 單段 slug（如 my-article）→ /{sortPath}/{slug}（欄目路徑 + slug）
//   3. 無 slug，欄目有 pathname → /{sortPath}/{id}
//   4. 無 slug，欄目無 pathname → /content/{id}（兜底）
// 使用 ctx.sortPathCache 快取欄目路徑，避免同一頁面重複查詢 ContentSort
func contentURL(ctx *Context, c *content.Content) string {
	if c.Outlink != "" {
		return c.Outlink
	}
	// 查詢欄目 pathname（使用快取避免重複查詢）
	var sortPath string
	if c.Scode != "" {
		if ctx.sortPathCache == nil {
			ctx.sortPathCache = make(map[string]string)
		}
		if cached, ok := ctx.sortPathCache[c.Scode]; ok {
			sortPath = cached
		} else {
			var s content.ContentSort
			if db.DB.WithContext(ctx.Ctx).Where("scode = ?", c.Scode).First(&s).Error == nil {
				if s.Filename != "" {
					sortPath = s.Filename
				} else if s.URLName != "" {
					sortPath = s.URLName
				}
			}
			ctx.sortPathCache[c.Scode] = sortPath
		}
	}
	// 多段 slug（含 /）→ 直接作為完整路徑
	if c.Filename != "" {
		if strings.Contains(c.Filename, "/") {
			return "/" + c.Filename
		}
		// 單段 slug → 欄目路徑 + slug
		if sortPath != "" {
			return "/" + sortPath + "/" + c.Filename
		}
		return "/" + c.Filename
	}
	if c.URLName != "" {
		if strings.Contains(c.URLName, "/") {
			return "/" + c.URLName
		}
		if sortPath != "" {
			return "/" + sortPath + "/" + c.URLName
		}
		return "/" + c.URLName
	}
	// 無 slug，欄目有 pathname → /{sortPath}/{id}
	if sortPath != "" {
		return "/" + sortPath + "/" + strconv.Itoa(int(c.ID))
	}
	// 兜底
	return "/content/" + strconv.Itoa(int(c.ID))
}

// buildFilterURL 生成帶篩選參數的 URL（保留其他欄位的篩選）
func buildFilterURL(ctx *Context, field, value string) string {
	// 基礎路徑：當前欄目頁
	base := "/"
	if ctx.Sort != nil {
		if ctx.Sort.Filename != "" {
			base = "/" + ctx.Sort.Filename
		} else if ctx.Sort.URLName != "" {
			base = "/" + ctx.Sort.URLName
		} else {
			base = "/sort/" + ctx.Sort.Scode
		}
	}
	// 構建查詢參數（驗證字段名安全，防止 URL 注入）
	var params []string
	for k, v := range ctx.Filters {
		if k == field {
			continue // 跳過當前欄位（由 value 決定是否重新加入）
		}
		if !IsSafeFieldName(k) {
			continue
		}
		params = append(params, k+"="+urlEncode(v))
	}
	if value != "" && IsSafeFieldName(field) {
		params = append(params, field+"="+urlEncode(value))
	}
	if len(params) > 0 {
		base += "?" + strings.Join(params, "&")
	}
	return base
}

// urlEncode 簡單 URL 編碼
func urlEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}


// currentHomePath 返回當前語言區域的首頁路徑
// 默認區域返回 /，非默認返回 /{acode}/
func currentHomePath(ctx *Context) string {
	acode := acodeplugin.GetAcode(ctx.Ctx)
	if acode == "" {
		return "/"
	}
	// 查默認區域
	areas := system.GetCachedAreas()
	for _, a := range areas {
		if a.IsDefault == "1" && a.Acode == acode {
			return "/"
		}
	}
	return "/" + acode + "/"
}


