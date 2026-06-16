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
// 與 PbootCMS PHP 原版對比：
// PHP 原版在每個 Controller 的 mod() 中手動處理，邏輯分散且易遺漏。
// Go 版通過 GORM Plugin 集中處理，無需觸碰任何 Controller。
package mediaplugin

import (
	"strings"

	"gorm.io/gorm"
)

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
func (p *MediaDirtyPlugin) afterWrite(db *gorm.DB) {
	if db.Statement == nil {
		return
	}

	// 嘗試從 Statement 直接讀取 table 名（適用於 Updates/UpdateColumn 帶 map 時）
	tableName := db.Statement.Table

	// 若 Statement.Table 為空，回退到 Schema
	if tableName == "" && db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	}

	if tableName == "" {
		return
	}

	// 統一去掉 ay_ 前綴
	tableName = strings.TrimPrefix(tableName, "ay_")

	if !MediaReferencingTables[tableName] {
		// 非文件引用表，跳過以避免不必要的重掃
		return
	}

	// 標記媒體緩存為臟，下次讀取時自動重新掃描
	MarkDirty()
}
