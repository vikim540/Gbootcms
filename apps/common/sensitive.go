package common

import (
	"strings"
	"sync"

	"gbootcms/apps/admin/model"
)

// 敏感詞快取
var (
	sensitiveWords   []string
	sensitiveWordsMu sync.RWMutex
	sensitiveLoaded  bool
)

// LoadSensitiveWords 從資料庫載入敏感詞列表
// 敏感詞存儲在 ay_config 表的 sensitive_words 欄位，每行一個詞
func LoadSensitiveWords() {
	words := model.GetConfigValue("sensitive_words", "")
	if words == "" {
		sensitiveWordsMu.Lock()
		sensitiveWords = nil
		sensitiveLoaded = true
		sensitiveWordsMu.Unlock()
		return
	}

	parts := strings.Split(words, "\n")
	var result []string
	for _, w := range parts {
		w = strings.TrimSpace(w)
		if w != "" {
			result = append(result, w)
		}
	}

	sensitiveWordsMu.Lock()
	sensitiveWords = result
	sensitiveLoaded = true
	sensitiveWordsMu.Unlock()
}

// RefreshSensitiveWords 刷新敏感詞快取（管理員修改配置後呼叫）
func RefreshSensitiveWords() {
	LoadSensitiveWords()
}

// FilterSensitiveWords 過濾敏感詞，將匹配的詞替換為 ***
func FilterSensitiveWords(text string) string {
	if text == "" {
		return text
	}

	sensitiveWordsMu.RLock()
	words := sensitiveWords
	sensitiveWordsMu.RUnlock()

	if len(words) == 0 {
		// 首次呼叫時自動載入
		if !sensitiveLoaded {
			LoadSensitiveWords()
			sensitiveWordsMu.RLock()
			words = sensitiveWords
			sensitiveWordsMu.RUnlock()
		}
		if len(words) == 0 {
			return text
		}
	}

	result := text
	for _, word := range words {
		if word == "" {
			continue
		}
		// 不區分大小寫替換
		result = replaceIgnoreCase(result, word, "***")
	}
	return result
}

// replaceIgnoreCase 不區分大小寫替換字串
func replaceIgnoreCase(s, old, new string) string {
	if old == "" {
		return s
	}
	lowerS := strings.ToLower(s)
	lowerOld := strings.ToLower(old)
	var result strings.Builder
	idx := 0
	for {
		pos := strings.Index(lowerS[idx:], lowerOld)
		if pos < 0 {
			result.WriteString(s[idx:])
			break
		}
		result.WriteString(s[idx : idx+pos])
		result.WriteString(new)
		idx += pos + len(old)
	}
	return result.String()
}

// HasSensitiveWords 檢查文字是否包含敏感詞
func HasSensitiveWords(text string) bool {
	if text == "" {
		return false
	}

	sensitiveWordsMu.RLock()
	words := sensitiveWords
	sensitiveWordsMu.RUnlock()

	if len(words) == 0 {
		if !sensitiveLoaded {
			LoadSensitiveWords()
			sensitiveWordsMu.RLock()
			words = sensitiveWords
			sensitiveWordsMu.RUnlock()
		}
		if len(words) == 0 {
			return false
		}
	}

	lowerText := strings.ToLower(text)
	for _, word := range words {
		if word != "" && strings.Contains(lowerText, strings.ToLower(word)) {
			return true
		}
	}
	return false
}
