package parser

import (
	"context"
	"fmt"
	"gbootcms/apps/admin/model"
	"gbootcms/core/acodeplugin"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 預編譯正則表達式
// 對齊 PbootCMS: /(＜a .*?>.*?<\/a>)|(alt=.*?>)|(title=.*?>)/i
// 改進：保護整個 <a>...</a> 區塊（含內容）+ 所有 HTML 標籤
// 比原版更全面：原版只保護 <a> 和 alt/title 屬性，此處保護所有標籤結構
var protectHTMLRe = regexp.MustCompile(`(?is)(<a\b[^>]*>.*?</a>)|(<[^>]+>)`)

// tagsCache 記憶體快取：避免每次渲染都查詢 ay_tags 表
var (
	tagsCache     []model.Tags
	tagsCacheTime time.Time
	tagsCacheMu   sync.RWMutex
)

// GetCachedTags 取得快取的標籤列表（60 秒 TTL，由 GORM 回調連帶清除）
func GetCachedTags() []model.Tags {
	tagsCacheMu.RLock()
	if time.Since(tagsCacheTime) < 60*time.Second && tagsCache != nil {
		t := tagsCache
		tagsCacheMu.RUnlock()
		return t
	}
	tagsCacheMu.RUnlock()

	tagsCacheMu.Lock()
	defer tagsCacheMu.Unlock()
	if time.Since(tagsCacheTime) < 60*time.Second && tagsCache != nil {
		return tagsCache
	}
	var tags []model.Tags
	skipCtx := acodeplugin.SkipAcode(context.Background())
	// 對齊 PbootCMS: ORDER BY length(name) DESC — 長關鍵字優先處理
	model.DB.WithContext(skipCtx).Order("length(name) DESC, sorting ASC, id ASC").Find(&tags)
	tagsCache = tags
	tagsCacheTime = time.Now()
	return tagsCache
}

// ClearTagsCache 清除標籤快取（由 GORM 回調觸發）
func ClearTagsCache() {
	tagsCacheMu.Lock()
	tagsCache = nil
	tagsCacheMu.Unlock()
}

// replaceContentTags 對文章內容執行內鏈替換（對齊 PbootCMS parserCurrentContentLabel case 'content' 邏輯）
//
// 改進點（相對於 PbootCMS PHP 原版）：
//  1. 去重：同名標籤只保留一個（PbootCMS 原版的 strpos 過濾會移除全部同名標籤，導致重複標籤完全不被替換）
//  2. 預佔位替換：先用佔位符替換標籤名，全部替換完成後再還原為 <a> 標籤
//     避免後續標籤匹配到前面已替換的 <a> 標籤內容（PbootCMS 原版的潛在 bug）
//  3. 使用 strings.Replace 而非 preg_replace，避免標籤名中的正則特殊字元被誤解析
func replaceContentTags(content string, _ context.Context) string {
	tags := GetCachedTags()
	if len(tags) == 0 {
		return content
	}

	// 1. 保護所有 HTML 區塊（對齊 PbootCMS: 保護 <a>...</a> 整體 + 所有 HTML 標籤）
	// 只替換標籤之間的純文字，避免破壞 HTML 結構或在屬性值內誤替換
	htmlPlaceholders := make(map[string]string)
	idx := 0
	content = protectHTMLRe.ReplaceAllStringFunc(content, func(match string) string {
		key := fmt.Sprintf("#rega:%d#", idx)
		htmlPlaceholders[key] = match
		idx++
		return key
	})

	// 2. 去重：同名標籤只保留第一個（改進 PbootCMS 原版邏輯）
	// PbootCMS 原版的 strpos 過濾會移除全部同名標籤，導致重複標籤完全不被替換
	seen := make(map[string]bool)
	deduped := make([]model.Tags, 0, len(tags))
	for _, tag := range tags {
		if tag.Name == "" || seen[tag.Name] {
			continue
		}
		seen[tag.Name] = true
		deduped = append(deduped, tag)
	}

	// 3. 去除包含關係的短 tags，實現長關鍵字優先（對齊 PbootCMS 原版過濾邏輯）
	// 只有當 tag2.Name 嚴格長於 tag.Name 時才認為 tag.Name 是「被包含的短標籤」
	// 去重後同名標籤已唯一，不需要處理等長情況
	filtered := make([]model.Tags, 0, len(deduped))
	for i, tag := range deduped {
		isShort := false
		for j, tag2 := range deduped {
			if i != j && len(tag2.Name) > len(tag.Name) && strings.Contains(tag2.Name, tag.Name) {
				isShort = true
				break
			}
		}
		if !isShort {
			filtered = append(filtered, tag)
		}
	}

	// 4. 執行內鏈替換 — 預佔位策略（改進 PbootCMS 原版）
	// 先用佔位符替換標籤名，避免替換文字中的標籤名被後續標籤匹配
	// 例：標籤 "PbootCMS" 替換為 #tagrep:0#，而非直接替換為 <a href="...">PbootCMS</a>
	// 這樣後續標籤無法在 #tagrep:0# 中匹配到任何內容
	replaceNum := model.GetConfigValue("content_tags_replace_num", "3")
	num := 0
	fmt.Sscanf(replaceNum, "%d", &num)
	if num <= 0 {
		num = 3
	}

	tagReplacements := make(map[string]string)
	repIdx := 0
	for _, tag := range filtered {
		if tag.Name == "" || tag.Link == "" {
			continue
		}
		placeholder := fmt.Sprintf("#tagrep:%d#", repIdx)
		repIdx++
		tagReplacements[placeholder] = fmt.Sprintf(`<a href="%s">%s</a>`, tag.Link, tag.Name)
		// strings.Replace 行為等同 PHP preg_replace 的 $limit 參數：
		// 每次替換後搜尋位置前進到替換文字之後，不會重複匹配替換文字中的 tag.Name
		content = strings.Replace(content, tag.Name, placeholder, num)
	}

	// 5. 還原標籤替換佔位符 → 實際 <a> 標籤
	for key, val := range tagReplacements {
		content = strings.ReplaceAll(content, key, val)
	}

	// 6. 還原保護的 HTML 內容
	for key, val := range htmlPlaceholders {
		content = strings.ReplaceAll(content, key, val)
	}

	return content
}

// replaceKeyword 對文章內容執行敏感詞過濾（對齊 PbootCMS parserReplaceKeyword 邏輯）
// 將配置的敏感詞替換為等長星號
func replaceKeyword(content string) string {
	keywordReplace := model.GetConfigValue("content_keyword_replace", "")
	if keywordReplace == "" {
		return content
	}
	keywords := strings.Split(keywordReplace, ",")
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		// 按字元長度生成對應數量的星號（對齊 PHP: str_repeat('*', mb_strlen($value))）
		stars := strings.Repeat("*", len([]rune(kw)))
		content = strings.ReplaceAll(content, kw, stars)
	}
	return content
}
