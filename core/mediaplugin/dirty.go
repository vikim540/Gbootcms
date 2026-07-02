// Package mediaplugin 提供 GORM Plugin 自動管理媒體庫緩存失效。
//
// 設計目標：
//   - 一次註冊，所有含文件引用的表的 Controller 無需任何改動
//   - 不引入新的循環引用
//   - 集中維護「會引用文件的表名」白名單
package mediaplugin

// dirtyFlag 在 plugin.go 中聲明（atomic.Bool），這裡是對外的讀寫 API。
// 使用 atomic 操作避免鎖競爭，O(1) 無鎖。

// MarkDirty 標記媒體緩存為臟。
// 由 GORM Plugin 在寫操作回調中調用。
//
// 性能：atomic.Store 操作，無鎖。
func MarkDirty() {
	dirtyFlag.Store(true)
}

// IsDirty 檢查緩存是否被標記為臟。
// 由 MediaController 在讀取前調用。
//
// 性能：atomic.Load 操作，無鎖。
// 注意：返回值只是當前快照，因為多線程可能隨時 MarkDirty。
// 但 MediaController 內部會再用讀寫鎖保護，這裡只是快速路徑。
func IsDirty() bool {
	return dirtyFlag.Load()
}

// ClearDirty 清除臟標記。
// 由 MediaController 在掃描成功後調用。
//
// 性能：atomic.Store 操作，無鎖。
func ClearDirty() {
	dirtyFlag.Store(false)
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
	"member":       true,
	"label":        true,
}
