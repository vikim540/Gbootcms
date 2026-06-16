// Package mediaplugin 提供 GORM Plugin 自動管理媒體庫緩存失效。
//
// 設計目標：
//   - 一次註冊，所有含文件引用的表的 Controller 無需任何改動
//   - 不引入新的循環引用
//   - 集中維護「會引用文件的表名」白名單
package mediaplugin

import (
	"sync"
)

// 媒體庫緩存失效標記的全局狀態。
// 為避免循環引用，這裡自管理 dirty 標記，
// MediaController 通過 IsDirty/ClearDirty 讀取與清除。

var (
	dirtyMu sync.RWMutex
	dirty   = false
)

// MarkDirty 標記媒體緩存為臟。
// 由 GORM Plugin 在寫操作回調中調用。
func MarkDirty() {
	dirtyMu.Lock()
	dirty = true
	dirtyMu.Unlock()
}

// IsDirty 檢查緩存是否被標記為臟。
// 由 MediaController 在讀取前調用。
func IsDirty() bool {
	dirtyMu.RLock()
	defer dirtyMu.RUnlock()
	return dirty
}

// ClearDirty 清除臟標記。
// 由 MediaController 在掃描成功後調用。
func ClearDirty() {
	dirtyMu.Lock()
	dirty = false
	dirtyMu.Unlock()
}

// MediaReferencingTables 引用媒體文件的表白名單。
// 如果未來新增了帶文件引用的表，只需在此加入表名。
var MediaReferencingTables = map[string]bool{
	"slide":        true,
	"content":      true,
	"content_sort": true,
	"link":         true,
	"company":      true,
	"site":         true,
}
