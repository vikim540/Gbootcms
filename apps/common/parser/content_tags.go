package parser

import (
	"context"
	"fmt"
	"gbootcms/apps/admin/model"
	"gbootcms/core/acodeplugin"
	"regexp"
	"strings"
)

// replaceContentTags 對文章內容執行內鏈替換（對齊 PbootCMS parserCurrentContentLabel case 'content' 邏輯）
// 1. 查詢當前語言的所有 tags
// 2. 保護已有的 <a> 連結和 alt/title 屬性
// 3. 去除包含關係的短 tags，實現長關鍵字優先
// 4. 執行替換（限制次數）
// 5. 還原保護的內容
func replaceContentTags(content string, ctx context.Context) string {
	var tags []model.Tags
	// 使用 SkipAcode 查詢所有語言的標籤，因為內鏈替換應跨語言生效
	// （例如英文文章中的 "Go" 關鍵字也應被替換為連結）
	skipCtx := acodeplugin.SkipAcode(ctx)
	model.DB.WithContext(skipCtx).Order("sorting ASC, id ASC").Find(&tags)
	if len(tags) == 0 {
		return content
	}

	// 1. 保護所有 HTML 標籤（包括屬性值中的內容），只替換標籤之間的文字
	protectRegex := regexp.MustCompile(`(?i)<[^>]+>`)
	placeholders := make(map[string]string)
	idx := 0
	content = protectRegex.ReplaceAllStringFunc(content, func(match string) string {
		key := fmt.Sprintf("#rega:%d#", idx)
		placeholders[key] = match
		idx++
		return key
	})

	// 2. 去除包含關係的短 tags，實現長關鍵字優先
	// 修正：只有當 tag2.Name 嚴格長於 tag.Name 時才認為 tag.Name 是「被包含的短標籤」
	// 避免相同名稱的 tag 互相過濾導致全部被移除
	filtered := make([]model.Tags, 0, len(tags))
	for i, tag := range tags {
		isShort := false
		for j, tag2 := range tags {
			if i != j && len(tag2.Name) > len(tag.Name) && strings.Contains(tag2.Name, tag.Name) {
				isShort = true
				break
			}
		}
		if !isShort {
			filtered = append(filtered, tag)
		}
	}

	// 3. 執行內鏈替換（對齊 PHP: preg_replace 限制次數）
	replaceNum := model.GetConfigValue("content_tags_replace_num", "3")
	num := 0
	fmt.Sscanf(replaceNum, "%d", &num)
	if num <= 0 {
		num = 3
	}

	for _, tag := range filtered {
		if tag.Name == "" || tag.Link == "" {
			continue
		}
		replacement := fmt.Sprintf(`<a href="%s">%s</a>`, tag.Link, tag.Name)
		// 每個 tag 限制替換 num 次
		count := 0
		tagRegex := regexp.MustCompile(regexp.QuoteMeta(tag.Name))
		content = tagRegex.ReplaceAllStringFunc(content, func(s string) string {
			if count >= num {
				return s
			}
			count++
			return replacement
		})
	}

	// 4. 還原保護的內容
	for key, val := range placeholders {
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
