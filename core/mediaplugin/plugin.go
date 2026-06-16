// Package mediaplugin 提供 GORM Plugin 自動管理媒體庫緩存失效。
//
// 設計目標：
//   - 一次註冊，所有含文件引用的表的 Controller 無需任何改動
//   - 不引入新的循環引用
//   - 集中維護「會引用文件的表名」白名單
//
// 使用方法（在 core/db 初始化時註冊一次）：
//
//	db.Use(&mediaplugin.MediaDirtyPlugin{})
//
// 工作原理：
//   - 註冊 Create / Update / Delete 鉤子
//   - 寫操作完成後檢查表名是否在白名單
//   - 是 → 標記 dirty，下一次 MediaController 讀取時自動重掃
//   - 否 → 跳過，避免不必要的重掃
//
// 性能優化：
//   - 使用 sync/atomic 而非 sync.RWMutex，dirty 標記讀寫都是 O(1) 無鎖
//   - 預計算 shortName（去 ay_ 前綴）緩存到 sync.Map，避免熱路徑上重複 TrimPrefix
//   - 通過 Statement.Table 優先、Schema.Table 回退的雙重策略，兼容所有 GORM API
//
// 與 PbootCMS PHP 原版對比：
// PHP 原版在每個 Controller 的 mod() 中手動處理，邏輯分散且易遺漏。
// Go 版通過 GORM Plugin 集中處理，無需觸碰任何 Controller。
package mediaplugin

import (
	"strings"
	"sync"
	"sync/atomic"

	"gorm.io/gorm"
)

// dirtyFlag 使用 atomic.Bool 避免鎖競爭。
// 在高併發寫場景下，atomic 開關性能遠優於 RWMutex。
var dirtyFlag atomic.Bool

// schemaTableCache 緩存 GORM Schema 對應的短表名。
// 為什麼需要：GORM 內部會把 "ay_slide" 這樣的表名反覆傳遞，
// 在熱路徑上避免重複 TrimPrefix 可以減少字符串分配。
var schemaTableCache sync.Map // map[interface{}]string (key: *schema.Schema, value: shortName)

// MediaDirtyPlugin 是一個 GORM Plugin
type MediaDirtyPlugin struct{}

// Name 插件名稱（GORM Plugin 接口要求）
func (p *MediaDirtyPlugin) Name() string {
	return "MediaDirtyPlugin"
}

// Initialize 在 GORM 初始化時註冊回調
func (p *MediaDirtyPlugin) Initialize(db *gorm.DB) error {
	// 注冊 Create 回調
	if err := db.Callback().Create().After("gorm:after_create").Register("media_dirty:after_create", p.afterWrite); err != nil {
		return err
	}
	// 注冊 Update 回調（含 Updates、UpdateColumn 等）
	if err := db.Callback().Update().After("gorm:after_update").Register("media_dirty:after_update", p.afterWrite); err != nil {
		return err
	}
	// 注冊 Delete 回調
	if err := db.Callback().Delete().After("gorm:after_delete").Register("media_dirty:after_delete", p.afterWrite); err != nil {
		return err
	}
	return nil
}

// afterWrite 在所有寫操作後自動標記媒體緩存為臟
//
// 執行路徑（按開銷從小到大）：
//  1. Schema 緩存命中（最常見的熱路徑）：sync.Map.Load → MarkDirty
//  2. Statement.Table 已有短表名：TrimPrefix → MarkDirty
//  3. Statement.Table 是 "ay_xxx"：TrimPrefix → MarkDirty
//  4. 只能從 Schema.Table 取：緩存到 sync.Map
//  5. 都取不到：return（極少見，如 Exec 原始 SQL）
func (p *MediaDirtyPlugin) afterWrite(db *gorm.DB) {
	if db.Statement == nil {
		return
	}

	// 最常見熱路徑：Statement.Table 直接就有表名
	// 這種情況下做一次 TrimPrefix 後查白名單
	if shortName := normalizeTableName(db.Statement.Table); shortName != "" {
		if MediaReferencingTables[shortName] {
			MarkDirty()
		}
		return
	}

	// 回退路徑：Schema 為 nil 跳過
	if db.Statement.Schema == nil {
		return
	}

	// Schema 緩存查找（避免重複 TrimPrefix）
	if v, ok := schemaTableCache.Load(db.Statement.Schema); ok {
		if s, ok := v.(string); ok && MediaReferencingTables[s] {
			MarkDirty()
		}
		return
	}

	// 首次訪問此 Schema，計算並緩存
	fullName := db.Statement.Schema.Table
	short := strings.TrimPrefix(fullName, "ay_")
	schemaTableCache.Store(db.Statement.Schema, short)
	if MediaReferencingTables[short] {
		MarkDirty()
	}
}

// normalizeTableName 統一處理表名，提取短名（去 ay_ 前綴）。
// 對於空字符串返回空字符串，便於上層判斷。
//
// 注意：這裡做的是「白名單測試友好」的處理——只關心末尾標識符是什麼。
// 即使傳入 "ay_slide" 或 "slide"，都能正確匹配白名單。
func normalizeTableName(fullName string) string {
	if fullName == "" {
		return ""
	}
	// 去掉可能的 schema 前綴（如 "public.ay_slide"）
	if idx := strings.LastIndex(fullName, "."); idx >= 0 {
		fullName = fullName[idx+1:]
	}
	return strings.TrimPrefix(fullName, "ay_")
}
