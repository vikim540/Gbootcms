// 一次性腳本：為現有 ay_content_sort 中 filename 為空的記錄
// 自動生成基於 name 的拼音 URL 名稱（簡化版，僅支援英文+數字+底線）。
//
// 對於中文 name，預設 filename = scode 作為後備。
//
// 用法：在項目根目錄執行
//   go run ./cmd/backfill_filename
package main

import (
	"fmt"
	"log"
	"regexp"

	"pbootcms-go/apps/admin/model"
	contentmodel "pbootcms-go/apps/admin/model/content"
	"pbootcms-go/config"
	"pbootcms-go/core/db"
)

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9\-_\/]+`)
var multiDash = regexp.MustCompile(`-+`)

func toAsciiURL(s string) string {
	// 非 ASCII 字符全部替換為空（簡化版；未引入 pinyin 庫以保持零依賴）
	if allASCII(s) {
		out := nonAlnum.ReplaceAllString(s, "-")
		out = multiDash.ReplaceAllString(out, "-")
		out = trimDash(out)
		return out
	}
	return ""
}

func allASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func trimDash(s string) string {
	for len(s) > 0 && (s[0] == '-' || s[0] == '_' || s[0] == '/') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == '-' || s[len(s)-1] == '_' || s[len(s)-1] == '/') {
		s = s[:len(s)-1]
	}
	return s
}

func main() {
	cfg := config.Load("config/config.json")
	if err := model.InitDB(cfg); err != nil {
		log.Fatalf("init db: %v", err)
	}

	var sorts []model.ContentSort
	if err := db.DB.Order("id ASC").Find(&sorts).Error; err != nil {
		log.Fatalf("query sorts: %v", err)
	}

	updated := 0
	skipped := 0
	for _, s := range sorts {
		if s.Filename != "" {
			skipped++
			continue
		}

		// 嘗試從 name 生成 URL 名稱
		candidate := toAsciiURL(s.Name)
		if candidate == "" {
			// 中文 name：使用 scode 作為後備 filename
			candidate = s.Scode
		}
		if candidate == "" {
			candidate = fmt.Sprintf("sort-%d", s.ID)
		}

		// 確保唯一
		final := contentmodel.GenerateUniqueFilename(candidate, fmt.Sprintf("id<>%d", s.ID))

		if err := db.DB.Model(&model.ContentSort{}).
			Where("id = ?", s.ID).
			Update("filename", final).Error; err != nil {
			log.Printf("[WARN] id=%d update filename failed: %v", s.ID, err)
			continue
		}
		fmt.Printf("[OK] id=%d scode=%s name=%q -> filename=%q\n", s.ID, s.Scode, s.Name, final)
		updated++
	}

	fmt.Printf("\nDone. updated=%d skipped=%d total=%d\n", updated, skipped, len(sorts))
}
